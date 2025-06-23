package cmd

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	yamlutil "k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/dynamic"
)

var k8sConfigFlags *genericclioptions.ConfigFlags

var k8sAPICmd = &cobra.Command{
	Use:   "k8sapi",
	Short: "Interact with Kubernetes API",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Error: must also specify a command like list, create, delete, etc.")
		os.Exit(1)
	},
}

func init() {
	defaultKubeConfig := "~/.kube/config"

	k8sConfigFlags = genericclioptions.NewConfigFlags(true)
	k8sConfigFlags.KubeConfig = &defaultKubeConfig

	pf := k8sAPICmd.PersistentFlags()
	k8sConfigFlags.AddFlags(pf)
	viper.BindPFlag("kubeconfig", pf.Lookup("kubeconfig"))

	rootCmd.AddCommand(k8sAPICmd)
}

// resolveGVR turns "deploy", "deployments", "deploy.apps", "deployments.v1.apps", etc.
// into a concrete GroupVersionResource, using live discovery data so it
// works for CRDs as well.
func resolveGVRs(mapper meta.RESTMapper, tokens ...string) (gvrs []schema.GroupVersionResource, err error) {
	for _, token := range tokens {
		var guess schema.GroupVersionResource
		if parts := strings.Split(token, "."); len(parts) > 1 {
			// token like "deployments.v1.apps"
			guess.Resource = parts[0]
			guess.Version = parts[1]
			if len(parts) > 2 {
				guess.Group = strings.Join(parts[2:], ".")
			}
		} else {
			guess.Resource = token
		}

		full, err := mapper.ResourceFor(guess)
		if meta.IsNoMatchError(err) {
			return gvrs, fmt.Errorf("unknown resource %q", token)
		}
		gvrs = append(gvrs, full)
	}

	return gvrs, nil
}

func makeDynamicClient(flags *genericclioptions.ConfigFlags) (*dynamic.DynamicClient, error) {
	kubeconfigPath := viper.GetString("kubeconfig")
	flags.KubeConfig = &kubeconfigPath
	kubeConfig, err := flags.ToRESTConfig()
	if err != nil {
		return nil, err
	}

	dynamicClient, err := dynamic.NewForConfig(kubeConfig)
	if err != nil {
		return nil, err
	}

	return dynamicClient, nil
}

func makeResourceClient(
	flags *genericclioptions.ConfigFlags,
	dynamicClient *dynamic.DynamicClient,
	gvr schema.GroupVersionResource,
	namespace string,
) (dynamic.ResourceInterface, error) {
	if dynamicClient == nil {
		var err error
		dynamicClient, err = makeDynamicClient(flags)
		if err != nil {
			return nil, err
		}
	}

	if flags.Namespace != nil && *flags.Namespace != "" {
		namespace = *flags.Namespace
	}

	if namespace != "" {
		return dynamicClient.Resource(gvr).Namespace(namespace), nil
	} else {
		return dynamicClient.Resource(gvr), nil
	}
}

// Buffer size that comfortably fits a typical manifest header.
const bufSize = 4096

// readObjects takes a list of paths (use "-" for stdin) and returns all parsed K8s objects.
func readObjects(paths []string) ([]*unstructured.Unstructured, error) {
	var out []*unstructured.Unstructured

	// default to stdin if no -f was given — same UX as kubectl
	if len(paths) == 0 {
		paths = []string{"-"}
	}

	for _, p := range paths {
		var r io.ReadCloser
		switch p {
		case "-":
			r = io.NopCloser(os.Stdin)
		default:
			f, err := os.Open(p)
			if err != nil {
				return nil, fmt.Errorf("open %s: %w", p, err)
			}
			defer f.Close()
			r = f
		}

		d := yamlutil.NewYAMLOrJSONDecoder(r, bufSize) // streams & splits docs :contentReference[oaicite:4]{index=4}
		for {
			u := &unstructured.Unstructured{}
			err := d.Decode(u)
			switch {
			case err == io.EOF: // end of this reader
				goto nextSource
			case err != nil:
				return nil, fmt.Errorf("decode %s: %w", p, err)
			}
			// Empty document (possible if file ends with '---\n')
			if len(u.Object) == 0 {
				continue
			}
			out = append(out, u)
		}
	nextSource:
		r.Close()
	}
	return out, nil
}
