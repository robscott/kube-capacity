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
	"encoding/json"
	"fmt"

	"sigs.k8s.io/yaml"
)

type listNodeMetric struct {
	Name   string              `json:"name"`
	CPU    *listResourceOutput `json:"cpu,omitempty"`
	Memory *listResourceOutput `json:"memory,omitempty"`
	Pods   []*listPod          `json:"pods,omitempty"`
}

type listPod struct {
	Name      string              `json:"name"`
	Namespace string              `json:"namespace"`
	CPU       *listResourceOutput `json:"cpu"`
	Memory    *listResourceOutput `json:"memory"`
}

type listResourceOutput struct {
	Requests       string `json:"requests"`
	RequestsPct    string `json:"requests_pct"`
	Limits         string `json:"limits"`
	LimitsPct      string `json:"limits_pct"`
	Utilization    string `json:"utilization,omitempty"`
	UtilizationPct string `json:"utilization_pct,omitempty"`
}

type listClusterMetrics struct {
	Nodes         []*listNodeMetric `json:"nodes"`
	ClusterTotals struct {
		CPU    *listResourceOutput `json:"cpu"`
		Memory *listResourceOutput `json:"memory"`
	} `json:"cluster_totals"`
}

type listPrinter struct {
	cm       *clusterMetric
	showPods bool
	showUtil bool
}

func (lp listPrinter) Print(outputType string) {
	listOutput := lp.buildListClusterMetrics()

	jsonRaw, err := json.MarshalIndent(listOutput, "", "  ")
	if err != nil {
		fmt.Println("Error Marshalling JSON")
		fmt.Println(err)
	} else {
		if outputType == JSONOutput {
			fmt.Printf("%s", jsonRaw)
		} else {
			// This is a strange approach, but the k8s YAML package
			// already marshalls to JSON before converting to YAML,
			// this just allows us to follow the same code path.
			yamlRaw, err := yaml.JSONToYAML(jsonRaw)
			if err != nil {
				fmt.Println("Error Converting JSON to Yaml")
				fmt.Println(err)
			} else {
				fmt.Printf("%s", yamlRaw)
			}
		}
	}
}

func (lp *listPrinter) buildListClusterMetrics() listClusterMetrics {
	var response listClusterMetrics

	response.ClusterTotals.CPU = lp.buildListResourceOutput(lp.cm.cpu)
	response.ClusterTotals.Memory = lp.buildListResourceOutput(lp.cm.memory)

	for key, val := range lp.cm.nodeMetrics {
		var node listNodeMetric
		node.Name = key
		node.CPU = lp.buildListResourceOutput(val.cpu)
		node.Memory = lp.buildListResourceOutput(val.memory)
		if lp.showPods {
			for _, val := range val.podMetrics {
				var newNode listPod
				newNode.Name = val.name
				newNode.Namespace = val.namespace
				newNode.CPU = lp.buildListResourceOutput(val.cpu)
				newNode.Memory = lp.buildListResourceOutput(val.memory)
				node.Pods = append(node.Pods, &newNode)
			}
		}
		response.Nodes = append(response.Nodes, &node)
	}

	return response
}

func (lp *listPrinter) buildListResourceOutput(item *resourceMetric) *listResourceOutput {
	valueCalculator := item.valueFunction()
	percentCalculator := item.percentFunction()

	out := listResourceOutput{
		Requests:    valueCalculator(item.request),
		RequestsPct: percentCalculator(item.request),
		Limits:      valueCalculator(item.limit),
		LimitsPct:   percentCalculator(item.limit),
	}

	if lp.showUtil {
		out.Utilization = valueCalculator(item.utilization)
		out.UtilizationPct = percentCalculator(item.utilization)
	}
	return &out
}
