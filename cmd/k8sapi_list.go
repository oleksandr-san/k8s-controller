package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/scheme"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/printers"
)

var k8sListCmd = &cobra.Command{
	Use:   "list resource[.version][.group.com]",
	Short: "List Kubernetes deployments in the default namespace",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			log.Error().Msg("No resource type provided")
			os.Exit(1)
		}

		gvr, err := resolveGVR(k8sConfigFlags, args[0])
		if err != nil {
			log.Error().Err(err).Msg("failed to resolve GVR")
			os.Exit(1)
		}

		resourceClient, err := makeResourceClient(k8sConfigFlags, nil, gvr, "")
		if err != nil {
			log.Error().Err(err).Msg("failed to create Kubernetes client")
			os.Exit(1)
		}

		resources, err := resourceClient.List(context.Background(), metav1.ListOptions{})
		if err != nil {
			log.Error().Err(err).Msg("Failed to list deployments")
			os.Exit(1)
		}

		if k8sConfigFlags.Namespace != nil && *k8sConfigFlags.Namespace != "" {
			fmt.Printf("Found %d %s in '%s' namespace:\n", len(resources.Items), gvr.Resource, *k8sConfigFlags.Namespace)
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
				ShowLabels:    false,
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
