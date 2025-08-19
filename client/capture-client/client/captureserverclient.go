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

// CaptureServerClient handles communication with the capture server
type CaptureServerClient interface {
	GetClientSettings(ctx context.Context) (*ClientSettingsResponse, error)
	UploadClip(ctx context.Context, request UploadClipRequest) error
}

// captureServerClient implements ClientService using HTTP
type captureServerClient struct {
	serverURL    string
	clientID     string
	clientSecret string
	httpClient   *http.Client
}

// NewCaptureServerClient creates a new HTTP client service
func NewCaptureServerClient(serverURL, clientID, clientSecret string, timeout time.Duration) CaptureServerClient {
	return &captureServerClient{
		serverURL:    serverURL,
		clientID:     clientID,
		clientSecret: clientSecret,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

// GetClientSettings fetches client settings from the server
func (s *captureServerClient) GetClientSettings(ctx context.Context) (*ClientSettingsResponse, error) {
	url := fmt.Sprintf("%s/api/client/settings", s.serverURL)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add Basic Auth header
	auth := base64.StdEncoding.EncodeToString([]byte(s.clientID + ":" + s.clientSecret))
	req.Header.Set("Authorization", "Basic "+auth)

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
	auth := base64.StdEncoding.EncodeToString([]byte(s.clientID + ":" + s.clientSecret))
	req.Header.Set("Authorization", "Basic "+auth)
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
