package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/cli-runtime/pkg/genericclioptions"
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

// ResolveGVR turns "deploy", "deployments", "deploy.apps", "deployments.v1.apps", etc.
// into a concrete GroupVersionResource, using live discovery data so it
// works for CRDs as well.
func resolveGVR(flags *genericclioptions.ConfigFlags, token string) (schema.GroupVersionResource, error) {
	rm, err := flags.ToRESTMapper()
	if err != nil {
		return schema.GroupVersionResource{}, fmt.Errorf("create mapper: %w", err)
	}

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

	full, err := rm.ResourceFor(guess)
	if meta.IsNoMatchError(err) {
		return schema.GroupVersionResource{}, fmt.Errorf("unknown resource %q", token)
	}
	return full, err
}
