package pipeline

import (
	"context"
	"fmt"
	"github.com/neptune-media/robin/pkg/tasks"
	"go.uber.org/zap"
	"os"
	"path/filepath"
)

type Pipeline struct {
	Logger    *zap.SugaredLogger
	Plex      PlexOptions
	OutputDir string
	Split     *tasks.SplitVideo
	Transcode *tasks.TranscodeVideo
}

type PlexOptions struct {
	Enabled   bool
	Episode   int
	MediaType string
	Name      string
	Season    int
	Year      int
}

func (p *Pipeline) Do(ctx context.Context, input string) ([]string, error) {
	var err error
	files := []string{input}

	// Split the input file into multiple files
	if p.Split != nil {
		files, err = p.Split.Do(context.TODO(), input)
		if err != nil {
			p.Logger.Errorw("error while splitting video", "err", err)
			return nil, err
		}
	}

	outputs := make([]string, 0)
	for _, file := range files {
		// Transcode each file from the split
		transcoded, err := p.Transcode.Do(context.TODO(), file)
		if err != nil {
			p.Logger.Errorw("error while transcoding video", "err", err)
			return nil, err
		}

		// Copy the output file
		output := p.getOutputPath(transcoded)
		if err := copyFile(transcoded, output); err != nil {
			p.Logger.Errorw("error while copying video to output dir", "err", err)
			return nil, err
		}
		outputs = append(outputs, output)
		p.Plex.Episode += 1
	}

	return outputs, nil
}

func (p *Pipeline) getOutputPath(name string) string {
	if !p.Plex.Enabled {
		// Just copy the result to the output dir with the same name
		return filepath.Join(p.OutputDir, filepath.Base(name))
	}

	// We have a little bit of extra work to do if we want plex naming
	outPath := filepath.Join(p.OutputDir, p.getPlexPath())
	if err := os.MkdirAll(filepath.Dir(outPath), 0750); err != nil {
		p.Logger.Errorw("error while creating plex output dir", "err", err)
		return ""
	}

	return outPath
}

func (p *Pipeline) getPlexPath() string {
	plex := p.Plex

	plexName := plex.Name
	if plex.Year > 0 {
		plexName = fmt.Sprintf("%s (%d)", plex.Name, plex.Year)
	}

	// TODO: Add validation somewhere to enforce checking for these
	switch plex.MediaType {
	case "movie":
		return fmt.Sprintf("%s/%s.mkv", plexName, plexName)
	case "tv":
		return fmt.Sprintf(
			"%s/Season %02d/%s - s%02de%02d.mkv",
			plexName,
			plex.Season,
			plexName,
			plex.Season,
			plex.Episode)
	}

	return "unknown.mkv"
}
