package tasks

import (
	"context"
	"fmt"
	"github.com/neptune-media/robin/pkg/ffmpeg"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
	"path/filepath"
	"strings"
)

type TranscodeVideo struct {
	Logger  *zap.SugaredLogger
	Options TranscodeVideoOptions
	WorkDir string
}

type TranscodeVideoOptions struct {
	AudioLanguages          []string               `yaml:"audio_languages,omitempty"`
	AudioEncodingOptions    map[string]interface{} `yaml:"audio_options,omitempty"`
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
	buf, _ := yaml.Marshal(opts)
	return ffmpeg.NewEncodingOptionsFromBytes(buf)
}

func (t *TranscodeVideo) Do(ctx context.Context, inputFilename string) (string, error) {
	logger := t.Logger
	basename := strings.TrimSuffix(filepath.Base(inputFilename), filepath.Ext(inputFilename))
	outputFilename := filepath.Join(t.WorkDir, fmt.Sprintf("%s-output.mkv", basename))

	opts := t.Options
	audioOpts, _ := newEncodingOptionsFromTask(opts.AudioEncodingOptions)
	subtitleOpts, _ := newEncodingOptionsFromTask(opts.SubtitleEncodingOptions)
	videoOpts, _ := newEncodingOptionsFromTask(opts.VideoEncodingOptions)

	listener := new(ffmpeg.ProgressListener)
	addr, err := listener.Begin()
	if err != nil {
		logger.Errorw("error while starting ffmpeg progress listener", "err", err)
		return "", err
	}
	defer listener.Close()

	runner := &ffmpeg.Runner{
		AudioLanguages:    opts.AudioLanguages,
		AudioOptions:      audioOpts,
		EnableFastStart:   opts.EnableFastStart,
		InputArgs:         append(opts.InputArgs, "-progress", addr),
		InputFilename:     inputFilename,
		OutputFilename:    outputFilename,
		SubtitleLanguages: opts.SubtitleLanguages,
		SubtitleOptions:   subtitleOpts,
		VideoOptions:      videoOpts,
	}

	go listener.Run(logger)
	logger.Infow("running ffmpeg", "command", runner.GetCommandString())

	err = runner.Do()
	if err != nil {
		fmt.Printf("output? %s\n", runner.GetOutput())
	}

	return outputFilename, err
}
