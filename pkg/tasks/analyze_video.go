package tasks

import (
	"context"
	"fmt"
	"github.com/neptune-media/MediaKit-go/tools/ffprobe"
	"go.uber.org/zap"
	"strconv"
	"strings"
	"time"
)

type AnalyzeVideo struct {
	Logger           *zap.SugaredLogger
	Threads          int
	UseLowerPriority bool
	UseThreads       bool
}

type AnalyzeResults struct {
	Duration           time.Duration // Length of the video
	NumAudioStreams    int           // Number of audio streams in source file
	NumSubtitleStreams int           // Number of subtitle streams in source file
	NumVideoStreams    int           // Number of video streams in source file
	TotalFrames        int           // Total number of frames in the video
}

func (t *AnalyzeVideo) Do(ctx context.Context, inputFilename string) (*AnalyzeResults, error) {
	logger := t.Logger

	// Analyze video streams
	logger.Infow("using input file", "filename", inputFilename)
	logger.Infow("reading video data")
	probe := &ffprobe.FFProbe{
		Filename:      inputFilename,
		GetFrameCount: true,
		LowPriority:   t.UseLowerPriority,
		Threads:       t.Threads,
		UseThreads:    t.UseThreads,
	}

	if err := probe.DoWithContext(ctx); err != nil {
		return nil, fmt.Errorf("error while analyzing video: %s", err)
	}

	output, err := probe.GetOutput()
	if err != nil {
		return nil, err
	}

	// Select first video stream we find
	videoStream := ffprobe.Stream{}
	for _, stream := range output.Streams {
		if stream.CodecType == "video" {
			videoStream = stream
			break
		}
	}

	totalFrames, _ := strconv.Atoi(videoStream.NbReadFrames)
	results := &AnalyzeResults{TotalFrames: totalFrames}
	err = results.SetDurationFromFramerateString(videoStream.AvgFrameRate)

	for _, stream := range output.Streams {
		switch stream.CodecType {
		case "audio":
			results.NumAudioStreams++
		case "subtitle":
			results.NumSubtitleStreams++
		case "video":
			results.NumVideoStreams++
		}
	}

	logger.Infow("analysis results",
		"total frames", results.TotalFrames,
		"duration", results.Duration,
		"duration-friendly", results.Duration.String(),
		"num-audio-streams", results.NumAudioStreams,
		"num-subtitle-streams", results.NumSubtitleStreams,
		"num-video-streams", results.NumVideoStreams,
	)
	return results, err
}

func (r *AnalyzeResults) SetDurationFromFramerateString(framerate string) error {
	fps, err := parseStringToFloat(framerate)
	if err != nil {
		return err
	}

	r.Duration = time.Duration(float64(r.TotalFrames)/fps) * time.Second
	return nil
}

func parseStringToFloat(s string) (float64, error) {
	parts := strings.Split(s, "/")

	// We're expecting format "numerator/divisor", so if we don't have a divisor, we need to add one
	if len(parts) == 1 {
		parts = append(parts, "1")
	}

	numerator, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, err
	}

	divisor, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, err
	}

	return float64(numerator) / float64(divisor), nil
}
