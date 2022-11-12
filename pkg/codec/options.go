package codec

import (
	"fmt"
	"github.com/neptune-media/MediaKit-go/tools/ffmpeg"
	"gopkg.in/yaml.v3"
)

type stubOptions struct {
	Codec string
}

func NewEncodingOptionsFromBytes(data []byte) (ffmpeg.EncodingOptions, error) {
	stub := &stubOptions{}
	if err := yaml.Unmarshal(data, stub); err != nil {
		return nil, err
	}

	var opts ffmpeg.EncodingOptions
	switch codec := stub.Codec; codec {
	case "copy":
		opts = &ffmpeg.CopyOptions{}
	case "libx264":
		opts = &ffmpeg.Libx264Options{}
	case "libx265":
		opts = &ffmpeg.Libx265Options{}
	default:
		return nil, fmt.Errorf("unknown codec: %s", codec)
	}

	err := yaml.Unmarshal(data, opts)
	return opts, err
}
