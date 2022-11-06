package ffmpeg

import (
	"fmt"
	"gopkg.in/yaml.v3"
)

type EncodingOptions interface {
	Decode(data []byte) error
	GetCodecOptions() []string
}

type stubOptions struct {
	Codec string
}

func NewEncodingOptionsFromBytes(data []byte) (EncodingOptions, error) {
	stub := &stubOptions{}
	if err := yaml.Unmarshal(data, stub); err != nil {
		return nil, err
	}

	var opts EncodingOptions
	switch codec := stub.Codec; codec {
	case "copy":
		opts = &CopyOptions{}
	case "libx264":
		opts = &Libx264Options{}
	case "libx265":
		opts = &Libx265Options{}
	default:
		return nil, fmt.Errorf("unknown codec: %s", codec)
	}

	return opts, opts.Decode(data)
}
