package e2e

import (
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/yeti47/cryospy/server/core/ccc/logging"
	"github.com/yeti47/cryospy/server/core/clients"
	"github.com/yeti47/cryospy/server/core/config"
	"github.com/yeti47/cryospy/server/core/encryption"
)

// staticMekStore implements encryption.MekStore for testing
type staticMekStore struct {
	mek []byte
}

func (s *staticMekStore) GetMek() ([]byte, error) {
	if len(s.mek) == 0 {
		return nil, fmt.Errorf("mek not set")
	}
	return s.mek, nil
}

func (s *staticMekStore) SetMek(mek []byte) error {
	s.mek = mek
	return nil
}

func (s *staticMekStore) ClearMek() error {
	s.mek = nil
	return nil
}

type clientConfig struct {
	ClientID             string `json:"client_id"`
	ClientSecret         string `json:"client_secret"`
	ServerURL            string `json:"server_url"`
	CameraDevice         string `json:"camera_device"`
	BufferSize           int    `json:"buffer_size"`
	SettingsSyncSeconds  int    `json:"settings_sync_seconds"`
	ServerTimeoutSeconds int    `json:"server_timeout_seconds"`
	ProxyAuthHeader      string `json:"proxy_auth_header"`
	ProxyAuthValue       string `json:"proxy_auth_value"`
}

func TestE2E(t *testing.T) {
	// Ensure we are in the correct directory or can find the docker-compose file
	// Tests run in the directory of the test file.

	testDataDir := "testdata"
	serverDataDir := filepath.Join(testDataDir, "server-data")
	clientConfigDir := filepath.Join(testDataDir, "client-config")

	// Cleanup function
	cleanup := func() {
		// Stop containers
		cmd := exec.Command("docker", "compose", "down")
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Logf("Failed to stop containers: %v\nOutput: %s", err, output)
		}
		// Remove test data
		os.RemoveAll(testDataDir)
	}

	// Clean up previous runs
	cleanup()
	// Defer cleanup for this run
	defer cleanup()

	// 1. Setup Environment (Directories, DB, Configs)
	t.Log("Setting up test environment...")
	if err := setupEnvironment(serverDataDir, clientConfigDir); err != nil {
		t.Fatalf("Failed to setup environment: %v", err)
	}

	// 2. Generate Test Video
	t.Log("Generating test video...")
	videoPath := filepath.Join(testDataDir, "test.mp4")
	if err := generateTestVideo(videoPath); err != nil {
		t.Fatalf("Failed to generate test video: %v", err)
	}

	// 3. Sync filesystem to ensure all files are flushed before Docker starts
	t.Log("Syncing filesystem...")
	syncCmd := exec.Command("sync")
	if err := syncCmd.Run(); err != nil {
		t.Logf("Warning: sync command failed (may be unavailable): %v", err)
	}

	// Verify the client config exists before starting Docker
	clientConfigPath := filepath.Join(clientConfigDir, "config.json")
	if _, err := os.Stat(clientConfigPath); os.IsNotExist(err) {
		t.Fatalf("Client config file does not exist at %s before Docker starts", clientConfigPath)
	}
	t.Logf("Client config verified at %s", clientConfigPath)

	// 4. Run Docker Compose
	t.Log("Starting Docker Compose...")
	cmd := exec.Command("docker", "compose", "up", "--build", "--abort-on-container-exit")
	// Stream output to stdout so we can see it in test logs
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		t.Fatalf("Failed to start Docker Compose: %v", err)
	}

	// Wait for server to be ready
	t.Log("Waiting for server to be ready...")
	for i := range 60 {
		resp, err := http.Get("http://localhost:8080/auth/login")
		if err == nil && resp.StatusCode == 200 {
			resp.Body.Close()
			t.Log("Server is ready")
			break
		}
		if resp != nil {
			resp.Body.Close()
		}
		time.Sleep(1 * time.Second)
		if i >= 59 {
			t.Fatal("Server did not become ready within 60 seconds")
		}
	}

	// Wait for the video duration (40s) plus a buffer, then stop the client
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case err := <-done:
		if err != nil {
			// If it exits early with an error, fail the test
			t.Fatalf("Docker Compose exited unexpectedly: %v", err)
		}
	case <-time.After(50 * time.Second):
		// Verify that clips were uploaded and are visible in the dashboard
		t.Log("Verifying clips in dashboard...")
		if err := verifyClips(t); err != nil {
			t.Errorf("Verification failed: %v", err)
		}

		t.Log("Test duration reached. Stopping Docker Compose...")
		if err := cmd.Process.Signal(os.Interrupt); err != nil {
			t.Fatalf("Failed to send interrupt signal: %v", err)
		}

		// Wait for graceful shutdown
		select {
		case err := <-done:
			// Expect clean exit or check error if necessary
			if err != nil {
				t.Logf("Docker Compose exited with error after signal (expected): %v", err)
			}
		case <-time.After(30 * time.Second):
			t.Fatal("Docker Compose failed to stop gracefully")
			cmd.Process.Kill()
		}
	}
}

