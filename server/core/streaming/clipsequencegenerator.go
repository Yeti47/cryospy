package streaming

import (
	"time"
)

type ClipSequenceGenerator interface {
	// GetStreamSequenceNumber returns a sequence number for a stream position based on virtual time
	// This provides continuous sequence numbering for HLS streams
	GetStreamSequenceNumber(virtualTime time.Time, segmentDuration time.Duration) int64
}

type ChronoClipSequenceGenerator struct {
}

func NewChronoClipSequenceGenerator() *ChronoClipSequenceGenerator {
	return &ChronoClipSequenceGenerator{}
}

func (g *ChronoClipSequenceGenerator) GetStreamSequenceNumber(virtualTime time.Time, segmentDuration time.Duration) int64 {
	// Use a custom epoch for smaller numbers.
	const customEpoch = 1609459200 // January 1, 2021

	// Calculate sequence number based on virtual time and typical segment duration
	// This ensures continuous sequence numbers for HLS streams
	virtualUnix := virtualTime.Unix() - customEpoch

	// Use standard 30-second segments if duration is not provided or invalid
	duration := segmentDuration.Seconds()
	if duration <= 0 {
		duration = 30.0
	}

	// Calculate sequence number as virtual time divided by segment duration
	return int64(float64(virtualUnix) / duration)
}
