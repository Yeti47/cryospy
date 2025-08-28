package motiondetection

import (
	"fmt"
	"image"
	"log"

	"github.com/yeti47/cryospy/client/capture-client/config"
	"gocv.io/x/gocv"
)

var DefaultMotionDetectionSettings = MotionDetectionSettings{
	MotionMinArea:      1000, // Default minimum area of motion
	MaxFramesToCheck:   300,  // Default maximum frames to check for motion
	WarmUpFrames:       30,   // Default warm-up frames to skip
	MotionMinWidth:     20,   // Default minimum width of detected motion
	MotionMinHeight:    20,   // Default minimum height of detected motion
	MotionMinAspect:    0.3,  // Default minimum aspect ratio
	MotionMaxAspect:    3.0,  // Default maximum aspect ratio
	MotionMogHistory:   500,  // Default MOG2 history parameter
	MotionMogVarThresh: 16.0, // Default MOG2 var threshold parameter
}

type MotionDetector interface {
	// DetectMotion analyzes the video at the given path
	// and returns true if motion is detected, false otherwise.
	DetectMotion(videoPath string) (bool, error)
}

type GoCVMotionDetector struct {
	settingsProvider config.SettingsProvider[MotionDetectionSettings]
}

func NewGoCVMotionDetector(provider config.SettingsProvider[MotionDetectionSettings]) *GoCVMotionDetector {
	return &GoCVMotionDetector{
		settingsProvider: provider,
	}
}

func (d *GoCVMotionDetector) DetectMotion(videoPath string) (bool, error) {

	// Get the latest settings for this operation.
	// The provider is responsible for its own thread safety.
	settings := d.settingsProvider.GetSettings()

	video, err := gocv.OpenVideoCapture(videoPath)
	if err != nil {
		return false, fmt.Errorf("failed to open video file: %w", err)
	}
	defer video.Close()

	detector := gocv.NewBackgroundSubtractorMOG2WithParams(
		settings.MotionMogHistory,
		settings.MotionMogVarThresh,
		false, // disable shadow detection
	)
	defer detector.Close()

	img := gocv.NewMat()
	defer img.Close()

	gray := gocv.NewMat()
	defer gray.Close()

	blurred := gocv.NewMat()
	defer blurred.Close()

	fgMask := gocv.NewMat()
	defer fgMask.Close()

	thresh := gocv.NewMat()
	defer thresh.Close()

	kernel := gocv.GetStructuringElement(gocv.MorphRect, image.Pt(3, 3))
	defer kernel.Close()

	prevBlurred := gocv.NewMat()
	defer prevBlurred.Close()

	motionDetected := false
	frameCount := 0

	// Use settings from the thread-safe snapshot
	maxFramesToCheck := settings.MaxFramesToCheck
	warmUpFrames := settings.WarmUpFrames
	minArea := settings.MotionMinArea
	minWidth := settings.MotionMinWidth
	minHeight := settings.MotionMinHeight
	minAspect := settings.MotionMinAspect
	maxAspect := settings.MotionMaxAspect

	if minArea <= 0 {
		minArea = DefaultMotionDetectionSettings.MotionMinArea
	}
	if maxFramesToCheck <= 0 {
		maxFramesToCheck = DefaultMotionDetectionSettings.MaxFramesToCheck
	}
	if warmUpFrames < 0 {
		warmUpFrames = DefaultMotionDetectionSettings.WarmUpFrames
	}
	if minWidth <= 0 {
		minWidth = DefaultMotionDetectionSettings.MotionMinWidth
	}
	if minHeight <= 0 {
		minHeight = DefaultMotionDetectionSettings.MotionMinHeight
	}
	if minAspect <= 0 {
		minAspect = DefaultMotionDetectionSettings.MotionMinAspect
	}
	if maxAspect <= 0 {
		maxAspect = DefaultMotionDetectionSettings.MotionMaxAspect
	}

	log.Printf(
		"Starting motion detection on video '%s': minArea = %d, maxFramesToCheck = %d, warmUpFrames = %d, minWidth = %d, minHeight = %d, minAspect = %.2f, maxAspect = %.2f",
		videoPath,
		minArea,
		maxFramesToCheck,
		warmUpFrames,
		minWidth,
		minHeight,
		minAspect,
		maxAspect,
	)

	for frameCount < maxFramesToCheck {
		if ok := video.Read(&img); !ok {
			break
		}
		if img.Empty() {
			continue
		}

		gocv.CvtColor(img, &gray, gocv.ColorBGRToGray)
		gocv.GaussianBlur(gray, &blurred, image.Pt(21, 21), 0, 0, gocv.BorderDefault)

		// Background subtraction
		detector.Apply(blurred, &fgMask)

		// Skip motion detection during warm-up
		if frameCount < warmUpFrames {
			frameCount++
			continue
		}

		// Frame differencing
		if !prevBlurred.Empty() {
			diff := gocv.NewMat()
			defer diff.Close()
			gocv.AbsDiff(blurred, prevBlurred, &diff)
			gocv.Threshold(diff, &diff, 25, 255, gocv.ThresholdBinary)

			nonZero := gocv.CountNonZero(diff)
			if nonZero < 5000 {
				blurred.CopyTo(&prevBlurred)
				frameCount++
				continue
			}
		}
		blurred.CopyTo(&prevBlurred)

		// Threshold and dilate
		gocv.Threshold(fgMask, &thresh, 25, 255, gocv.ThresholdBinary)
		gocv.Dilate(thresh, &thresh, kernel)

		contours := gocv.FindContours(thresh, gocv.RetrievalExternal, gocv.ChainApproxSimple)

		// log the number of contours found
		log.Printf("Frame %d: Found %d contours", frameCount, contours.Size())

		for i := 0; i < contours.Size(); i++ {
			area := gocv.ContourArea(contours.At(i))
			rect := gocv.BoundingRect(contours.At(i))
			aspectRatio := float64(rect.Dx()) / float64(rect.Dy())

			filtered := false
			var filterReason string

			if area < float64(minArea) {
				filtered = true
				filterReason = fmt.Sprintf("area %.2f < minArea %d", area, minArea)
			} else if rect.Dx() < minWidth {
				filtered = true
				filterReason = fmt.Sprintf("width %d < minWidth %d", rect.Dx(), minWidth)
			} else if rect.Dy() < minHeight {
				filtered = true
				filterReason = fmt.Sprintf("height %d < minHeight %d", rect.Dy(), minHeight)
			} else if aspectRatio < minAspect {
				filtered = true
				filterReason = fmt.Sprintf("aspectRatio %.2f < minAspect %.2f", aspectRatio, minAspect)
			} else if aspectRatio > maxAspect {
				filtered = true
				filterReason = fmt.Sprintf("aspectRatio %.2f > maxAspect %.2f", aspectRatio, maxAspect)
			}

			if filtered {
				log.Printf("Motion filtered out in Frame %d: Contour %d area = %.2f, width = %d, height = %d, aspectRatio = %.2f, reason: %s", frameCount, i, area, rect.Dx(), rect.Dy(), aspectRatio, filterReason)
				continue
			}

			log.Printf("Motion Detected in Frame %d: Contour %d area = %.2f, width = %d, height = %d, aspectRatio = %.2f", frameCount, i, area, rect.Dx(), rect.Dy(), aspectRatio)
			motionDetected = true
			break
		}
		contours.Close()

		if motionDetected {
			break
		}
		frameCount++
	}

	log.Printf("Finished motion detection on video: %s - Motion Detected: %t", videoPath, motionDetected)

	return motionDetected, nil
}
