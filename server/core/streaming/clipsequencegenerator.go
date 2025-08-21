package streaming

import "github.com/yeti47/cryospy/server/core/videos"

type ClipSequenceGenerator interface {
	// GetSequenceNumber returns the sequence number for the given clip.
	// Sequence numbers are used to order clips in a sequence and are unique for each clip within the same client.
	GetSequenceNumber(clip *videos.ClipInfo) int64
}

type ChronoClipSequenceGenerator struct {
}

func NewChronoClipSequenceGenerator() *ChronoClipSequenceGenerator {
	return &ChronoClipSequenceGenerator{}
}

func (g *ChronoClipSequenceGenerator) GetSequenceNumber(clip *videos.ClipInfo) int64 {

	// Use a custom epoch for smaller numbers.
	const customEpoch = 1609459200 // January 1, 2021

	return clip.TimeStamp.Unix() - customEpoch
}
