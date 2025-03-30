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
	//CSVOutput is the constant value for output type csv
	CSVOutput string = "csv"
	//TSVOutput is the constant value for output type csv
	TSVOutput string = "tsv"
	//JSONOutput is the constant value for output type JSON
	JSONOutput string = "json"
	//YAMLOutput is the constant value for output type YAML
	YAMLOutput string = "yaml"
)

// SupportedOutputs returns a string list of output formats supposed by this package
func SupportedOutputs() []string {
	return []string{
		TableOutput,
		CSVOutput,
		TSVOutput,
		JSONOutput,
		YAMLOutput,
	}
}

func printList(cm *clusterMetric, opts Options) {
	output := opts.OutputFormat
	if output == JSONOutput || output == YAMLOutput {
		lp := &listPrinter{
			cm:   cm,
			opts: opts,
		}
		lp.Print(output)
	} else if output == TableOutput {
		tp := &tablePrinter{
			cm:   cm,
			w:    new(tabwriter.Writer),
			opts: opts,
		}
		if !tp.hasVisibleColumns() {
			fmt.Fprintln(os.Stderr, "Error: No data columns selected for display. At least one of the following must be enabled:")
			fmt.Fprintln(os.Stderr, "- Resource requests (enabled by default, disabled with --hide-requests)")
			fmt.Fprintln(os.Stderr, "- Resource limits (enabled by default, disabled with --hide-limits)")
			fmt.Fprintln(os.Stderr, "- Resource utilization (enabled with --util)")
			fmt.Fprintln(os.Stderr, "- Pod count (enabled with --pod-count)")
			fmt.Fprintln(os.Stderr, "- Node labels (enabled with --show-labels)")
			os.Exit(1)
		}
		tp.Print()
	} else if output == CSVOutput || output == TSVOutput {
		cp := &csvPrinter{
			cm:   cm,
			opts: opts,
		}
		cp.Print(output)
	} else {
		fmt.Fprintf(os.Stderr, "Called with an unsupported output type: %s\n", output)
		os.Exit(1)
	}
}