func setupEnvironment(serverDataDir, clientConfigDir string) error {
	if err := os.MkdirAll(serverDataDir, 0o755); err != nil {
		return fmt.Errorf("failed to create server data dir: %w", err)
	}

	if err := os.MkdirAll(clientConfigDir, 0o755); err != nil {
		return fmt.Errorf("failed to create client config dir: %w", err)
	}

	dbPath := filepath.Join(serverDataDir, "cryospy.db")

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return fmt.Errorf("failed to open sqlite database: %w", err)
	}
	defer db.Close()

	encryptor := encryption.NewAESEncryptor()
	mekRepo, err := encryption.NewSQLiteMekRepository(db)
	if err != nil {
		return fmt.Errorf("failed to create mek repo: %w", err)
	}
	mekService := encryption.NewMekService(logging.NopLogger, mekRepo, encryptor)

	mekPassword := "test-password"
	mek, err := mekService.CreateMek(mekPassword)
	if err != nil {
		return fmt.Errorf("failed to create mek: %w", err)
	}

	decryptedMek, err := encryption.DecryptMek(mek, mekPassword, encryptor)
	if err != nil {
		return fmt.Errorf("failed to decrypt mek: %w", err)
	}

	clientRepo, err := clients.NewSQLiteClientRepository(db)
	if err != nil {
		return fmt.Errorf("failed to create client repo: %w", err)
	}
	clientService := clients.NewClientService(logging.NopLogger, clientRepo, encryptor)

	mekStore := &staticMekStore{mek: decryptedMek}

	clientID := "test-client"
	req := clients.CreateClientRequest{
		ID:                    clientID,
		StorageLimitMegabytes: 1024,
		ClipDurationSeconds:   30,
		MotionOnly:            false,
		Grayscale:             false,
		OutputFormat:          "mp4",
		OutputCodec:           "libopenh264",
		VideoBitRate:          "1000k",
		MotionMinArea:         1000,
		MotionMaxFrames:       300,
		MotionWarmUpFrames:    30,
		MotionMinWidth:        20,
		MotionMinHeight:       20,
		MotionMinAspect:       0.3,
		MotionMaxAspect:       3.0,
		MotionMogHistory:      500,
		MotionMogVarThresh:    16.0,
		CaptureCodec:          "MJPG",
		CaptureFrameRate:      15.0,
	}

	clientEntity, secret, err := clientService.CreateClient(req, mekStore)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	secretHex := hex.EncodeToString(secret)

	// Client Config
	cfg := clientConfig{
		ClientID:             clientEntity.ID,
		ClientSecret:         secretHex,
		ServerURL:            "http://server:8081", // Docker service name
		CameraDevice:         "0",
		BufferSize:           5,
		SettingsSyncSeconds:  60,
		ServerTimeoutSeconds: 30,
	}

	configPath := filepath.Join(clientConfigDir, "config.json")
	cfgBytes, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal client config: %w", err)
	}
	if err := os.WriteFile(configPath, cfgBytes, 0o644); err != nil {
		return fmt.Errorf("failed to write client config: %w", err)
	}

	// Verify the config file was written successfully
	if _, err := os.Stat(configPath); err != nil {
		return fmt.Errorf("config file not created: %w", err)
	}

	// Server Config
	serverConfig := &config.Config{
		WebAddr:     "0.0.0.0",
		WebPort:     8080,
		CapturePort: 8081,
		// These paths are inside the container
		DatabasePath: "/home/cryospy/cryospy/cryospy.db",
		LogPath:      "/home/cryospy/cryospy/logs",
		LogLevel:     "info",
	}

	if err := os.MkdirAll(filepath.Join(serverDataDir, "logs"), 0o755); err != nil {
		return fmt.Errorf("failed to create log directory: %w", err)
	}

	serverConfigPath := filepath.Join(serverDataDir, "config.json")
	if err := serverConfig.SaveConfig(serverConfigPath); err != nil {
		return fmt.Errorf("failed to save server config: %w", err)
	}

	return nil
}

