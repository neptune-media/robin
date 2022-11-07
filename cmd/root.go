package cmd

import (
	"context"
	"fmt"
	"github.com/neptune-media/robin/pkg/pipeline"
	"github.com/neptune-media/robin/pkg/tasks"
	"github.com/spf13/pflag"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
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
	ARG_SPLIT           = "split"
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

		pipe := &pipeline.Pipeline{
			Logger: logger,
			Plex: pipeline.PlexOptions{
				Enabled:   viper.GetBool(ARG_PLEX),
				Episode:   viper.GetInt(ARG_PLEX_EPISODE),
				MediaType: viper.GetString(ARG_PLEX_MEDIA_TYPE),
				Name:      viper.GetString(ARG_PLEX_NAME),
				Season:    viper.GetInt(ARG_PLEX_SEASON),
				Year:      viper.GetInt(ARG_PLEX_YEAR),
			},
			OutputDir: outputDir,
		}

		if viper.GetBool(ARG_SPLIT) {
			// Setup the split video task
			pipe.Split = &tasks.SplitVideo{
				Logger:  logger,
				WorkDir: tempDir,
			}
		}

		// Setup the transcoding task
		pipe.Transcode = &tasks.TranscodeVideo{
			Logger:  logger,
			WorkDir: tempDir,
		}
		if err := loadTemplates(pipe.Transcode); err != nil {
			logger.Errorw("error while loading templates", "err", err)
			return
		}

		for _, input := range args {
			if _, err := pipe.Do(context.TODO(), input); err != nil {
				logger.Errorw("error while running pipeline", "err", err)
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
	rootCmd.Flags().Bool(ARG_SPLIT, false, "Enables multi-episode file splitting before transcoding")
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
