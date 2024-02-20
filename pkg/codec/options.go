package codec

import (
	"fmt"
	"github.com/neptune-media/MediaKit-go/tools/ffmpeg"
	"gopkg.in/yaml.v3"
)

type stubOptions struct {
	Codec  string
	Format string
}

func NewEncodingOptionsFromBytes(data []byte) (ffmpeg.EncodingOptions, error) {
	return NewEncodingOptionsFromBytesWithFallback(data, nil)
}

func NewEncodingOptionsFromBytesWithFallback(data []byte, fallback ffmpeg.EncodingOptions) (ffmpeg.EncodingOptions, error) {
	stub := &stubOptions{}
	if err := yaml.Unmarshal(data, stub); err != nil {
		return nil, err
	}

	// Get type from codec
	optionType := stub.Codec
	if len(optionType) == 0 {
		// If no codec, then maybe it's a format instead (container options, etc)
		optionType = stub.Format
	}

	var opts ffmpeg.EncodingOptions
	switch optionType {
	case "copy":
		opts = &ffmpeg.CopyOptions{}
	case "libx264":
		opts = &ffmpeg.Libx264Options{}
	case "libx265":
		opts = &ffmpeg.Libx265Options{}
	case "matroska":
		opts = &ffmpeg.MkvContainerOptions{}
	case "mp4":
		opts = &ffmpeg.Mp4ContainerOptions{}
	default:
		if fallback == nil {
			return nil, fmt.Errorf("unknown codec or format: %s", optionType)
		}
		opts = fallback
	}

	err := yaml.Unmarshal(data, opts)
	return opts, err
}
