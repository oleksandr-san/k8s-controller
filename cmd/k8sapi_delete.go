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

var k8sDeleteCmd = &cobra.Command{
	Use:   "delete [resource[.version[.group]] name] [-f path/to/manifest.yaml]",
	Short: "Delete Kubernetes resources",
	Run: func(cmd *cobra.Command, args []string) {
		dryRun, err := cmd.Flags().GetBool("dry-run")
		if err != nil {
			log.Error().Err(err).Msg("failed to get dry-run flag")
			os.Exit(1)
		}
		filenames, err := cmd.Flags().GetStringSlice("filename")
		if err != nil {
			log.Error().Err(err).Msg("failed to get filename flag")
			os.Exit(1)
		}

		options := metav1.DeleteOptions{}
		if dryRun {
			options.DryRun = []string{metav1.DryRunAll}
		}
		dynamicClient, err := makeDynamicClient(k8sConfigFlags)
		if err != nil {
			log.Error().Err(err).Msg("failed to create Kubernetes client")
			os.Exit(1)
		}

		if len(args) > 1 {
			gvr, err := resolveGVR(k8sConfigFlags, args[0])
			if err != nil {
				log.Error().Err(err).Msg("failed to resolve GVR")
				os.Exit(1)
			}
			names := args[1:]
			if len(names) == 0 {
				log.Error().Msg("no resource names provided")
				os.Exit(1)
			}

			resourceClient, err := makeResourceClient(k8sConfigFlags, dynamicClient, gvr, "")
			if err != nil {
				log.Error().Err(err).Msg("failed to create Kubernetes client")
				os.Exit(1)
			}

			for _, name := range names {
				err := resourceClient.Delete(context.Background(), name, options)
				if err != nil {
					log.Error().Err(err).Msg("dailed to delete resources")
					os.Exit(1)
				}
				log.Info().Str("name", name).Msg("deleted resource")
			}
		}

		if len(filenames) > 0 {
			objects, err := readObjects(filenames)
			if err != nil {
				log.Error().Err(err).Msg("failed to read objects")
				os.Exit(1)
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
				err = resourceClient.Delete(context.Background(), name, options)
				if err != nil {
					log.Error().Err(err).Msg("failed to apply resource")
					os.Exit(1)
				}
				log.Info().Str("name", name).Str("gvk", gvk.String()).Msg("deleted resource")
			}
		}
	},
}

func init() {
	k8sAPICmd.AddCommand(k8sDeleteCmd)

	f := k8sDeleteCmd.Flags()
	f.Bool("dry-run", false, "Wether to submit server-side request without persisting the resource.")
	f.StringSliceP(
		"filename",
		"f",
		[]string{},
		"Paths to manifests to read (use '-' for stdin; may be repeated)",
	)
}
