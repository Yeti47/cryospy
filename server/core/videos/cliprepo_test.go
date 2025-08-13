package videos

import (
	"context"
	"testing"
	"time"

	"github.com/yeti47/cryospy/server/core/ccc/db"
)

func setupTestRepo(t *testing.T) (*SQLiteClipRepository, func()) {
	testDB, err := db.NewInMemoryDB()
	if err != nil {
		t.Fatalf("Failed to create in-memory database: %v", err)
	}

	repo, err := NewSQLiteClipRepository(testDB)
	if err != nil {
		testDB.Close()
		t.Fatalf("Failed to create repository: %v", err)
	}

	cleanup := func() {
		testDB.Close()
	}

	return repo, cleanup
}

func createTestClip() *Clip {
	now := time.Now().UTC()
	return &Clip{
		ID:                 "test-clip-1",
		ClientID:           "client-123",
		Title:              "Test Video Clip",
		TimeStamp:          now,
		Duration:           time.Duration(30 * time.Second),
		HasMotion:          true,
		EncryptedVideo:     []byte("encrypted-video-data"),
		VideoWidth:         1920,
		VideoHeight:        1080,
		VideoMimeType:      "video/mp4",
		EncryptedThumbnail: []byte("encrypted-thumbnail-data"),
		ThumbnailWidth:     320,
		ThumbnailHeight:    240,
		ThumbnailMimeType:  "image/jpeg",
	}
}

func TestSQLiteClipRepository_Add(t *testing.T) {
	repo, cleanup := setupTestRepo(t)
	defer cleanup()

	ctx := context.Background()
	clip := createTestClip()

	err := repo.Add(ctx, clip)
	if err != nil {
		t.Fatalf("Failed to add clip: %v", err)
	}

	// Verify the clip was added by retrieving it
	retrieved, err := repo.GetByID(ctx, clip.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve clip: %v", err)
	}

	if retrieved == nil {
		t.Fatal("Retrieved clip is nil")
	}

	// Compare all fields
	if retrieved.ID != clip.ID {
		t.Errorf("Expected ID %s, got %s", clip.ID, retrieved.ID)
	}
	if retrieved.ClientID != clip.ClientID {
		t.Errorf("Expected ClientID %s, got %s", clip.ClientID, retrieved.ClientID)
	}
	if retrieved.Title != clip.Title {
		t.Errorf("Expected Title %s, got %s", clip.Title, retrieved.Title)
	}
	if !retrieved.TimeStamp.Equal(clip.TimeStamp) {
		t.Errorf("Expected timestamp %v, got %v", clip.TimeStamp, retrieved.TimeStamp)
	}
	if retrieved.Duration != clip.Duration {
		t.Errorf("Expected duration %v, got %v", clip.Duration, retrieved.Duration)
	}
	if retrieved.HasMotion != clip.HasMotion {
		t.Errorf("Expected HasMotion %v, got %v", clip.HasMotion, retrieved.HasMotion)
	}
	if string(retrieved.EncryptedVideo) != string(clip.EncryptedVideo) {
		t.Errorf("Expected encrypted video %s, got %s", string(clip.EncryptedVideo), string(retrieved.EncryptedVideo))
	}
	if retrieved.VideoWidth != clip.VideoWidth {
		t.Errorf("Expected video width %d, got %d", clip.VideoWidth, retrieved.VideoWidth)
	}
}

func TestSQLiteClipRepository_GetByID_NotFound(t *testing.T) {
	repo, cleanup := setupTestRepo(t)
	defer cleanup()

	ctx := context.Background()

	retrieved, err := repo.GetByID(ctx, "non-existent-id")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if retrieved != nil {
		t.Error("Expected nil for non-existent clip, got a clip")
	}
}

