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
	Name     string              `json:"name"`
	Labels   map[string]string   `json:"labels,omitempty"`
	CPU      *listResourceOutput `json:"cpu,omitempty"`
	Memory   *listResourceOutput `json:"memory,omitempty"`
	Pods     []*listPod          `json:"pods,omitempty"`
	PodCount string              `json:"podCount,omitempty"`
}

type listPod struct {
	Name       string              `json:"name"`
	Namespace  string              `json:"namespace"`
	CPU        *listResourceOutput `json:"cpu"`
	Memory     *listResourceOutput `json:"memory"`
	Containers []listContainer     `json:"containers,omitempty"`
}

type listContainer struct {
	Name   string              `json:"name"`
	CPU    *listResourceOutput `json:"cpu"`
	Memory *listResourceOutput `json:"memory"`
}

type listResourceOutput struct {
	Requests       string `json:"requests,omitempty"`
	RequestsPct    string `json:"requestsPercent,omitempty"`
	Limits         string `json:"limits,omitempty"`
	LimitsPct      string `json:"limitsPercent,omitempty"`
	Utilization    string `json:"utilization,omitempty"`
	UtilizationPct string `json:"utilizationPercent,omitempty"`
}

type listClusterMetrics struct {
	Nodes         []*listNodeMetric  `json:"nodes"`
	ClusterTotals *listClusterTotals `json:"clusterTotals"`
}

type listClusterTotals struct {
	CPU      *listResourceOutput `json:"cpu"`
	Memory   *listResourceOutput `json:"memory"`
	PodCount string              `json:"podCount,omitempty"`
}

type listPrinter struct {
	cm   *clusterMetric
	opts Options
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

	response.ClusterTotals = &listClusterTotals{
		CPU:    lp.buildListResourceOutput(lp.cm.cpu),
		Memory: lp.buildListResourceOutput(lp.cm.memory),
	}

	if lp.opts.ShowPodCount {
		response.ClusterTotals.PodCount = lp.cm.podCount.podCountString()
	}

	for _, nodeMetric := range lp.cm.getSortedNodeMetrics(lp.opts.SortBy) {
		var node listNodeMetric
		node.Name = nodeMetric.name
		node.CPU = lp.buildListResourceOutput(nodeMetric.cpu)
		node.Memory = lp.buildListResourceOutput(nodeMetric.memory)

		if lp.opts.ShowPodCount {
			node.PodCount = nodeMetric.podCount.podCountString()
		}

		if lp.opts.ShowLabels {
			node.Labels = nodeMetric.labels
		}

		if lp.opts.ShowPods || lp.opts.ShowContainers {
			for _, podMetric := range nodeMetric.getSortedPodMetrics(lp.opts.SortBy) {
				var pod listPod
				pod.Name = podMetric.name
				pod.Namespace = podMetric.namespace
				pod.CPU = lp.buildListResourceOutput(podMetric.cpu)
				pod.Memory = lp.buildListResourceOutput(podMetric.memory)

				if lp.opts.ShowContainers {
					for _, containerMetric := range podMetric.getSortedContainerMetrics(lp.opts.SortBy) {
						pod.Containers = append(pod.Containers, listContainer{
							Name:   containerMetric.name,
							Memory: lp.buildListResourceOutput(containerMetric.memory),
							CPU:    lp.buildListResourceOutput(containerMetric.cpu),
						})
					}
				}
				node.Pods = append(node.Pods, &pod)
			}
		}
		response.Nodes = append(response.Nodes, &node)
	}

	return response
}

func (lp *listPrinter) buildListResourceOutput(item *resourceMetric) *listResourceOutput {
	valueCalculator := item.valueFunction()
	percentCalculator := item.percentFunction()

	out := listResourceOutput{}

	if !lp.opts.HideRequests {
		out.Requests = valueCalculator(item.request)
		out.RequestsPct = percentCalculator(item.request)
	}

	if !lp.opts.HideLimits {
		out.Limits = valueCalculator(item.limit)
		out.LimitsPct = percentCalculator(item.limit)
	}

	if lp.opts.ShowUtil {
		out.Utilization = valueCalculator(item.utilization)
		out.UtilizationPct = percentCalculator(item.utilization)
	}
	return &out
}
