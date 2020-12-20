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

var showContainers bool
var showPods bool
var showUtil bool
var podLabels string
var nodeLabels string
var namespaceLabels string
var kubeContext string
var outputFormat string
var sortBy string
var availableFormat bool

var rootCmd = &cobra.Command{
	Use:   "kube-capacity",
	Short: "kube-capacity provides an overview of the resource requests, limits, and utilization in a Kubernetes cluster.",
	Long:  "kube-capacity provides an overview of the resource requests, limits, and utilization in a Kubernetes cluster.",
	Run: func(cmd *cobra.Command, args []string) {
		if err := cmd.ParseFlags(args); err != nil {
			fmt.Printf("Error parsing flags: %v", err)
		}

		if err := validateOutputType(outputFormat); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		capacity.FetchAndPrint(showContainers, showPods, showUtil, availableFormat, podLabels, nodeLabels, namespaceLabels, kubeContext, outputFormat, sortBy)
	},
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&showContainers,
		"containers", "c", false, "includes containers in output")
	rootCmd.PersistentFlags().BoolVarP(&showPods,
		"pods", "p", false, "includes pods in output")
	rootCmd.PersistentFlags().BoolVarP(&showUtil,
		"util", "u", false, "includes resource utilization in output")
	rootCmd.PersistentFlags().BoolVarP(&availableFormat,
		"available", "a", false, "formats the output in terms of available instead of percentage")
	rootCmd.PersistentFlags().StringVarP(&podLabels,
		"pod-labels", "l", "", "labels to filter pods with")
	rootCmd.PersistentFlags().StringVarP(&nodeLabels,
		"node-labels", "", "", "labels to filter nodes with")
	rootCmd.PersistentFlags().StringVarP(&namespaceLabels,
		"namespace-labels", "n", "", "labels to filter namespaces with")
	rootCmd.PersistentFlags().StringVarP(&kubeContext,
		"context", "", "", "context to use for Kubernetes config")
	rootCmd.PersistentFlags().StringVarP(&sortBy,
		"sort", "", "name",
		fmt.Sprintf("attribute to sort results be (supports: %v)", capacity.SupportedSortAttributes))

	rootCmd.PersistentFlags().StringVarP(&outputFormat,
		"output", "o", capacity.TableOutput,
		fmt.Sprintf("output format for information (supports: %v)", capacity.SupportedOutputs()))
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