func TestSQLiteClipRepository_Query(t *testing.T) {
	repo, cleanup := setupTestRepo(t)
	defer cleanup()

	ctx := context.Background()

	// Add multiple test clips
	now := time.Now().UTC()
	clips := []*Clip{
		{
			ID:                 "clip-1",
			ClientID:           "client-1",
			Title:              "First Clip",
			TimeStamp:          now.Add(-2 * time.Hour),
			Duration:           time.Duration(30 * time.Second),
			HasMotion:          true,
			EncryptedVideo:     []byte("video-1"),
			VideoWidth:         1920,
			VideoHeight:        1080,
			VideoMimeType:      "video/mp4",
			EncryptedThumbnail: []byte("thumb-1"),
			ThumbnailWidth:     320,
			ThumbnailHeight:    240,
			ThumbnailMimeType:  "image/jpeg",
		},
		{
			ID:                 "clip-2",
			ClientID:           "client-2",
			Title:              "Second Clip",
			TimeStamp:          now.Add(-1 * time.Hour),
			Duration:           time.Duration(45 * time.Second),
			HasMotion:          false,
			EncryptedVideo:     []byte("video-2"),
			VideoWidth:         1280,
			VideoHeight:        720,
			VideoMimeType:      "video/mp4",
			EncryptedThumbnail: []byte("thumb-2"),
			ThumbnailWidth:     256,
			ThumbnailHeight:    144,
			ThumbnailMimeType:  "image/png",
		},
		{
			ID:                 "clip-3",
			ClientID:           "client-1",
			Title:              "Third Clip",
			TimeStamp:          now,
			Duration:           time.Duration(60 * time.Second),
			HasMotion:          true,
			EncryptedVideo:     []byte("video-3"),
			VideoWidth:         1920,
			VideoHeight:        1080,
			VideoMimeType:      "video/mp4",
			EncryptedThumbnail: []byte("thumb-3"),
			ThumbnailWidth:     320,
			ThumbnailHeight:    240,
			ThumbnailMimeType:  "image/jpeg",
		},
	}

	for _, clip := range clips {
		err := repo.Add(ctx, clip)
		if err != nil {
			t.Fatalf("Failed to add clip %s: %v", clip.ID, err)
		}
	}

	// Test query with no filters (should return all clips, ordered by timestamp DESC)
	allClips, totalCount, err := repo.Query(ctx, ClipQuery{})
	if err != nil {
		t.Fatalf("Failed to query all clips: %v", err)
	}

	if len(allClips) != 3 {
		t.Errorf("Expected 3 clips, got %d", len(allClips))
	}

	if totalCount != 3 {
		t.Errorf("Expected total count 3, got %d", totalCount)
	}

	// Verify order (newest first)
	if allClips[0].ID != "clip-3" || allClips[1].ID != "clip-2" || allClips[2].ID != "clip-1" {
		t.Error("Clips not ordered correctly by timestamp DESC")
	}

	// Test query with motion filter
	hasMotion := true
	motionClips, _, err := repo.Query(ctx, ClipQuery{HasMotion: &hasMotion})
	if err != nil {
		t.Fatalf("Failed to query clips with motion: %v", err)
	}

	if len(motionClips) != 2 {
		t.Errorf("Expected 2 clips with motion, got %d", len(motionClips))
	}

	// Test query with time range
	startTime := now.Add(-90 * time.Minute)
	endTime := now.Add(-30 * time.Minute)
	timeRangeClips, timeRangeCount, err := repo.Query(ctx, ClipQuery{StartTime: &startTime, EndTime: &endTime})
	if err != nil {
		t.Fatalf("Failed to query clips in time range: %v", err)
	}

	if len(timeRangeClips) != 1 {
		t.Errorf("Expected 1 clip in time range, got %d", len(timeRangeClips))
	}
	if timeRangeCount != 1 {
		t.Errorf("Expected total count 1 for time range, got %d", timeRangeCount)
	}
	if timeRangeClips[0].ID != "clip-2" {
		t.Errorf("Expected clip-2 in time range, got %s", timeRangeClips[0].ID)
	}

	// Test query with ClientID filter
	client1Clips, client1Count, err := repo.Query(ctx, ClipQuery{ClientID: "client-1"})
	if err != nil {
		t.Fatalf("Failed to query clips for client-1: %v", err)
	}

	if len(client1Clips) != 2 {
		t.Errorf("Expected 2 clips for client-1, got %d", len(client1Clips))
	}
	if client1Count != 2 {
		t.Errorf("Expected total count 2 for client-1, got %d", client1Count)
	}
	// Should be ordered by timestamp DESC: clip-3, clip-1
	if client1Clips[0].ID != "clip-3" || client1Clips[1].ID != "clip-1" {
		t.Error("Client-1 clips not ordered correctly")
	}
}

func TestSQLiteClipRepository_QueryInfo(t *testing.T) {
	repo, cleanup := setupTestRepo(t)
	defer cleanup()

	ctx := context.Background()
	clip := createTestClip()

	err := repo.Add(ctx, clip)
	if err != nil {
		t.Fatalf("Failed to add clip: %v", err)
	}

	clipInfos, totalCount, err := repo.QueryInfo(ctx, ClipQuery{})
	if err != nil {
		t.Fatalf("Failed to query clip info: %v", err)
	}

	if len(clipInfos) != 1 {
		t.Errorf("Expected 1 clip info, got %d", len(clipInfos))
	}

	if totalCount != 1 {
		t.Errorf("Expected total count 1, got %d", totalCount)
	}

	clipInfo := clipInfos[0]
	if clipInfo.ID != clip.ID {
		t.Errorf("Expected ID %s, got %s", clip.ID, clipInfo.ID)
	}
	if clipInfo.ClientID != clip.ClientID {
		t.Errorf("Expected ClientID %s, got %s", clip.ClientID, clipInfo.ClientID)
	}
	if clipInfo.Title != clip.Title {
		t.Errorf("Expected Title %s, got %s", clip.Title, clipInfo.Title)
	}
	if !clipInfo.TimeStamp.Equal(clip.TimeStamp) {
		t.Errorf("Expected timestamp %v, got %v", clip.TimeStamp, clipInfo.TimeStamp)
	}
	if clipInfo.HasMotion != clip.HasMotion {
		t.Errorf("Expected HasMotion %v, got %v", clip.HasMotion, clipInfo.HasMotion)
	}
	// ClipInfo should not contain video data - we can't test this directly
	// but the struct definition ensures it
}