func generateTestVideo(path string) error {
	// Check if ffmpeg is available
	if _, err := exec.LookPath("ffmpeg"); err == nil {
		cmd := exec.Command("ffmpeg", "-y", "-f", "lavfi", "-i", "testsrc=duration=40:size=640x480:rate=15", "-c:v", "libopenh264", "-pix_fmt", "yuv420p", path)
		return cmd.Run()
	}

	// Fallback to docker
	log.Println("ffmpeg not found, using docker to generate video")
	absPath, err := filepath.Abs(filepath.Dir(path))
	if err != nil {
		return err
	}
	filename := filepath.Base(path)

	cmd := exec.Command("docker", "run", "--rm", "-v", fmt.Sprintf("%s:/out", absPath), "jrottenberg/ffmpeg", "-y", "-f", "lavfi", "-i", "testsrc=duration=40:size=640x480:rate=15", "-c:v", "libx264", "-pix_fmt", "yuv420p", fmt.Sprintf("/out/%s", filename))
	return cmd.Run()
}

func verifyClips(t *testing.T) error {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return fmt.Errorf("failed to create cookie jar: %w", err)
	}

	client := &http.Client{
		Jar:     jar,
		Timeout: 10 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	// 1. Login
	loginURL := "http://localhost:8080/auth/login"
	data := url.Values{}
	data.Set("password", "test-password")

	resp, err := client.PostForm(loginURL, data)
	if err != nil {
		return fmt.Errorf("failed to login: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusFound && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("login failed with status: %d", resp.StatusCode)
	}

	// Check for cookies
	cookies := jar.Cookies(resp.Request.URL)
	if len(cookies) == 0 {
		// Try getting cookies from response directly if jar didn't catch them (though it should have)
		cookies = resp.Cookies()
		if len(cookies) == 0 {
			return fmt.Errorf("no cookies received after login")
		}
	}

	// 2. Get Clips
	clipsURL := "http://localhost:8080/clips"
	req, err := http.NewRequest("GET", clipsURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Manually add cookies because cookiejar filters Secure cookies on http://localhost
	// and the server sets Secure; SameSite=None by default
	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}

	resp, err = client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to get clips: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Read body for debugging
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("get clips failed with status: %d, body: %s", resp.StatusCode, string(bodyBytes))
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}
	bodyString := string(bodyBytes)

	// 3. Check for clips
	// We look for "clip_" which is the prefix for clip filenames
	if !strings.Contains(bodyString, "clip_") {
		return fmt.Errorf("no clips found in dashboard response")
	}

	t.Log("Successfully verified clips in dashboard")
	return nil
}
