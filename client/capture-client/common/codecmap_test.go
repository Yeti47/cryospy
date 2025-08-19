package common

import (
	"testing"
)

func TestGetAvailableCodecs(t *testing.T) {
	// Create a new codec provider
	provider := NewFFmpegCodecProvider()

	// Get available codecs
	codecs := provider.GetAvailableCodecs()

	// Test that we get a map back
	if codecs == nil {
		t.Fatal("GetAvailableCodecs() returned nil")
	}

	// Test that modifying the returned map doesn't affect the internal cache
	originalLen := len(codecs)
	codecs["test_codec"] = true

	// Get codecs again and verify the internal state wasn't modified
	codecs2 := provider.GetAvailableCodecs()
	if len(codecs2) != originalLen {
		t.Errorf("Internal codec cache was modified. Expected length %d, got %d", originalLen, len(codecs2))
	}

	// Verify the test codec we added isn't in the fresh copy
	if _, exists := codecs2["test_codec"]; exists {
		t.Error("Internal codec cache was modified - test_codec should not exist in fresh copy")
	}
}

func TestGetAvailableCodecs_ReturnsCopy(t *testing.T) {
	provider := NewFFmpegCodecProvider()

	// Get two separate copies
	codecs1 := provider.GetAvailableCodecs()
	codecs2 := provider.GetAvailableCodecs()

	// Modify one copy
	codecs1["modification_test"] = true

	// Verify the other copy is unaffected
	if _, exists := codecs2["modification_test"]; exists {
		t.Error("GetAvailableCodecs() doesn't return independent copies")
	}
}

func TestFFmpegCodecProvider_IsCodecAvailable(t *testing.T) {
	provider := NewFFmpegCodecProvider()

	// Test with a codec that definitely doesn't exist
	if provider.IsCodecAvailable("definitely_nonexistent_codec_12345") {
		t.Error("IsCodecAvailable should return false for non-existent codec")
	}

	// Get all available codecs for verification
	codecs := provider.GetAvailableCodecs()
	t.Logf("Found %d available codecs", len(codecs))

	// Log all available codecs
	t.Logf("All available codecs:")
	for codecName, available := range codecs {
		if available {
			t.Logf("  - %s", codecName)
		}
	}

	// Also log key codecs we're specifically interested in (whether available or not)
	keyCodecs := []string{"libx264", "libopenh264", "h264_vaapi", "h264_qsv", "h264_v4l2m2m", "libx265"}
	t.Logf("Key H.264/H.265 codec availability:")
	for _, codec := range keyCodecs {
		available := provider.IsCodecAvailable(codec)
		t.Logf("  - %s: %t", codec, available)
	}

	// Test that IsCodecAvailable is consistent with GetAvailableCodecs
	for codecName, isAvailable := range codecs {
		if provider.IsCodecAvailable(codecName) != isAvailable {
			t.Errorf("IsCodecAvailable('%s') inconsistent with GetAvailableCodecs result", codecName)
		}
	}
}

func TestGetFallbackCodec_WithAvailableCodec(t *testing.T) {
	provider := NewFFmpegCodecProvider()

	// Get available codecs to test with
	availableCodecs := provider.GetAvailableCodecs()

	// Find any available codec to test with
	var testCodec string
	for codec, available := range availableCodecs {
		if available {
			testCodec = codec
			break
		}
	}

	if testCodec == "" {
		t.Skip("No available codecs found for testing")
	}

	// Test that requesting an available codec returns itself
	result, err := provider.GetFallbackCodec(testCodec)
	if err != nil {
		t.Errorf("GetFallbackCodec failed for available codec '%s': %v", testCodec, err)
	}

	if result != testCodec {
		t.Errorf("Expected GetFallbackCodec to return '%s', got '%s'", testCodec, result)
	}
}

func TestGetFallbackCodec_WithUnavailableCodec(t *testing.T) {
	provider := NewFFmpegCodecProvider()

	// Test with libx264 which is likely unavailable based on the user's issue
	result, err := provider.GetFallbackCodec("libx264")

	if err != nil {
		// This is expected if no fallback is available
		t.Logf("No fallback available for libx264: %v", err)
		return
	}

	// If we got a result, verify it's in the fallback chain
	fallbackChain := CodecFallbackMap["libx264"]
	found := false
	for _, codec := range fallbackChain {
		if codec == result {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("Fallback codec '%s' is not in the defined fallback chain %v", result, fallbackChain)
	}

	// Verify the fallback codec is actually available
	if !provider.IsCodecAvailable(result) {
		t.Errorf("Fallback codec '%s' is not available according to IsCodecAvailable", result)
	}

	t.Logf("Successfully fell back from libx264 to %s", result)
}

func TestGetFallbackCodec_UndefinedCodec(t *testing.T) {
	provider := NewFFmpegCodecProvider()

	// Test with a codec that has no fallback defined
	_, err := provider.GetFallbackCodec("nonexistent_codec_with_no_fallback")

	if err == nil {
		t.Error("Expected error for codec with no fallback defined")
	}

	expectedMsg := "no fallback is defined"
	if !contains(err.Error(), expectedMsg) {
		t.Errorf("Expected error message to contain '%s', got: %s", expectedMsg, err.Error())
	}
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || (len(s) > len(substr) && containsAtPosition(s, substr)))
}

func containsAtPosition(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