func TestSQLiteClipRepository_GetThumbnailByID(t *testing.T) {
	repo, cleanup := setupTestRepo(t)
	defer cleanup()

	ctx := context.Background()
	clip := createTestClip()

	err := repo.Add(ctx, clip)
	if err != nil {
		t.Fatalf("Failed to add clip: %v", err)
	}

	thumbnail, err := repo.GetThumbnailByID(ctx, clip.ID)
	if err != nil {
		t.Fatalf("Failed to get thumbnail: %v", err)
	}

	if thumbnail == nil {
		t.Fatal("Retrieved thumbnail is nil")
	}

	if string(thumbnail.Data) != string(clip.EncryptedThumbnail) {
		t.Errorf("Expected thumbnail data %s, got %s", string(clip.EncryptedThumbnail), string(thumbnail.Data))
	}
	if thumbnail.Width != clip.ThumbnailWidth {
		t.Errorf("Expected thumbnail width %d, got %d", clip.ThumbnailWidth, thumbnail.Width)
	}
	if thumbnail.Height != clip.ThumbnailHeight {
		t.Errorf("Expected thumbnail height %d, got %d", clip.ThumbnailHeight, thumbnail.Height)
	}
	if thumbnail.MimeType != clip.ThumbnailMimeType {
		t.Errorf("Expected thumbnail mime type %s, got %s", clip.ThumbnailMimeType, thumbnail.MimeType)
	}
}

func TestSQLiteClipRepository_GetThumbnailByID_NotFound(t *testing.T) {
	repo, cleanup := setupTestRepo(t)
	defer cleanup()

	ctx := context.Background()

	thumbnail, err := repo.GetThumbnailByID(ctx, "non-existent-id")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if thumbnail != nil {
		t.Error("Expected nil for non-existent thumbnail, got a thumbnail")
	}
}

func TestSQLiteClipRepository_Delete(t *testing.T) {
	repo, cleanup := setupTestRepo(t)
	defer cleanup()

	ctx := context.Background()
	clip := createTestClip()

	// Add the clip
	err := repo.Add(ctx, clip)
	if err != nil {
		t.Fatalf("Failed to add clip: %v", err)
	}

	// Verify it exists
	retrieved, err := repo.GetByID(ctx, clip.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve clip: %v", err)
	}
	if retrieved == nil {
		t.Fatal("Clip not found after adding")
	}

	// Delete it
	err = repo.Delete(ctx, clip.ID)
	if err != nil {
		t.Fatalf("Failed to delete clip: %v", err)
	}

	// Verify it's gone
	retrieved, err = repo.GetByID(ctx, clip.ID)
	if err != nil {
		t.Fatalf("Unexpected error after deletion: %v", err)
	}
	if retrieved != nil {
		t.Error("Clip still exists after deletion")
	}
}

func TestSQLiteClipRepository_BooleanConversion(t *testing.T) {
	repo, cleanup := setupTestRepo(t)
	defer cleanup()

	ctx := context.Background()

	// Test with HasMotion = false
	clipNoMotion := createTestClip()
	clipNoMotion.ID = "no-motion-clip"
	clipNoMotion.ClientID = "client-no-motion"
	clipNoMotion.Title = "Clip Without Motion"
	clipNoMotion.HasMotion = false

	err := repo.Add(ctx, clipNoMotion)
	if err != nil {
		t.Fatalf("Failed to add clip without motion: %v", err)
	}

	retrieved, err := repo.GetByID(ctx, clipNoMotion.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve clip: %v", err)
	}

	if retrieved.HasMotion != false {
		t.Errorf("Expected HasMotion false, got %v", retrieved.HasMotion)
	}

	// Test querying for clips without motion
	hasMotion := false
	noMotionClips, _, err := repo.Query(ctx, ClipQuery{HasMotion: &hasMotion})
	if err != nil {
		t.Fatalf("Failed to query clips without motion: %v", err)
	}

	if len(noMotionClips) != 1 {
		t.Errorf("Expected 1 clip without motion, got %d", len(noMotionClips))
	}
}

