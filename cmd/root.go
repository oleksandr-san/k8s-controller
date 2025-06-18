/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var logLevel string

func parseLogLevel(logLevel string) zerolog.Level {
	switch logLevel {
	case "trace":
		return zerolog.TraceLevel
	case "debug":
		return zerolog.DebugLevel
	case "info":
		return zerolog.InfoLevel
	case "warn":
		return zerolog.WarnLevel
	case "error":
		return zerolog.ErrorLevel
	default:
		return zerolog.InfoLevel
	}
}

func configureLogger(level zerolog.Level) {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	zerolog.SetGlobalLevel(level)

	if level == zerolog.TraceLevel {
		zerolog.CallerMarshalFunc = func(pc uintptr, file string, line int) string {
			return fmt.Sprintf("%s:%d", file, line)
		}
		zerolog.CallerFieldName = "caller"
		log.Logger = log.Logger.With().Caller().Logger()

		log.Logger = log.Output(zerolog.ConsoleWriter{
			Out:        os.Stderr,
			TimeFormat: "2006-01-02 15:04:05.000",
		})
	} else if level == zerolog.DebugLevel {
		zerolog.CallerMarshalFunc = func(pc uintptr, file string, line int) string {
			return fmt.Sprintf("%s:%d", file, line)
		}
		zerolog.CallerFieldName = "caller"
		log.Logger = log.Logger.With().Caller().Logger()
	} else {
		log.Logger = log.Output(os.Stderr)
	}
}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "k8s-controller",
	Short: "A brief description of your application",
	Long: `A longer description that spans multiple lines and likely contains
examples and usage of using your application. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	Run: func(cmd *cobra.Command, args []string) {
		configureLogger(parseLogLevel(logLevel))

		fmt.Println("Log level is set to:", logLevel)
		log.Trace().Msg("This is a trace log")
		log.Debug().Msg("This is a debug log")
		log.Info().Msg("This is an info log")
		log.Warn().Msg("This is a warn log")
		log.Error().Msg("This is an error log")
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
	rootCmd.PersistentFlags().StringVar(&logLevel, "log-level", "info", "Set log level: trace, info, warn, error")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
