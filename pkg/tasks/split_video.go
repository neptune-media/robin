package tasks

import (
	"context"
	"fmt"
	mediakit "github.com/neptune-media/MediaKit-go"
	mediatasks "github.com/neptune-media/MediaKit-go/tasks"
	"github.com/neptune-media/MediaKit-go/tools/ffprobe"
	"github.com/neptune-media/MediaKit-go/tools/mkvmerge"
	"github.com/neptune-media/MediaKit-go/tools/mkvpropedit"
	"go.uber.org/zap"
	"log"
	"path/filepath"
	"time"
)

type SplitVideo struct {
	Logger           *zap.SugaredLogger
	Options          SplitVideoOptions
	UseLowerPriority bool
	WorkDir          string
}

type SplitVideoOptions struct {
	EndingChapterTime    int
	MinimumChapters      int
	MinimumEpisodeLength int
}

var defaultEpisodeBuilderOptions = mediakit.EpisodeBuilderOptions{
	EndingChapterTime:    60 * time.Second,
	IgnoreMissingEnd:     false,
	MinimumChapters:      2,
	MinimumEpisodeLength: 20 * time.Minute,
}

// Helper null sink for matroska logs
type sink int

func (s sink) Write(p []byte) (int, error) {
	return len(p), nil
}

func (t *SplitVideo) Do(ctx context.Context, inputFilename string) ([]string, error) {
	logger := t.Logger
	outputFilename := filepath.Join(t.WorkDir, "episode.mkv")

	// matroska-go outputs every block and is super noisy
	log.SetOutput(new(sink))

	// Read video I-frames
	logger.Infow("using input file", "filename", inputFilename)
	logger.Infow("reading i-frames")
	probe := &ffprobe.FFProbe{Filename: inputFilename, GetFrames: true, LowPriority: t.UseLowerPriority}
	frames, err := mediatasks.ReadVideoIFrames(probe)
	if err != nil {
		return nil, err
	}

	// Use I-frames to calculate episode cutpoints
	opts := t.getEpisodeBuilderOptions(frames)
	logger.Infow("reading video episodes")
	episodes, err := mediatasks.ReadVideoEpisodes(inputFilename, *opts)
	if err != nil {
		return nil, fmt.Errorf("error while reading episodes: %v", err)
	}

	// Split video
	logger.Infow("splitting video")
	runner := mkvmerge.NewSplitter(
		inputFilename,
		outputFilename,
		episodes,
	)
	runner.LowPriority = t.UseLowerPriority

	err = runner.Do()
	if err != nil {
		return nil, fmt.Errorf("error while splitting file: %v\noutput from command:\n%s\n%s", err, runner.GetStdout(), runner.GetStderr())
	}

	logger.Infow("fixing episode chapter names")
	err = mkvpropedit.FixEpisodeChapterNames(episodes, outputFilename)
	if err != nil {
		return nil, fmt.Errorf("error while writing chapters: %v", err)
	}

	filenames := make([]string, len(episodes))
	for i := range episodes {
		filenames[i] = mkvmerge.FormatSplitOutputName(outputFilename, i)
	}

	return filenames, nil
}

func (t *SplitVideo) getEpisodeBuilderOptions(frames []time.Duration) *mediakit.EpisodeBuilderOptions {
	taskOpts := t.Options
	opts := &mediakit.EpisodeBuilderOptions{
		FrameSeeker: &mediakit.FrameSeeker{Frames: frames},

		EndingChapterTime:    time.Duration(taskOpts.EndingChapterTime) * time.Second,
		MinimumChapters:      taskOpts.MinimumChapters,
		MinimumEpisodeLength: time.Duration(taskOpts.MinimumEpisodeLength) * time.Minute,
	}

	if opts.EndingChapterTime == 0 {
		opts.EndingChapterTime = defaultEpisodeBuilderOptions.EndingChapterTime
	}

	if opts.MinimumChapters == 0 {
		opts.MinimumChapters = defaultEpisodeBuilderOptions.MinimumChapters
	}

	if opts.MinimumEpisodeLength == 0 {
		opts.MinimumEpisodeLength = defaultEpisodeBuilderOptions.MinimumEpisodeLength
	}

	return opts
}
