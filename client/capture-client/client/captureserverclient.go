package client

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"time"
)

// ClientAuth holds client authentication credentials
type ClientAuth struct {
	ClientID     string
	ClientSecret string
}

// ProxyAuth holds proxy authentication configuration
type ProxyAuth struct {
	Header string
	Value  string
}

// CaptureServerClient handles communication with the capture server
type CaptureServerClient interface {
	GetClientSettings(ctx context.Context) (*ClientSettingsResponse, error)
	UploadClip(ctx context.Context, request UploadClipRequest) error
}

// captureServerClient implements ClientService using HTTP
type captureServerClient struct {
	serverURL  string
	clientAuth ClientAuth
	proxyAuth  ProxyAuth
	httpClient *http.Client
}

// NewCaptureServerClient creates a new HTTP client service
func NewCaptureServerClient(serverURL string, clientAuth ClientAuth, proxyAuth ProxyAuth, timeout time.Duration) CaptureServerClient {
	return &captureServerClient{
		serverURL:  serverURL,
		clientAuth: clientAuth,
		proxyAuth:  proxyAuth,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

// addAuthHeaders adds both basic auth and proxy auth headers to the request
func (s *captureServerClient) addAuthHeaders(req *http.Request) {
	// Add Basic Auth header
	auth := base64.StdEncoding.EncodeToString([]byte(s.clientAuth.ClientID + ":" + s.clientAuth.ClientSecret))
	req.Header.Set("Authorization", "Basic "+auth)

	// Add proxy auth header if configured
	if s.proxyAuth.Header != "" && s.proxyAuth.Value != "" {
		req.Header.Set(s.proxyAuth.Header, s.proxyAuth.Value)
	}
}

// GetClientSettings fetches client settings from the server
func (s *captureServerClient) GetClientSettings(ctx context.Context) (*ClientSettingsResponse, error) {
	url := fmt.Sprintf("%s/api/client/settings", s.serverURL)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add authentication headers
	s.addAuthHeaders(req)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned status %d", resp.StatusCode)
	}

	var settings ClientSettingsResponse
	if err := json.NewDecoder(resp.Body).Decode(&settings); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &settings, nil
}

// UploadClip uploads a video clip to the server
func (s *captureServerClient) UploadClip(ctx context.Context, request UploadClipRequest) error {
	url := fmt.Sprintf("%s/api/clips", s.serverURL)

	// Create multipart form
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// Add timestamp field - use the actual recording timestamp instead of upload time
	if err := writer.WriteField("timestamp", request.RecordingTimestamp.UTC().Format(time.RFC3339)); err != nil {
		return fmt.Errorf("failed to write timestamp field: %w", err)
	}

	// Add duration field using the known duration from client settings
	durationStr := fmt.Sprintf("%.1f", request.Duration.Seconds())
	if err := writer.WriteField("duration", durationStr); err != nil {
		return fmt.Errorf("failed to write duration field: %w", err)
	}

	// Add has_motion field using the actual motion detection result
	motionStr := "false"
	if request.HasMotion {
		motionStr = "true"
	}
	if err := writer.WriteField("has_motion", motionStr); err != nil {
		return fmt.Errorf("failed to write has_motion field: %w", err)
	}

	// Add file field
	part, err := writer.CreateFormFile("video", "clip.mp4")
	if err != nil {
		return fmt.Errorf("failed to create form file: %w", err)
	}

	if _, err := part.Write(request.VideoData); err != nil {
		return fmt.Errorf("failed to write video data: %w", err)
	}

	if err := writer.Close(); err != nil {
		return fmt.Errorf("failed to close writer: %w", err)
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, "POST", url, &buf)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Add headers
	s.addAuthHeaders(req)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	// Make request
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("server returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}
