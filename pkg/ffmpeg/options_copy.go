package ffmpeg

import (
	"fmt"
	"gopkg.in/yaml.v3"
)

type CopyOptions struct{}

func (o *CopyOptions) Decode(data []byte) error {
	if err := yaml.Unmarshal(data, o); err != nil {
		return fmt.Errorf("error while unmarshalling data: %s", err)
	}
	return nil
}

func (o *CopyOptions) GetCodecOptions() []string {
	return []string{"copy"}
}
