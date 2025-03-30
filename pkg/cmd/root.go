// Copyright 2019 Kube Capacity Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"fmt"
	"os"

	"github.com/robscott/kube-capacity/pkg/capacity"
	"github.com/spf13/cobra"
)

var opts capacity.Options

var rootCmd = &cobra.Command{
	Use:   "kube-capacity",
	Short: "kube-capacity provides an overview of the resource requests, limits, and utilization in a Kubernetes cluster.",
	Long:  "kube-capacity provides an overview of the resource requests, limits, and utilization in a Kubernetes cluster.",
	Run: func(cmd *cobra.Command, args []string) {
		if err := cmd.ParseFlags(args); err != nil {
			fmt.Printf("Error parsing flags: %v", err)
		}

		if err := validateOutputType(opts.OutputFormat); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		capacity.FetchAndPrint(opts)
	},
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&opts.ShowContainers,
		"containers", "c", false, "includes containers in output")
	rootCmd.PersistentFlags().BoolVarP(&opts.ShowPods,
		"pods", "p", false, "includes pods in output")
	rootCmd.PersistentFlags().BoolVarP(&opts.ShowUtil,
		"util", "u", false, "includes resource utilization in output")
	rootCmd.PersistentFlags().BoolVarP(&opts.ShowPodCount,
		"pod-count", "", false, "includes pod count per node in output")
	rootCmd.PersistentFlags().BoolVarP(&opts.AvailableFormat,
		"available", "a", false, "includes quantity available instead of percentage used")
	rootCmd.PersistentFlags().StringVarP(&opts.PodLabels,
		"pod-labels", "l", "", "labels to filter pods with")
	rootCmd.PersistentFlags().StringVarP(&opts.NodeLabels,
		"node-labels", "", "", "labels to filter nodes with")
	rootCmd.PersistentFlags().BoolVarP(&opts.ExcludeTainted,
		"no-taint", "", false, "exclude nodes with taints")
	rootCmd.PersistentFlags().StringVarP(&opts.NodeTaints,
		"node-taints", "t", "", "comma seperated list of taints to filter nodes with, prefix taint with '!' to filter out")
	rootCmd.PersistentFlags().StringVarP(&opts.NamespaceLabels,
		"namespace-labels", "", "", "labels to filter namespaces with")
	rootCmd.PersistentFlags().StringVarP(&opts.Namespace,
		"namespace", "n", "", "only include pods from this namespace")
	rootCmd.PersistentFlags().StringVarP(&opts.KubeContext,
		"context", "", "", "context to use for Kubernetes config")
	rootCmd.PersistentFlags().StringVarP(&opts.KubeConfig,
		"kubeconfig", "", "", "kubeconfig file to use for Kubernetes config")
	rootCmd.PersistentFlags().BoolVarP(&opts.InsecureSkipTLSVerify,
		"insecure-skip-tls-verify", "", false, "If true, the server's certificate will not be checked for validity. This will make your HTTPS connections insecure")
	rootCmd.PersistentFlags().StringVarP(&opts.SortBy,
		"sort", "", "name",
		fmt.Sprintf("attribute to sort results by (supports: %v)", capacity.SupportedSortAttributes))
	rootCmd.PersistentFlags().StringVarP(&opts.OutputFormat,
		"output", "o", capacity.TableOutput,
		fmt.Sprintf("output format for information (supports: %v)", capacity.SupportedOutputs()))
	rootCmd.PersistentFlags().StringVarP(&opts.ImpersonateUser,
		"as", "", "", "user to impersonate kube-capacity with")
	rootCmd.PersistentFlags().StringVarP(&opts.ImpersonateGroup,
		"as-group", "", "", "group to impersonate kube-capacity with")
	rootCmd.PersistentFlags().BoolVarP(&opts.HideRequests,
		"hide-requests", "", false, "hide requests from output")
	rootCmd.PersistentFlags().BoolVarP(&opts.HideLimits,
		"hide-limits", "", false, "hide limits from output")
	rootCmd.PersistentFlags().BoolVarP(&opts.ShowLabels,
		"show-labels", "", false, "includes node labels in output")
}

// Execute is the primary entrypoint for this CLI
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func validateOutputType(outputType string) error {
	for _, format := range capacity.SupportedOutputs() {
		if format == outputType {
			return nil
		}
	}
	return fmt.Errorf("Unsupported Output Type. We only support: %v", capacity.SupportedOutputs())
}
