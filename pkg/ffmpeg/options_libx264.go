package ffmpeg

import (
	"fmt"
	"gopkg.in/yaml.v3"
)

type Libx264Options struct {
	// Corresponds to the -qp flag
	QP int

	// -preset flag
	Preset string

	// -tune flag
	Tune string

	// Add -qp flag
	UseQP bool
}

func (o *Libx264Options) Decode(data []byte) error {
	if err := yaml.Unmarshal(data, o); err != nil {
		return fmt.Errorf("error while unmarshalling data: %s", err)
	}
	return nil
}

func (o *Libx264Options) GetCodecOptions() []string {
	args := []string{
		"libx264",
	}
	args = append(args, o.getQuantizationParam()...)
	args = append(args, o.getPreset()...)
	args = append(args, o.getTune()...)

	return filter(args, func(s string) bool { return len(s) > 0 })
}

func (o *Libx264Options) getQuantizationParam() []string {
	if !o.UseQP {
		return []string{}
	}

	return []string{"-qp", fmt.Sprintf("%d", o.QP)}
}

func (o *Libx264Options) getPreset() []string {
	if o.Preset == "" {
		return []string{}
	}

	return []string{"-preset", o.Preset}
}

func (o *Libx264Options) getTune() []string {
	if o.Tune == "" {
		return []string{}
	}

	return []string{"-tune", o.Tune}
}
