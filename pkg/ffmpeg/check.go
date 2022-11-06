package ffmpeg

import (
	"github.com/neptune-media/MediaKit-go/tools"
)

func NewCheck() tools.Check {
	return &tools.ExecutableCheck{Executable: cmdName}
}
