package cmd

import (
	"fmt"
	"os"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/valyala/fasthttp"
)

const (
	requestIDKey = "requestID"
)

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Start a FastHTTP server",
	Run: func(cmd *cobra.Command, args []string) {
		handler := func(ctx *fasthttp.RequestCtx) {
			requestID := uuid.New().String()
			ctx.SetUserValue(requestIDKey, requestID)

			log.Info().
				Str("requestID", requestID).
				Str("method", string(ctx.Method())).
				Str("path", string(ctx.Path())).
				Str("query", string(ctx.QueryArgs().String())).
				Msg("received request")
		}

		serverPort := viper.GetInt("app.port")
		addr := fmt.Sprintf(":%d", serverPort)
		log.Info().Msgf("Starting FastHTTP server on %s", addr)
		if err := fasthttp.ListenAndServe(addr, handler); err != nil {
			log.Error().Err(err).Msg("Error starting FastHTTP server")
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(serverCmd)

	f := serverCmd.Flags()
	f.Int("port", 8080, "Port to run the server on")
	viper.BindPFlag("app.port", f.Lookup("port"))
}
