package cmd

import (
	"context"
	"fmt"
	"github.com/neptune-media/robin/pkg/tasks"
	"github.com/spf13/pflag"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/yaml.v2"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	ARG_OUTPUT          = "output"
	ARG_PLEX            = "plex"
	ARG_PLEX_EPISODE    = "plex-episode"
	ARG_PLEX_MEDIA_TYPE = "plex-media-type"
	ARG_PLEX_NAME       = "plex-name"
	ARG_PLEX_SEASON     = "plex-season"
	ARG_PLEX_YEAR       = "plex-year"
	ARG_TEMPLATE        = "template"
	ARG_WORKDIR         = "work-dir"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "robin [input files...]",
	Args:  cobra.MinimumNArgs(1),
	Short: "A video splitting and transcoding pipeline",
	Long: `Robin is a video splitting and transcoding pipeline,
used to split multi-episode video files into single files, and
transcode the resulting files.

Optionally, the resulting files can also be renamed and stored in
a folder structure expected by Plex, to make adding files to the
media library a bit easier.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return bindFlags(cmd)
	},
	Run: func(cmd *cobra.Command, args []string) {
		// Setup logging
		baseLogger, _ := newLogger(zap.DebugLevel)
		defer baseLogger.Sync()
		logger := baseLogger.Sugar()
		logger.Infow("Starting robin...")

		// Create a temporary directory for storing intermediate files in
		tempDir, cleanup, err := createTaskDirectory()
		if err != nil {
			logger.Errorw("error while creating work dir", "err", err)
			return
		}
		defer cleanup(logger)

		// Create the output directory to store results in
		outputDir, err := createOutputDirectory()
		if err != nil {
			logger.Errorw("error while creating output dir", "err", err)
			return
		}

		// Setup the split video task
		splitVideo := &tasks.SplitVideo{
			Logger:  logger,
			WorkDir: tempDir,
		}

		// Setup the transcoding task
		transcodeVideo := &tasks.TranscodeVideo{
			Logger:  logger,
			WorkDir: tempDir,
		}
		if err := loadTemplates(transcodeVideo); err != nil {
			logger.Errorw("error while loading templates", "err", err)
			return
		}

		// Split the input file into multiple files
		files, err := splitVideo.Do(context.TODO(), args[0])
		if err != nil {
			logger.Errorw("error while splitting video", "err", err)
			return
		}

		for i, file := range files {
			// Transcode each file from the split
			transcoded, err := transcodeVideo.Do(context.TODO(), file)
			if err != nil {
				logger.Errorw("error while transcoding video", "err", err)
			}

			// Determine the file output name
			var output string
			episode := viper.GetInt(ARG_PLEX_EPISODE) + i
			if viper.GetBool(ARG_PLEX) {
				// We have a little bit of extra work to do if we want plex naming
				output = filepath.Join(outputDir, formatPlexName(episode))
				if err := os.MkdirAll(filepath.Dir(output), 0750); err != nil {
					logger.Errorw("error while creating plex output dir", "err", err)
					return
				}
			} else {
				// Just copy the result to the output dir with the same name
				output = filepath.Join(outputDir, filepath.Base(transcoded))
			}

			// Copy the output file
			if err := copyFile(transcoded, output); err != nil {
				logger.Errorw("error while copying video to output dir", "err", err)
				return
			}
		}
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))

	rootCmd.Flags().String(ARG_OUTPUT, "robin-output", "Specifies a folder to copy final output to")
	rootCmd.Flags().Bool(ARG_PLEX, false, "Enables renaming of output to plex recommendations")
	rootCmd.Flags().Int(ARG_PLEX_EPISODE, 1, "Starting episode number for plex tv shows")
	rootCmd.Flags().String(ARG_PLEX_MEDIA_TYPE, "", "Specifies if media is movie or tv show")
	rootCmd.Flags().String(ARG_PLEX_NAME, "", "Movie or TV Show name")
	rootCmd.Flags().Int(ARG_PLEX_SEASON, 1, "Season number for plex tv shows")
	rootCmd.Flags().Int(ARG_PLEX_YEAR, 0, "Year of the plex media item")
	rootCmd.Flags().StringArray(ARG_TEMPLATE, nil, "Specifies a path to a template file")
	rootCmd.Flags().String(ARG_WORKDIR, "", "Specifies a directory to use for scratch space")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	viper.SetEnvPrefix("robin")
	viper.AutomaticEnv() // read in environment variables that match
}

func bindFlags(cmd *cobra.Command) error {
	flagSets := []*pflag.FlagSet{
		cmd.PersistentFlags(),
		cmd.Flags(),
	}
	for _, flags := range flagSets {
		if err := viper.BindPFlags(flags); err != nil {
			return fmt.Errorf("error while binding flags: %s", err)
		}
	}

	return nil
}

func copyFile(sourceName, destName string) error {
	// Open source for reading
	in, err := os.Open(sourceName)
	if err != nil {
		return err
	}
	defer in.Close()

	// Open destination for writing
	out, err := os.Create(destName)
	if err != nil {
		return err
	}
	defer out.Close()

	// Allocate a 4k buffer
	buf := make([]byte, 4096)

	// Go!
	_, err = io.CopyBuffer(out, in, buf)
	return err
}

func createOutputDirectory() (string, error) {
	// Get the name of the output directory
	name := viper.GetString(ARG_OUTPUT)

	// Try to make the directory if needed
	if err := os.MkdirAll(name, 0750); err != nil {
		return "", err
	}

	return name, nil
}

func createTaskDirectory() (string, func(logger *zap.SugaredLogger), error) {
	dir, err := os.MkdirTemp(viper.GetString(ARG_WORKDIR), "robin-")
	if err != nil {
		return "", nil, err
	}
	f := func(logger *zap.SugaredLogger) {
		if err := os.RemoveAll(dir); err != nil {
			logger.Errorw("error while cleaning up temp directory",
				"err", err)
		}
	}

	return dir, f, nil
}

func formatPlexName(episode int) string {
	name := viper.GetString(ARG_PLEX_NAME)
	year := viper.GetInt(ARG_PLEX_YEAR)
	plexName := name
	if year > 0 {
		plexName = fmt.Sprintf("%s (%d)", name, year)
	}

	// TODO: Add validation somewhere to enforce checking for these
	switch viper.GetString(ARG_PLEX_MEDIA_TYPE) {
	case "movie":
		return fmt.Sprintf("%s/%s.mkv", plexName, plexName)
	case "tv":
		season := viper.GetInt(ARG_PLEX_SEASON)
		return fmt.Sprintf("%s/Season %02d/%s - s%02de%02d.mkv", plexName, season, plexName, season, episode)
	}

	return "unknown.mkv"
}

func loadTemplates(task *tasks.TranscodeVideo) error {
	paths := viper.GetStringSlice(ARG_TEMPLATE)

	for _, path := range paths {
		// Read template
		data, err := ioutil.ReadFile(path)
		if err != nil {
			return err
		}

		// Unpack into task options
		if err := yaml.Unmarshal(data, &task.Options); err != nil {
			return err
		}
	}

	return nil
}

func newLogger(logLevel zapcore.Level) (*zap.Logger, error) {
	cfg := zap.NewProductionConfig()
	cfg.Level = zap.NewAtomicLevelAt(logLevel)
	return cfg.Build()
}
