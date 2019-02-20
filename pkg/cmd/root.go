// Copyright 2019 Rob Scott
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

var showPods bool
var showUtil bool
var podLabels string
var nodeLabels string
var namespaceLabels string

var rootCmd = &cobra.Command{
	Use:   "kube-capacity",
	Short: "kube-capacity provides an overview of the resource requests, limits, and utilization in a Kubernetes cluster",
	Long:  "kube-capacity provides an overview of the resource requests, limits, and utilization in a Kubernetes cluster",
	Run: func(cmd *cobra.Command, args []string) {
		if err := cmd.ParseFlags(args); err != nil {
			fmt.Printf("Error parsing flags: %v", err)
		}

		capacity.List(showPods, showUtil, podLabels, nodeLabels, namespaceLabels)
	},
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&showPods, "pods", "p", false, "Set this flag to include pods in output")
	rootCmd.PersistentFlags().BoolVarP(&showUtil, "util", "u", false, "Set this flag to include resource utilization in output")
	rootCmd.PersistentFlags().StringVarP(&podLabels, "pod-labels", "l", "", "Labels to filter pods with.")
	rootCmd.PersistentFlags().StringVarP(&nodeLabels, "node-labels", "", "", "Labels to filter nodes with.")
	rootCmd.PersistentFlags().StringVarP(&namespaceLabels, "namespace-labels", "n", "", "Labels to filter namespaces with.")
}

// Execute is the primary entrypoint for this CLI
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
