package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var k8sApplyCmd = &cobra.Command{
	Use:   "apply -f path/to/manifest.yaml",
	Short: "Apply Kubernetes resources",
	Run: func(cmd *cobra.Command, args []string) {
		filenames, err := cmd.Flags().GetStringSlice("filename")
		if err != nil {
			log.Error().Err(err).Msg("failed to get filename flag")
			os.Exit(1)
		}
		dryRun, err := cmd.Flags().GetBool("dry-run")
		if err != nil {
			log.Error().Err(err).Msg("failed to get dry-run flag")
			os.Exit(1)
		}

		dynamicClient, err := makeDynamicClient(k8sConfigFlags)
		if err != nil {
			log.Error().Err(err).Msg("failed to create Kubernetes client")
			os.Exit(1)
		}
		objects, err := readObjects(filenames)
		if err != nil {
			log.Error().Err(err).Msg("failed to read objects")
			os.Exit(1)
		}
		options := metav1.ApplyOptions{
			FieldManager: "k8s-controller",
			Force:        true, // ok for create-or-replace
		}
		if dryRun {
			options.DryRun = []string{metav1.DryRunAll}
		}

		restMapper, err := k8sConfigFlags.ToRESTMapper()
		if err != nil {
			log.Error().Err(err).Msg("failed to create REST mapper")
			os.Exit(1)
		}

		for _, obj := range objects {
			name := obj.GetName()
			gvk := obj.GroupVersionKind()
			gvr := schema.GroupVersionResource{
				Group:    gvk.Group,
				Version:  gvk.Version,
				Resource: strings.ToLower(gvk.Kind),
			}
			gvr, err := restMapper.ResourceFor(gvr)
			if err != nil {
				log.Error().Err(err).Str("gvk", gvk.String()).Msg("failed to get resolve resource")
				os.Exit(1)
			}

			fmt.Println(name, gvr)
			resourceClient, err := makeResourceClient(k8sConfigFlags, dynamicClient, gvr, obj.GetNamespace())
			if err != nil {
				log.Error().Err(err).Msg("failed to create Kubernetes client")
				os.Exit(1)
			}
			appliedObj, err := resourceClient.Apply(context.Background(), name, obj, options)
			if err != nil {
				log.Error().Err(err).Msg("failed to apply resource")
				os.Exit(1)
			}
			log.Info().Str("name", name).Str("namespace", appliedObj.GetNamespace()).Str("gvk", gvk.String()).Msg("applied resource")
		}
	},
}

func init() {
	k8sAPICmd.AddCommand(k8sApplyCmd)

	f := k8sApplyCmd.Flags()
	f.Bool("dry-run", false, "Wether to submit server-side request without persisting the resource.")
	f.StringSliceP(
		"filename",
		"f",
		[]string{},
		"Paths to manifests to read (use '-' for stdin; may be repeated)",
	)
}
