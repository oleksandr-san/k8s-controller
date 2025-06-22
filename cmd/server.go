package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/oleksandr-san/k8s-controller/pkg/informer"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/valyala/fasthttp"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	requestIDKey = "requestID"
)

func loggingMiddleware(next fasthttp.RequestHandler) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		start := time.Now()

		requestID := string(ctx.Request.Header.Peek("X-Request-ID"))
		ctx.SetUserValue(requestIDKey, requestID)
		if requestID == "" {
			requestID = uuid.New().String()
			ctx.Response.Header.Set("X-Request-ID", requestID)
		}

		next(ctx)

		duration := time.Since(start)
		log.Debug().
			Str("method", string(ctx.Method())).
			Str("path", string(ctx.Path())).
			Str("remote_ip", ctx.RemoteIP().String()).
			Int("status", ctx.Response.StatusCode()).
			Dur("latency", duration).
			Str("request_id", requestID).
			Msg("HTTP request")
	}
}

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Start a FastHTTP server",
	Run: func(cmd *cobra.Command, args []string) {
		configureLogger(parseLogLevel(viper.GetString("log.level")))

		clientset, err := getServerKubeClient(viper.GetString("kubeconfig"), viper.GetBool("in-cluster"))
		if err != nil {
			log.Error().Err(err).Msg("Failed to create Kubernetes client")
			os.Exit(1)
		}
		ctx := context.Background()
		go informer.StartDeploymentInformer(ctx, clientset)

		handler := func(ctx *fasthttp.RequestCtx) {
			ctx.SetStatusCode(fasthttp.StatusOK)
			ctx.SetBodyString("Welcome to the FastHTTP server!")
		}

		serverPort := viper.GetInt("app.port")
		addr := fmt.Sprintf(":%d", serverPort)
		log.Info().Msgf("Starting FastHTTP server on %s", addr)
		if err := fasthttp.ListenAndServe(addr, loggingMiddleware(handler)); err != nil {
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

	f.String("kubeconfig", "~/.kube/config", "Path to the kubeconfig file")
	viper.BindPFlag("kubeconfig", f.Lookup("kubeconfig"))

	f.Bool("in-cluster", false, "Use in-cluster Kubernetes config")
	viper.BindPFlag("in-cluster", f.Lookup("in-cluster"))
}

func getServerKubeClient(kubeconfigPath string, inCluster bool) (*kubernetes.Clientset, error) {
	var config *rest.Config
	var err error
	if inCluster {
		config, err = rest.InClusterConfig()
	} else {
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	}
	if err != nil {
		return nil, err
	}
	return kubernetes.NewForConfig(config)
}
