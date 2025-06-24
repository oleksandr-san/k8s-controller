package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/oleksandr-san/k8s-controller/pkg/ctrl"
	"github.com/oleksandr-san/k8s-controller/pkg/informer"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/valyala/fasthttp"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	ctrlruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/manager"
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

type server struct {
	mi     *informer.MultiInformer
	mapper meta.RESTMapper
}

type resourceReference struct {
	gvr       schema.GroupVersionResource
	namespace string
	name      string
}

func (srv *server) parseResourceReference(path []byte) (resourceReference, error) {
	parts := bytes.Split(path[1:], []byte("/")) // Skip leading slash and split
	if 1 > len(parts) || len(parts) > 3 {
		return resourceReference{}, fmt.Errorf("invalid path format: expected /resource[/namespace[/name]], got %s", path)
	}

	resourceParts := bytes.Split(parts[0], []byte("."))
	guess := schema.GroupVersionResource{}

	switch len(resourceParts) {
	case 1:
		guess.Resource = string(resourceParts[0])
	case 2:
		guess.Resource = string(resourceParts[0])
		guess.Group = string(resourceParts[1])
	case 3:
		guess.Resource = string(resourceParts[0])
		guess.Version = string(resourceParts[1])
		guess.Group = string(resourceParts[2])
	default:
		return resourceReference{}, fmt.Errorf("invalid resource format: %s", parts[0])
	}

	full, err := srv.mapper.ResourceFor(guess)
	if err != nil {
		return resourceReference{}, fmt.Errorf("failed to get REST mapping: %w", err)
	}

	ref := resourceReference{
		gvr: full,
	}
	if len(parts) > 1 {
		ref.namespace = string(parts[1])
	}
	if len(parts) > 2 {
		ref.name = string(parts[2])
	}

	return ref, nil
}

func (srv *server) writeError(ctx *fasthttp.RequestCtx, statusCode int, err error) {
	log.Error().Err(err).Msg("error handling request")

	srv.writeResponse(ctx, map[string]string{
		"error": err.Error(),
	}, statusCode)
}

func (srv *server) writeResponse(ctx *fasthttp.RequestCtx, obj any, statusCode int) {
	buf := new(bytes.Buffer)
	err := json.NewEncoder(buf).Encode(obj)

	if err != nil {
		log.Error().Err(err).Msg("failed to encode error response")
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		ctx.SetBodyString(err.Error())
		return
	}

	ctx.SetStatusCode(statusCode)
	ctx.SetContentType("application/json")
	ctx.SetBody(buf.Bytes())
}

func (srv *server) handleRequest(ctx *fasthttp.RequestCtx) {
	path := ctx.Path()

	if bytes.HasPrefix(path, []byte("/api")) {
		ref, err := srv.parseResourceReference(path[4:])
		fmt.Println("API request", path[4:], ref)
		if err != nil {
			srv.writeError(ctx, fasthttp.StatusBadRequest, err)
			return
		}

		indexer := srv.mi.GetIndexer(ref.gvr)
		if indexer == nil {
			srv.writeError(ctx, fasthttp.StatusNotFound, fmt.Errorf("resource not found: %s", ref.gvr))
			return
		}

		if ref.name == "" && ref.namespace == "" {
			objs := indexer.List()
			srv.writeResponse(ctx, objs, fasthttp.StatusOK)
		} else if ref.name == "" && ref.namespace != "" {
			objs, err := indexer.ByIndex(cache.NamespaceIndex, ref.namespace)
			if err != nil {
				srv.writeError(ctx, fasthttp.StatusInternalServerError, err)
			}
			srv.writeResponse(ctx, objs, fasthttp.StatusOK)
		} else {
			obj, exists, err := indexer.GetByKey(ref.namespace + "/" + ref.name)
			if err != nil {
				srv.writeError(ctx, fasthttp.StatusInternalServerError, err)
			}
			if !exists {
				srv.writeError(ctx, fasthttp.StatusNotFound, fmt.Errorf("object %s/%s not found", ref.namespace, ref.name))
			}
			srv.writeResponse(ctx, obj, fasthttp.StatusOK)
		}

		ctx.SetStatusCode(fasthttp.StatusOK)
	} else {
		ctx.SetStatusCode(fasthttp.StatusOK)
		ctx.SetBodyString("OK")
	}
}

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Start a FastHTTP server",
	Run: func(cmd *cobra.Command, args []string) {
		configureLogger(parseLogLevel(viper.GetString("log.level")))

		resyncPeriod, err := time.ParseDuration(viper.GetString("app.resync-period"))
		if err != nil {
			log.Error().Err(err).Msg("failed to parse resync period")
			os.Exit(1)
		}

		config, err := getKubeConfig(viper.GetString("kubeconfig"), viper.GetBool("in-cluster"))
		if err != nil {
			log.Error().Err(err).Msg("failed to create Kubernetes client")
			os.Exit(1)
		}
		dc, err := discovery.NewDiscoveryClientForConfig(config)
		if err != nil {
			log.Error().Err(err).Msg("failed to create disovery client")
			os.Exit(1)
		}

		cached := memory.NewMemCacheClient(dc)
		mapper := restmapper.NewShortcutExpander(
			restmapper.NewDeferredDiscoveryRESTMapper(cached),
			cached,
			nil,
		)

		resources := viper.GetStringSlice("resources")
		if len(resources) == 0 {
			log.Error().Msg("no resources specified to watch")
			os.Exit(1)
		}
		gvrs, err := resolveGVRs(mapper, resources...)
		if err != nil {
			log.Error().Err(err).Msg("failed to resolve GVRs")
			os.Exit(1)
		}

		log.Info().Strs("resources", resources).Msg("start multi-informer")
		multiInformer, err := informer.NewMultiInformer(
			config,
			resyncPeriod,
			gvrs,
			viper.GetString("namespace"),
			nil,
		)
		if err != nil {
			log.Error().Err(err).Msg("failed to create informer")
			os.Exit(1)
		}
		ctx := context.Background()
		go multiInformer.Start(ctx)

		// Start controller-runtime manager and controller
		mgr, err := ctrlruntime.NewManager(config, manager.Options{})
		if err != nil {
			log.Error().Err(err).Msg("Failed to create controller-runtime manager")
			os.Exit(1)
		}
		if err := ctrl.AddDeploymentController(mgr); err != nil {
			log.Error().Err(err).Msg("Failed to add deployment controller")
			os.Exit(1)
		}
		go func() {
			log.Info().Msg("Starting controller-runtime manager...")
			if err := mgr.Start(cmd.Context()); err != nil {
				log.Error().Err(err).Msg("Manager exited with error")
				os.Exit(1)
			}
		}()

		srv := &server{
			mi:     multiInformer,
			mapper: mapper,
		}
		serverPort := viper.GetInt("app.port")
		addr := fmt.Sprintf(":%d", serverPort)
		log.Info().Msgf("Starting FastHTTP server on %s", addr)
		if err := fasthttp.ListenAndServe(addr, loggingMiddleware(srv.handleRequest)); err != nil {
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

	f.String("namespace", metav1.NamespaceAll, "Namespace to watch")
	viper.BindPFlag("namespace", f.Lookup("namespace"))

	f.StringSlice("resources", []string{"deployments"}, "Resources to watch")
	viper.BindPFlag("resources", f.Lookup("resources"))

	f.String("resync", "30s", "Resync period")
	viper.BindPFlag("app.resync-period", f.Lookup("resync"))
}

func getKubeConfig(kubeconfigPath string, inCluster bool) (*rest.Config, error) {
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
	return config, nil
}