func TestSQLiteClipRepository_ClientIDFiltering(t *testing.T) {
	repo, cleanup := setupTestRepo(t)
	defer cleanup()

	ctx := context.Background()

	// Add clips for different clients
	clips := []*Clip{
		{
			ID:                 "client-a-clip-1",
			ClientID:           "client-a",
			Title:              "Client A Clip 1",
			TimeStamp:          time.Now().UTC().Add(-1 * time.Hour),
			Duration:           time.Duration(30 * time.Second),
			HasMotion:          true,
			EncryptedVideo:     []byte("video-a1"),
			VideoWidth:         1920,
			VideoHeight:        1080,
			VideoMimeType:      "video/mp4",
			EncryptedThumbnail: []byte("thumb-a1"),
			ThumbnailWidth:     320,
			ThumbnailHeight:    240,
			ThumbnailMimeType:  "image/jpeg",
		},
		{
			ID:                 "client-a-clip-2",
			ClientID:           "client-a",
			Title:              "Client A Clip 2",
			TimeStamp:          time.Now().UTC(),
			Duration:           time.Duration(45 * time.Second),
			HasMotion:          false,
			EncryptedVideo:     []byte("video-a2"),
			VideoWidth:         1280,
			VideoHeight:        720,
			VideoMimeType:      "video/mp4",
			EncryptedThumbnail: []byte("thumb-a2"),
			ThumbnailWidth:     256,
			ThumbnailHeight:    144,
			ThumbnailMimeType:  "image/png",
		},
		{
			ID:                 "client-b-clip-1",
			ClientID:           "client-b",
			Title:              "Client B Clip 1",
			TimeStamp:          time.Now().UTC().Add(-30 * time.Minute),
			Duration:           time.Duration(60 * time.Second),
			HasMotion:          true,
			EncryptedVideo:     []byte("video-b1"),
			VideoWidth:         1920,
			VideoHeight:        1080,
			VideoMimeType:      "video/mp4",
			EncryptedThumbnail: []byte("thumb-b1"),
			ThumbnailWidth:     320,
			ThumbnailHeight:    240,
			ThumbnailMimeType:  "image/jpeg",
		},
	}

	for _, clip := range clips {
		err := repo.Add(ctx, clip)
		if err != nil {
			t.Fatalf("Failed to add clip %s: %v", clip.ID, err)
		}
	}

	// Test filtering by client-a
	clientAClips, _, err := repo.Query(ctx, ClipQuery{ClientID: "client-a"})
	if err != nil {
		t.Fatalf("Failed to query clips for client-a: %v", err)
	}

	if len(clientAClips) != 2 {
		t.Errorf("Expected 2 clips for client-a, got %d", len(clientAClips))
	}

	// Verify all clips belong to client-a
	for _, clip := range clientAClips {
		if clip.ClientID != "client-a" {
			t.Errorf("Expected ClientID 'client-a', got '%s'", clip.ClientID)
		}
	}

	// Test filtering by client-b
	clientBClips, _, err := repo.Query(ctx, ClipQuery{ClientID: "client-b"})
	if err != nil {
		t.Fatalf("Failed to query clips for client-b: %v", err)
	}

	if len(clientBClips) != 1 {
		t.Errorf("Expected 1 clip for client-b, got %d", len(clientBClips))
	}

	if clientBClips[0].ClientID != "client-b" {
		t.Errorf("Expected ClientID 'client-b', got '%s'", clientBClips[0].ClientID)
	}

	// Test filtering by non-existent client
	noClips, noClipsCount, err := repo.Query(ctx, ClipQuery{ClientID: "non-existent-client"})
	if err != nil {
		t.Fatalf("Failed to query clips for non-existent client: %v", err)
	}

	if len(noClips) != 0 {
		t.Errorf("Expected 0 clips for non-existent client, got %d", len(noClips))
	}

	if noClipsCount != 0 {
		t.Errorf("Expected total count 0 for non-existent client, got %d", noClipsCount)
	}

	// Test QueryInfo with ClientID filter
	clientAInfos, clientAInfosCount, err := repo.QueryInfo(ctx, ClipQuery{ClientID: "client-a"})
	if err != nil {
		t.Fatalf("Failed to query clip infos for client-a: %v", err)
	}

	if len(clientAInfos) != 2 {
		t.Errorf("Expected 2 clip infos for client-a, got %d", len(clientAInfos))
	}

	if clientAInfosCount != 2 {
		t.Errorf("Expected total count 2 for client-a infos, got %d", clientAInfosCount)
	}

	for _, info := range clientAInfos {
		if info.ClientID != "client-a" {
			t.Errorf("Expected ClientID 'client-a' in info, got '%s'", info.ClientID)
		}
	}
}

