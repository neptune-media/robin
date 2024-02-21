package tasks

import (
	"context"
	"fmt"
	"github.com/neptune-media/MediaKit-go/tools/ffmpeg"
	"github.com/neptune-media/robin/pkg/codec"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
	"path/filepath"
	"strings"
	"time"
)

const (
	// Defines amount of space (in kilobytes) at beginning of file to reserve for writing video cues.
	// Recommended values are 50 per hour of video, see:
	// https://www.ffmpeg.org/ffmpeg-formats.html#matroska
	matroskaReserveIndexSpacePerHour = 50
)

type TranscodeVideo struct {
	Logger           *zap.SugaredLogger
	Options          TranscodeVideoOptions
	UseLowerPriority bool
	WorkDir          string
}

type TranscodeVideoOptions struct {
	AudioLanguages          []string               `yaml:"audio_languages,omitempty"`
	AudioEncodingOptions    map[string]interface{} `yaml:"audio_options,omitempty"`
	ContainerOptions        map[string]interface{} `yaml:"container_options,omitempty"`
	CopyAllAudioStreams     bool                   `yaml:"copy_all_audio_streams,omitempty"`
	CopyAllSubtitleStreams  bool                   `yaml:"copy_all_subtitle_streams,omitempty"`
	CopyAllVideoStreams     bool                   `yaml:"copy_all_video_streams,omitempty"`
	DiscardAudio            bool                   `yaml:"discard_audio,omitempty"`
	DiscardSubtitles        bool                   `yaml:"discard_subtitles,omitempty"`
	DiscardVideo            bool                   `yaml:"discard_video,omitempty"`
	EnableFastStart         bool                   `yaml:"enable_fast_start,omitempty"`
	InputArgs               []string               `yaml:"input_args,omitempty"`
	OutputArgs              []string               `yaml:"output_args,omitempty"`
	SubtitleLanguages       []string               `yaml:"subtitle_languages,omitempty"`
	SubtitleEncodingOptions map[string]interface{} `yaml:"subtitle_options,omitempty"`
	VideoEncodingOptions    map[string]interface{} `yaml:"video_options,omitempty"`
}

func newEncodingOptionsFromTask(opts map[string]interface{}) (ffmpeg.EncodingOptions, error) {
	return newEncodingOptionsFromTaskWithFallback(opts, nil)
}

func newEncodingOptionsFromTaskWithFallback(opts map[string]interface{}, fallback ffmpeg.EncodingOptions) (ffmpeg.EncodingOptions, error) {
	buf, _ := yaml.Marshal(opts)
	return codec.NewEncodingOptionsFromBytesWithFallback(buf, fallback)
}

func (t *TranscodeVideo) Do(ctx context.Context, inputFilename string, analyzeResults *AnalyzeResults) (string, error) {
	logger := t.Logger
	basename := strings.TrimSuffix(filepath.Base(inputFilename), filepath.Ext(inputFilename))
	outputFilename := filepath.Join(t.WorkDir, fmt.Sprintf("%s-output.mkv", basename))

	opts := t.Options
	audioOpts, _ := newEncodingOptionsFromTaskWithFallback(opts.AudioEncodingOptions, &ffmpeg.GenericAudioOptions{})
	containerOpts, _ := newEncodingOptionsFromTask(opts.ContainerOptions)
	subtitleOpts, _ := newEncodingOptionsFromTask(opts.SubtitleEncodingOptions)
	videoOpts, _ := newEncodingOptionsFromTask(opts.VideoEncodingOptions)

	// Configure some container options from helper flags
	if err := t.configureContainerOptsFromFlags(containerOpts, analyzeResults); err != nil {
		return "", err
	}

	// Setup progress listener
	listener := new(ffmpeg.ProgressListener)
	listener.ReportInterval = time.Second
	if analyzeResults != nil {
		listener.TotalFrameCount = analyzeResults.TotalFrames
	}

	addr, err := listener.Begin()
	if err != nil {
		logger.Errorw("error while starting codec progress listener", "err", err)
		return "", err
	}
	defer listener.Close()

	runner := &ffmpeg.FFmpeg{
		AudioLanguages:        opts.AudioLanguages,
		AudioOptions:          audioOpts,
		ContainerOptions:      containerOpts,
		InputArgs:             append(opts.InputArgs, "-progress", addr),
		InputFilename:         inputFilename,
		MapAllAudioStreams:    opts.CopyAllAudioStreams,
		MapAllSubtitleStreams: opts.CopyAllSubtitleStreams,
		MapAllVideoStreams:    opts.CopyAllVideoStreams,
		OutputArgs:            opts.OutputArgs,
		OutputFilename:        outputFilename,
		SubtitleLanguages:     opts.SubtitleLanguages,
		SubtitleOptions:       subtitleOpts,
		UseLowerPriority:      t.UseLowerPriority,
		VideoOptions:          videoOpts,
	}

	go listener.Run(logger)
	logger.Infow("running ffmpeg",
		"command", runner.GetCommand(),
		"args", strings.Join(runner.GetCommandArgs(), " "))

	err = runner.DoWithContext(ctx)
	if err != nil {
		fmt.Printf("output? %s\n%s\n", runner.GetStdout(), runner.GetStderr())
	}

	return outputFilename, err
}

// configureContainerOptsFromFlags is used to update container options from helper flags in TranscodeVideoOptions
func (t *TranscodeVideo) configureContainerOptsFromFlags(opts ffmpeg.EncodingOptions, analyzeResults *AnalyzeResults) error {
	switch opts.(type) {
	case *ffmpeg.MkvContainerOptions:
		cOpts := opts.(*ffmpeg.MkvContainerOptions)
		if t.Options.EnableFastStart {
			// The MKV FastStart equivalent requires pre-allocating index space at the beginning of the file, which
			// requires knowing approximately the duration of the video.
			if analyzeResults != nil && analyzeResults.Duration > 0 {
				cOpts.ReserveIndexSpace = matroskaReserveIndexSpacePerHour * (int(analyzeResults.Duration.Truncate(time.Hour).Hours()) + 1)
			}
		}

	case *ffmpeg.Mp4ContainerOptions:
		cOpts := opts.(*ffmpeg.Mp4ContainerOptions)
		if t.Options.EnableFastStart {
			cOpts.EnableFastStart = true
		}
	}
	return nil
}
