package recording

import (
	"path"
	"strings"
	"time"

	"github.com/yeti47/cryospy/client/capture-client/common"
)

type RawClip struct {
	Path      string
	Codec     string
	Timestamp time.Time
	Duration  time.Duration
	Frames    int
	FrameRate float64
}

func (c *RawClip) FileExtension() string {
	// Determine extension based on path
	if strings.Contains(c.Path, ".") {
		return strings.TrimLeft(path.Ext(c.Path), ".")
	}
	// Fallback to codec-based extension
	return common.CodecToFileExtension(c.Codec)
}