func TestSQLiteClipRepository_Pagination(t *testing.T) {
	repo, cleanup := setupTestRepo(t)
	defer cleanup()

	ctx := context.Background()

	// Add multiple test clips to test pagination
	now := time.Now().UTC()
	clips := []*Clip{
		{
			ID:                 "clip-1",
			ClientID:           "client-1",
			Title:              "First Clip",
			TimeStamp:          now.Add(-4 * time.Hour),
			Duration:           time.Duration(30 * time.Second),
			HasMotion:          true,
			EncryptedVideo:     []byte("video-1"),
			VideoWidth:         1920,
			VideoHeight:        1080,
			VideoMimeType:      "video/mp4",
			EncryptedThumbnail: []byte("thumb-1"),
			ThumbnailWidth:     320,
			ThumbnailHeight:    240,
			ThumbnailMimeType:  "image/jpeg",
		},
		{
			ID:                 "clip-2",
			ClientID:           "client-1",
			Title:              "Second Clip",
			TimeStamp:          now.Add(-3 * time.Hour),
			Duration:           time.Duration(45 * time.Second),
			HasMotion:          false,
			EncryptedVideo:     []byte("video-2"),
			VideoWidth:         1280,
			VideoHeight:        720,
			VideoMimeType:      "video/mp4",
			EncryptedThumbnail: []byte("thumb-2"),
			ThumbnailWidth:     256,
			ThumbnailHeight:    144,
			ThumbnailMimeType:  "image/png",
		},
		{
			ID:                 "clip-3",
			ClientID:           "client-1",
			Title:              "Third Clip",
			TimeStamp:          now.Add(-2 * time.Hour),
			Duration:           time.Duration(60 * time.Second),
			HasMotion:          true,
			EncryptedVideo:     []byte("video-3"),
			VideoWidth:         1920,
			VideoHeight:        1080,
			VideoMimeType:      "video/mp4",
			EncryptedThumbnail: []byte("thumb-3"),
			ThumbnailWidth:     320,
			ThumbnailHeight:    240,
			ThumbnailMimeType:  "image/jpeg",
		},
		{
			ID:                 "clip-4",
			ClientID:           "client-1",
			Title:              "Fourth Clip",
			TimeStamp:          now.Add(-1 * time.Hour),
			Duration:           time.Duration(50 * time.Second),
			HasMotion:          true,
			EncryptedVideo:     []byte("video-4"),
			VideoWidth:         1920,
			VideoHeight:        1080,
			VideoMimeType:      "video/mp4",
			EncryptedThumbnail: []byte("thumb-4"),
			ThumbnailWidth:     320,
			ThumbnailHeight:    240,
			ThumbnailMimeType:  "image/jpeg",
		},
		{
			ID:                 "clip-5",
			ClientID:           "client-1",
			Title:              "Fifth Clip",
			TimeStamp:          now,
			Duration:           time.Duration(40 * time.Second),
			HasMotion:          false,
			EncryptedVideo:     []byte("video-5"),
			VideoWidth:         1920,
			VideoHeight:        1080,
			VideoMimeType:      "video/mp4",
			EncryptedThumbnail: []byte("thumb-5"),
			ThumbnailWidth:     320,
			ThumbnailHeight:    240,
			ThumbnailMimeType:  "image/jpeg",
		},
	}

	for _, clip := range clips {
		err := repo.Add(ctx, clip)
		if err != nil {
			t.Fatalf("Failed to add clip %s: %v", clip.ID, err)
		}
	}

	// Test pagination: Page 1, PageSize 3
	paginatedClips, totalCount, err := repo.Query(ctx, ClipQuery{Page: 1, PageSize: 3})
	if err != nil {
		t.Fatalf("Failed to query with pagination (page 1): %v", err)
	}

	if len(paginatedClips) != 3 {
		t.Errorf("Expected 3 clips on page 1, got %d", len(paginatedClips))
	}

	if totalCount != 5 {
		t.Errorf("Expected total count 5, got %d", totalCount)
	}

	// Verify order (newest first) - should be clip-5, clip-4, clip-3
	if paginatedClips[0].ID != "clip-5" || paginatedClips[1].ID != "clip-4" || paginatedClips[2].ID != "clip-3" {
		t.Error("Clips not ordered correctly on page 1")
	}

	// Test pagination: Page 2, PageSize 3
	paginatedClips2, totalCount2, err := repo.Query(ctx, ClipQuery{Page: 2, PageSize: 3})
	if err != nil {
		t.Fatalf("Failed to query with pagination (page 2): %v", err)
	}

	if len(paginatedClips2) != 2 {
		t.Errorf("Expected 2 clips on page 2, got %d", len(paginatedClips2))
	}

	if totalCount2 != 5 {
		t.Errorf("Expected total count 5 on page 2, got %d", totalCount2)
	}

	// Should be clip-2, clip-1
	if paginatedClips2[0].ID != "clip-2" || paginatedClips2[1].ID != "clip-1" {
		t.Error("Clips not ordered correctly on page 2")
	}

	// Test pagination with a page that should be empty
	paginatedClips3, _, err := repo.Query(ctx, ClipQuery{Page: 3, PageSize: 3})
	if err != nil {
		t.Fatalf("Failed to query with pagination (page 3): %v", err)
	}

	if len(paginatedClips3) != 0 {
		t.Errorf("Expected 0 clips on page 3, got %d", len(paginatedClips3))
	}

	// Test QueryInfo with pagination
	clipInfos, infoTotalCount, err := repo.QueryInfo(ctx, ClipQuery{Page: 1, PageSize: 2})
	if err != nil {
		t.Fatalf("Failed to query clip info with pagination: %v", err)
	}

	if len(clipInfos) != 2 {
		t.Errorf("Expected 2 clip infos with limit, got %d", len(clipInfos))
	}

	if infoTotalCount != 5 {
		t.Errorf("Expected total count 5 for clip infos, got %d", infoTotalCount)
	}

	// Should be clip-5, clip-4
	if clipInfos[0].ID != "clip-5" || clipInfos[1].ID != "clip-4" {
		t.Error("Clip infos not ordered correctly with pagination")
	}

	// Test that pagination works with filters
	hasMotion := true
	motionClips, motionTotalCount, err := repo.Query(ctx, ClipQuery{HasMotion: &hasMotion, Page: 1, PageSize: 2})
	if err != nil {
		t.Fatalf("Failed to query motion clips with pagination: %v", err)
	}

	if len(motionClips) != 2 {
		t.Errorf("Expected 2 motion clips with limit, got %d", len(motionClips))
	}

	if motionTotalCount != 3 {
		t.Errorf("Expected total count 3 for motion clips, got %d", motionTotalCount)
	}

	// Motion clips are clip-1, clip-3, clip-4. Ordered by time: clip-4, clip-3, clip-1
	// Page 1, size 2 should be clip-4, clip-3
	if motionClips[0].ID != "clip-4" || motionClips[1].ID != "clip-3" {
		t.Error("Motion clips not ordered correctly with pagination")
	}
}

