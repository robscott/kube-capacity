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

package capacity

import (
	"fmt"
	"os"
	"text/tabwriter"
)

const (
	//TableOutput is the constant value for output type text
	TableOutput string = "table"
	//JSONOutput is the constant value for output type text
	JSONOutput string = "json"
)

// SupportedOutputs returns a string list of output formats supposed by this package
func SupportedOutputs() []string {
	return []string{
		TableOutput,
		JSONOutput,
	}
}

type printer interface {
	Print()
}

func printList(cm *clusterMetric, showPods bool, showUtil bool, output string) {
	p, err := printerFactory(cm, showPods, showUtil, output)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	p.Print()
}

func printerFactory(cm *clusterMetric, showPods bool, showUtil bool, outputType string) (printer, error) {
	var response printer
	switch outputType {
	case JSONOutput:
		response = jsonPrinter{
			cm:       cm,
			showPods: showPods,
			showUtil: showUtil,
		}
		return response, nil
	case TableOutput:
		response = tablePrinter{
			cm:       cm,
			showPods: showPods,
			showUtil: showUtil,
			w:        new(tabwriter.Writer),
		}
		return response, nil
	default:
		return response, fmt.Errorf("Called with an unsupported output type: %s", outputType)
	}
}
