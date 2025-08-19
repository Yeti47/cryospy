package postprocessing

import "time"

type VideoClip struct {
	Path      string
	Codec     string
	Format    string
	Timestamp time.Time
	Duration  time.Duration
}