func TestSQLiteClipRepository_GetTotalStorageUsage(t *testing.T) {
	repo, cleanup := setupTestRepo(t)
	defer cleanup()

	ctx := context.Background()

	// Test with no clips (should return 0)
	totalUsage, err := repo.GetTotalStorageUsage(ctx, "client-empty")
	if err != nil {
		t.Fatalf("Failed to get storage usage for empty client: %v", err)
	}

	if totalUsage != 0 {
		t.Errorf("Expected 0 bytes for empty client, got %d", totalUsage)
	}

	// Add clips with different sizes for multiple clients
	now := time.Now().UTC()
	clips := []*Clip{
		{
			ID:                 "client-a-clip-1",
			ClientID:           "client-a",
			Title:              "Client A Clip 1",
			TimeStamp:          now.Add(-2 * time.Hour),
			Duration:           time.Duration(30 * time.Second),
			HasMotion:          true,
			EncryptedVideo:     make([]byte, 1024*1024), // 1MB
			VideoWidth:         1920,
			VideoHeight:        1080,
			VideoMimeType:      "video/mp4",
			EncryptedThumbnail: []byte("thumb-a1"),
			ThumbnailWidth:     320,
			ThumbnailHeight:    240,
			ThumbnailMimeType:  "image/jpeg",
		},
		{
			ID:                 "client-a-clip-2",
			ClientID:           "client-a",
			Title:              "Client A Clip 2",
			TimeStamp:          now.Add(-1 * time.Hour),
			Duration:           time.Duration(45 * time.Second),
			HasMotion:          false,
			EncryptedVideo:     make([]byte, 2*1024*1024), // 2MB
			VideoWidth:         1280,
			VideoHeight:        720,
			VideoMimeType:      "video/mp4",
			EncryptedThumbnail: []byte("thumb-a2"),
			ThumbnailWidth:     256,
			ThumbnailHeight:    144,
			ThumbnailMimeType:  "image/png",
		},
		{
			ID:                 "client-b-clip-1",
			ClientID:           "client-b",
			Title:              "Client B Clip 1",
			TimeStamp:          now,
			Duration:           time.Duration(60 * time.Second),
			HasMotion:          true,
			EncryptedVideo:     make([]byte, 3*1024*1024), // 3MB
			VideoWidth:         1920,
			VideoHeight:        1080,
			VideoMimeType:      "video/mp4",
			EncryptedThumbnail: []byte("thumb-b1"),
			ThumbnailWidth:     320,
			ThumbnailHeight:    240,
			ThumbnailMimeType:  "image/jpeg",
		},
	}

	// Fill video data with different patterns to ensure they're different sizes
	for i, clip := range clips {
		for j := range clip.EncryptedVideo {
			clip.EncryptedVideo[j] = byte((i + j) % 256)
		}
	}

	for _, clip := range clips {
		err := repo.Add(ctx, clip)
		if err != nil {
			t.Fatalf("Failed to add clip %s: %v", clip.ID, err)
		}
	}

	// Test client-a usage (1MB + 2MB = 3MB)
	clientAUsage, err := repo.GetTotalStorageUsage(ctx, "client-a")
	if err != nil {
		t.Fatalf("Failed to get storage usage for client-a: %v", err)
	}

	expectedClientAUsage := int64(3 * 1024 * 1024) // 3MB
	if clientAUsage != expectedClientAUsage {
		t.Errorf("Expected %d bytes for client-a, got %d", expectedClientAUsage, clientAUsage)
	}

	// Test client-b usage (3MB)
	clientBUsage, err := repo.GetTotalStorageUsage(ctx, "client-b")
	if err != nil {
		t.Fatalf("Failed to get storage usage for client-b: %v", err)
	}

	expectedClientBUsage := int64(3 * 1024 * 1024) // 3MB
	if clientBUsage != expectedClientBUsage {
		t.Errorf("Expected %d bytes for client-b, got %d", expectedClientBUsage, clientBUsage)
	}

	// Test non-existent client (should return 0)
	nonExistentUsage, err := repo.GetTotalStorageUsage(ctx, "non-existent-client")
	if err != nil {
		t.Fatalf("Failed to get storage usage for non-existent client: %v", err)
	}

	if nonExistentUsage != 0 {
		t.Errorf("Expected 0 bytes for non-existent client, got %d", nonExistentUsage)
	}

	// Delete one clip and verify usage is updated
	err = repo.Delete(ctx, "client-a-clip-1")
	if err != nil {
		t.Fatalf("Failed to delete clip: %v", err)
	}

	clientAUsageAfterDelete, err := repo.GetTotalStorageUsage(ctx, "client-a")
	if err != nil {
		t.Fatalf("Failed to get storage usage for client-a after delete: %v", err)
	}

	expectedClientAUsageAfterDelete := int64(2 * 1024 * 1024) // 2MB (only clip-2 remains)
	if clientAUsageAfterDelete != expectedClientAUsageAfterDelete {
		t.Errorf("Expected %d bytes for client-a after delete, got %d", expectedClientAUsageAfterDelete, clientAUsageAfterDelete)
	}
}

