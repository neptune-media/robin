package ffmpeg

import (
	"bytes"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"sync"
)

const (
	cmdName = "ffmpeg"
)

func filter(ss []string, test func(string) bool) []string {
	r := make([]string, 0)
	for _, s := range ss {
		if test(s) {
			r = append(r, s)
		}
	}
	return r
}

type Runner struct {
	// A list of ISO 639-1 language codes to select for output
	AudioLanguages []string

	// Audio output encoding options
	AudioOptions EncodingOptions

	// Discard audio from output
	DiscardAudio bool

	// Discard subtitles from output
	DiscardSubtitles bool

	// Discard video from output
	DiscardVideo bool

	// Modifies output to enable "Fast Start" for web streaming
	EnableFastStart bool

	// A list of raw args to set for input
	InputArgs []string

	// Path to read input from
	InputFilename string

	// A list of raw args to set for output
	OutputArgs []string

	// Path to store output in
	OutputFilename string

	// A list of ISO 639-1 language codes to select for output
	SubtitleLanguages []string

	// Subtitle output encoding options
	SubtitleOptions EncodingOptions

	// Video output encoding options
	VideoOptions EncodingOptions

	output []byte
}

func (r *Runner) Do() error {
	args := r.buildArgs()
	c := exec.Command(cmdName, args...)
	o, err := c.CombinedOutput()

	r.output = make([]byte, len(o))
	copy(r.output, o)
	return err
}

func (r *Runner) GetCommandString() string {
	cmd := []string{cmdName}
	cmd = append(cmd, r.buildArgs()...)
	return strings.Join(cmd, " ")
}

func (r *Runner) GetOutput() []byte {
	o := make([]byte, len(r.output))
	copy(o, r.output)
	return o
}

func (r *Runner) buildArgs() []string {
	args := make([]string, 0)
	args = append(args, r.InputArgs...)
	args = append(args,
		"-i",
		r.InputFilename,
	)

	if r.DiscardAudio {
		args = append(args, "-an")
	} else if r.AudioOptions != nil {
		args = append(args, "-c:a")
		args = append(args, r.AudioOptions.GetCodecOptions()...)
	}

	if r.DiscardSubtitles {
		args = append(args, "-sn")
	} else if r.SubtitleOptions != nil {
		args = append(args, "-c:s")
		args = append(args, r.SubtitleOptions.GetCodecOptions()...)
	}

	if r.DiscardVideo {
		args = append(args, "-vn")
	} else if r.VideoOptions != nil {
		args = append(args, "-c:v")
		args = append(args, r.VideoOptions.GetCodecOptions()...)
	}

	if r.EnableFastStart {
		args = append(args, "-movflags", "+faststart")
	}

	args = append(args, "-map", "0:v:0")
	if len(r.AudioLanguages) > 0 {
		for _, lang := range r.AudioLanguages {
			args = append(args, "-map", fmt.Sprintf("0:a:m:language:%s", lang))
		}
	}

	if len(r.SubtitleLanguages) > 0 {
		for _, lang := range r.SubtitleLanguages {
			args = append(args, "-map", fmt.Sprintf("0:s:m:language:%s", lang))
		}
	}

	args = append(args, r.OutputArgs...)
	args = append(args,
		"-y",
		r.OutputFilename,
	)

	return filter(args, func(s string) bool { return len(s) > 0 })
}

func (r *Runner) execAndWait(args ...string) ([]byte, error) {
	c := exec.Command(cmdName, args...)

	stdout, err := c.StdoutPipe()
	if err != nil {
		return nil, err
	}

	if err := c.Start(); err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	bufLock := &sync.Mutex{}
	go func() {
		bufLock.Lock()
		defer bufLock.Unlock()
		for {
			if _, err := io.Copy(&buf, stdout); err != nil {
				return
			}
		}
	}()

	if err := c.Wait(); err != nil {
		return nil, err
	}
	bufLock.Lock()
	defer bufLock.Unlock()

	arr := make([]byte, buf.Len())
	copy(arr, buf.Bytes())

	return arr, nil
}
