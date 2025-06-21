package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/scheme"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/printers"
	"k8s.io/client-go/dynamic"
)

var k8sListCmd = &cobra.Command{
	Use:   "list resource[.version][.group.com]",
	Short: "List Kubernetes deployments in the default namespace",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			log.Error().Msg("No resource type provided")
			os.Exit(1)
		}

		kubeconfigPath := viper.GetString("kubeconfig")
		k8sConfigFlags.KubeConfig = &kubeconfigPath
		kubeConfig, err := k8sConfigFlags.ToRESTConfig()
		if err != nil {
			log.Error().Err(err).Msg("failed to read Kubernetes config")
			os.Exit(1)
		}

		gvr, err := resolveGVR(k8sConfigFlags, args[0])
		if err != nil {
			log.Error().Err(err).Msg("failed to resolve GVR")
			os.Exit(1)
		}

		dynamicClient, err := dynamic.NewForConfig(kubeConfig)
		if err != nil {
			log.Error().Err(err).Msg("Failed to create Kubernetes client")
			os.Exit(1)
		}

		namespace := ""
		if k8sConfigFlags.Namespace != nil {
			namespace = *k8sConfigFlags.Namespace
		}

		var resourceClient dynamic.ResourceInterface
		if namespace != "" {
			resourceClient = dynamicClient.Resource(gvr).Namespace(namespace)
		} else {
			resourceClient = dynamicClient.Resource(gvr)
		}

		resources, err := resourceClient.List(context.Background(), metav1.ListOptions{})
		if err != nil {
			log.Error().Err(err).Msg("Failed to list deployments")
			os.Exit(1)
		}

		if namespace != "" {
			fmt.Printf("Found %d %s in '%s' namespace:\n", len(resources.Items), gvr.Resource, namespace)
		} else {
			fmt.Printf("Found %d %s in all namespaces:\n", len(resources.Items), gvr.Resource)
		}

		var printr printers.ResourcePrinter
		outputFormat, _ := cmd.Flags().GetString("output")
		switch strings.ToLower(outputFormat) {
		case "json":
			printr = printers.NewTypeSetter(scheme.Scheme).ToPrinter(&printers.JSONPrinter{})
		case "yaml":
			printr = printers.NewTypeSetter(scheme.Scheme).ToPrinter(&printers.YAMLPrinter{})
		case "name":
			printr = printers.NewTypeSetter(scheme.Scheme).ToPrinter(&printers.NamePrinter{})
		case "table":
			printr = printers.NewTablePrinter(printers.PrintOptions{
				Wide:          true,
				WithNamespace: true,
				WithKind:      true,
				ShowLabels:    true,
			})
		default:
			log.Error().Msgf("Unknown output format: %s", outputFormat)
			os.Exit(1)
		}

		for _, obj := range resources.Items {
			printr.PrintObj(&obj, os.Stdout)
		}
	},
}

func init() {
	k8sAPICmd.AddCommand(k8sListCmd)

	f := k8sListCmd.Flags()
	f.StringP("output", "o", "name", "Output format. One of: json|yaml|name|table")
}