func TestSQLiteClipRepository_GetOldestClips(t *testing.T) {
	repo, cleanup := setupTestRepo(t)
	defer cleanup()

	ctx := context.Background()

	// Test with no clips (should return empty slice)
	oldestClips, err := repo.GetOldestClips(ctx, "client-empty", 5)
	if err != nil {
		t.Fatalf("Failed to get oldest clips for empty client: %v", err)
	}

	if len(oldestClips) != 0 {
		t.Errorf("Expected 0 clips for empty client, got %d", len(oldestClips))
	}

	// Add clips with different timestamps for multiple clients
	baseTime := time.Now().UTC().Add(-10 * time.Hour)
	clips := []*Clip{
		{
			ID:                 "client-a-oldest",
			ClientID:           "client-a",
			Title:              "Client A Oldest",
			TimeStamp:          baseTime, // Oldest
			Duration:           time.Duration(30 * time.Second),
			HasMotion:          true,
			EncryptedVideo:     []byte("video-a-oldest"),
			VideoWidth:         1920,
			VideoHeight:        1080,
			VideoMimeType:      "video/mp4",
			EncryptedThumbnail: []byte("thumb-a-oldest"),
			ThumbnailWidth:     320,
			ThumbnailHeight:    240,
			ThumbnailMimeType:  "image/jpeg",
		},
		{
			ID:                 "client-a-middle",
			ClientID:           "client-a",
			Title:              "Client A Middle",
			TimeStamp:          baseTime.Add(2 * time.Hour), // Middle
			Duration:           time.Duration(45 * time.Second),
			HasMotion:          false,
			EncryptedVideo:     []byte("video-a-middle"),
			VideoWidth:         1280,
			VideoHeight:        720,
			VideoMimeType:      "video/mp4",
			EncryptedThumbnail: []byte("thumb-a-middle"),
			ThumbnailWidth:     256,
			ThumbnailHeight:    144,
			ThumbnailMimeType:  "image/png",
		},
		{
			ID:                 "client-a-newest",
			ClientID:           "client-a",
			Title:              "Client A Newest",
			TimeStamp:          baseTime.Add(4 * time.Hour), // Newest
			Duration:           time.Duration(60 * time.Second),
			HasMotion:          true,
			EncryptedVideo:     []byte("video-a-newest"),
			VideoWidth:         1920,
			VideoHeight:        1080,
			VideoMimeType:      "video/mp4",
			EncryptedThumbnail: []byte("thumb-a-newest"),
			ThumbnailWidth:     320,
			ThumbnailHeight:    240,
			ThumbnailMimeType:  "image/jpeg",
		},
		{
			ID:                 "client-b-only",
			ClientID:           "client-b",
			Title:              "Client B Only",
			TimeStamp:          baseTime.Add(1 * time.Hour),
			Duration:           time.Duration(50 * time.Second),
			HasMotion:          true,
			EncryptedVideo:     []byte("video-b-only"),
			VideoWidth:         1920,
			VideoHeight:        1080,
			VideoMimeType:      "video/mp4",
			EncryptedThumbnail: []byte("thumb-b-only"),
			ThumbnailWidth:     320,
			ThumbnailHeight:    240,
			ThumbnailMimeType:  "image/jpeg",
		},
	}

	for _, clip := range clips {
		err := repo.Add(ctx, clip)
		if err != nil {
			t.Fatalf("Failed to add clip %s: %v", clip.ID, err)
		}
	}

	// Test getting 1 oldest clip for client-a
	oldestOne, err := repo.GetOldestClips(ctx, "client-a", 1)
	if err != nil {
		t.Fatalf("Failed to get 1 oldest clip for client-a: %v", err)
	}

	if len(oldestOne) != 1 {
		t.Errorf("Expected 1 oldest clip, got %d", len(oldestOne))
	}

	if oldestOne[0].ID != "client-a-oldest" {
		t.Errorf("Expected oldest clip to be 'client-a-oldest', got '%s'", oldestOne[0].ID)
	}

	// Verify the timestamp is correctly parsed
	if !oldestOne[0].TimeStamp.Equal(baseTime) {
		t.Errorf("Expected timestamp %v, got %v", baseTime, oldestOne[0].TimeStamp)
	}

	// Verify other fields are correctly loaded
	if oldestOne[0].ClientID != "client-a" {
		t.Errorf("Expected ClientID 'client-a', got '%s'", oldestOne[0].ClientID)
	}
	if oldestOne[0].Title != "Client A Oldest" {
		t.Errorf("Expected Title 'Client A Oldest', got '%s'", oldestOne[0].Title)
	}
	if oldestOne[0].HasMotion != true {
		t.Errorf("Expected HasMotion true, got %v", oldestOne[0].HasMotion)
	}

	// Test getting 2 oldest clips for client-a
	oldestTwo, err := repo.GetOldestClips(ctx, "client-a", 2)
	if err != nil {
		t.Fatalf("Failed to get 2 oldest clips for client-a: %v", err)
	}

	if len(oldestTwo) != 2 {
		t.Errorf("Expected 2 oldest clips, got %d", len(oldestTwo))
	}

	// Should be ordered by timestamp ASC: oldest, middle
	if oldestTwo[0].ID != "client-a-oldest" || oldestTwo[1].ID != "client-a-middle" {
		t.Errorf("Expected clips ordered oldest->middle, got %s->%s", oldestTwo[0].ID, oldestTwo[1].ID)
	}

	// Test getting all clips for client-a
	allOldest, err := repo.GetOldestClips(ctx, "client-a", 10)
	if err != nil {
		t.Fatalf("Failed to get all oldest clips for client-a: %v", err)
	}

	if len(allOldest) != 3 {
		t.Errorf("Expected 3 clips for client-a, got %d", len(allOldest))
	}

	// Should be ordered by timestamp ASC: oldest, middle, newest
	expectedOrder := []string{"client-a-oldest", "client-a-middle", "client-a-newest"}
	for i, clip := range allOldest {
		if clip.ID != expectedOrder[i] {
			t.Errorf("Expected clip %d to be '%s', got '%s'", i, expectedOrder[i], clip.ID)
		}
	}

	// Test client-b (only 1 clip)
	clientBOldest, err := repo.GetOldestClips(ctx, "client-b", 5)
	if err != nil {
		t.Fatalf("Failed to get oldest clips for client-b: %v", err)
	}

	if len(clientBOldest) != 1 {
		t.Errorf("Expected 1 clip for client-b, got %d", len(clientBOldest))
	}

	if clientBOldest[0].ID != "client-b-only" {
		t.Errorf("Expected 'client-b-only', got '%s'", clientBOldest[0].ID)
	}

	// Test non-existent client
	nonExistentOldest, err := repo.GetOldestClips(ctx, "non-existent-client", 5)
	if err != nil {
		t.Fatalf("Failed to get oldest clips for non-existent client: %v", err)
	}

	if len(nonExistentOldest) != 0 {
		t.Errorf("Expected 0 clips for non-existent client, got %d", len(nonExistentOldest))
	}

	// Test limit of 0
	zeroLimit, err := repo.GetOldestClips(ctx, "client-a", 0)
	if err != nil {
		t.Fatalf("Failed to get oldest clips with limit 0: %v", err)
	}

	if len(zeroLimit) != 0 {
		t.Errorf("Expected 0 clips with limit 0, got %d", len(zeroLimit))
	}

	// Test that clips are properly isolated by client
	// Delete client-a clips and verify client-b is unaffected
	for _, clipID := range []string{"client-a-oldest", "client-a-middle", "client-a-newest"} {
		err = repo.Delete(ctx, clipID)
		if err != nil {
			t.Fatalf("Failed to delete clip %s: %v", clipID, err)
		}
	}

	// Client-a should now have no clips
	clientAAfterDelete, err := repo.GetOldestClips(ctx, "client-a", 5)
	if err != nil {
		t.Fatalf("Failed to get oldest clips for client-a after delete: %v", err)
	}

	if len(clientAAfterDelete) != 0 {
		t.Errorf("Expected 0 clips for client-a after delete, got %d", len(clientAAfterDelete))
	}

	// Client-b should still have its clip
	clientBAfterDelete, err := repo.GetOldestClips(ctx, "client-b", 5)
	if err != nil {
		t.Fatalf("Failed to get oldest clips for client-b after deleting client-a clips: %v", err)
	}

	if len(clientBAfterDelete) != 1 {
		t.Errorf("Expected 1 clip for client-b after deleting client-a clips, got %d", len(clientBAfterDelete))
	}
}
