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

package capacity

import (
	"fmt"
	"os"
	"text/tabwriter"
)

const (
	//TableOutput is the constant value for output type table
	TableOutput string = "table"
	//JSONOutput is the constant value for output type JSON
	JSONOutput string = "json"
	//YAMLOutput is the constant value for output type YAML
	YAMLOutput string = "yaml"
)

// SupportedOutputs returns a string list of output formats supposed by this package
func SupportedOutputs() []string {
	return []string{
		TableOutput,
		JSONOutput,
		YAMLOutput,
	}
}

func printList(cm *clusterMetric, showContainers, showPods, showUtil bool, output, sortBy string, availableFormat bool) {
	if output == JSONOutput || output == YAMLOutput {
		lp := &listPrinter{
			cm:             cm,
			showPods:       showPods,
			showUtil:       showUtil,
			showContainers: showContainers,
			sortBy:         sortBy,
		}
		lp.Print(output)
	} else if output == TableOutput {
		tp := &tablePrinter{
			cm:              cm,
			showPods:        showPods,
			showUtil:        showUtil,
			showContainers:  showContainers,
			sortBy:          sortBy,
			w:               new(tabwriter.Writer),
			availableFormat: availableFormat,
		}
		tp.Print()
	} else {
		fmt.Printf("Called with an unsupported output type: %s", output)
		os.Exit(1)
	}
}
