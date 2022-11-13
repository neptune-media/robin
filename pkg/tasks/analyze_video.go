package tasks

import (
	"context"
	"fmt"
	"github.com/neptune-media/MediaKit-go/tools/ffprobe"
	"go.uber.org/zap"
	"strconv"
)

type AnalyzeVideo struct {
	Logger           *zap.SugaredLogger
	UseLowerPriority bool
}

func (t *AnalyzeVideo) Do(ctx context.Context, inputFilename string) (int, error) {
	logger := t.Logger

	// Analyze video streams
	logger.Infow("using input file", "filename", inputFilename)
	logger.Infow("reading video data")
	probe := &ffprobe.FFProbe{Filename: inputFilename, GetFrameCount: true, LowPriority: t.UseLowerPriority}

	if err := probe.DoWithContext(ctx); err != nil {
		return 0, fmt.Errorf("error while analyzing video: %s", err)
	}

	output, err := probe.GetOutput()
	totalFrames, _ := strconv.Atoi(output.Streams[0].NbReadFrames)
	return totalFrames, err
}
