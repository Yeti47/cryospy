package streaming

import (
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/yeti47/cryospy/server/core/ccc/logging"
	"github.com/yeti47/cryospy/server/core/videos"
)

/*
Example M3U8 Playlist Format:

#EXTM3U
#EXT-X-VERSION:3
#EXT-X-TARGETDURATION:30
#EXT-X-MEDIA-SEQUENCE:810234900

#EXTINF:30.0,{"title":"clip title 1","recorded_at":"2025-08-19T20:00:00Z","motion":true}
segments/clip_105.ts

#EXTINF:30.0,{"title":"clip title 2","recorded_at":"2025-08-19T20:00:30Z","motion":false}
segments/clip_106.ts

#EXTINF:30.0,{"title":"clip title 3","recorded_at":"2025-08-19T20:01:00Z","motion":true}
segments/clip_107.ts

*/

const (
	protocolVersion = 3
)

type PlaylistGenerator interface {
	// GeneratePlaylist generates a playlist for the given clips.
	GeneratePlaylist(clips []*videos.ClipInfo, isLive bool) (string, error)
}

type M3U8PlaylistGenerator struct {
	logger logging.Logger
	seqGen ClipSequenceGenerator
}

func NewM3U8PlaylistGenerator(logger logging.Logger) *M3U8PlaylistGenerator {

	if logger == nil {
		logger = logging.NopLogger
	}

	return &M3U8PlaylistGenerator{
		logger: logger,
		seqGen: NewChronoClipSequenceGenerator(),
	}
}

func (p *M3U8PlaylistGenerator) GeneratePlaylist(clips []*videos.ClipInfo, isLive bool) (string, error) {
	p.logger.Info("Generating M3U8 playlist for clips", "count", len(clips))

	builder := &strings.Builder{}

	builder.WriteString("#EXTM3U\n")
	builder.WriteString(fmt.Sprintf("#EXT-X-VERSION:%d\n", protocolVersion))

	duration := 30 // default to 30 seconds
	var sequenceNumber int64 = 0

	if len(clips) > 0 {
		sort.Slice(clips, func(i, j int) bool {
			return clips[i].TimeStamp.Before(clips[j].TimeStamp)
		})
		duration = p.findMaxDuration(clips)
		sequenceNumber = p.seqGen.GetSequenceNumber(clips[0])
	} else {
		p.logger.Warn("No clips supplied, generating empty playlist")
	}

	builder.WriteString(fmt.Sprintf("#EXT-X-TARGETDURATION:%d\n", duration))
	builder.WriteString(fmt.Sprintf("#EXT-X-MEDIA-SEQUENCE:%d\n", sequenceNumber))

	builder.WriteString("\n")

	for _, clip := range clips {
		p.writeSegment(builder, clip)
		builder.WriteString("\n")
	}

	if !isLive {
		builder.WriteString("#EXT-X-ENDLIST\n")
	}

	playlist := builder.String()

	p.logger.Info("Generated M3U8 playlist", "length", len(playlist))

	return playlist, nil
}

func (p *M3U8PlaylistGenerator) findMaxDuration(clips []*videos.ClipInfo) int {
	maxDuration := 0
	for _, clip := range clips {
		// get the number of seconds rounded up to the nearest second
		duration := int(math.Ceil(clip.Duration.Seconds()))
		if duration > maxDuration {
			maxDuration = duration
		}
	}
	return maxDuration
}

func (p *M3U8PlaylistGenerator) writeSegment(builder *strings.Builder, clip *videos.ClipInfo) {
	// Format duration as seconds with one decimal place, e.g., 30.0
	duration := clip.Duration.Seconds()
	builder.WriteString(fmt.Sprintf("#EXTINF:%.1f,", duration))

	segmentMetaData := struct {
		Title      string `json:"title"`
		RecordedAt string `json:"recorded_at"`
		Motion     bool   `json:"motion"`
	}{
		Title:      clip.Title,
		RecordedAt: clip.TimeStamp.Format(time.RFC3339),
		Motion:     clip.HasMotion,
	}
	jsonBytes, err := json.Marshal(segmentMetaData)
	if err != nil {
		p.logger.Error("Failed to marshal segment meta data", "error", err)
		builder.WriteString("{}")
	} else {
		builder.Write(jsonBytes)
	}
	builder.WriteString("\n")

	builder.WriteString("/stream/" + clip.ClientID + "/segments/" + clip.ID + "\n")
}
