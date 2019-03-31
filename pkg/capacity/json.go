// Package capacity - json.go contains all the messy details for the json printer implementation
package capacity

import (
	"encoding/json"
	"fmt"
)

type jsonNodeMetric struct {
	Name   string              `json:"name"`
	CPU    *jsonResourceOutput `json:"cpu,omitempty"`
	Memory *jsonResourceOutput `json:"memory,omitempty"`
	Pods   []*jsonPod          `json:"pods,omitempty"`
}

type jsonPod struct {
	Name      string              `json:"name"`
	Namespace string              `json:"namespace"`
	CPU       *jsonResourceOutput `json:"cpu"`
	Memory    *jsonResourceOutput `json:"memory"`
}

type jsonResourceOutput struct {
	Requests       string `json:"requests"`
	RequestsPct    string `json:"requests_pct"`
	Limits         string `json:"limits"`
	LimitsPct      string `json:"limits_pct"`
	Utilization    string `json:"utilization,omitempty"`
	UtilizationPct string `json:"utilization_pct,omitempty"`
}

type jsonClusterMetrics struct {
	Nodes         []*jsonNodeMetric `json:"nodes"`
	ClusterTotals struct {
		CPU    *jsonResourceOutput `json:"cpu"`
		Memory *jsonResourceOutput `json:"memory"`
	} `json:"cluster_totals"`
}

type jsonPrinter struct {
	cm       *clusterMetric
	showPods bool
	showUtil bool
}

func (jp jsonPrinter) Print() {
	jsonOutput := jp.buildJSONClusterMetrics()

	jsonRaw, err := json.MarshalIndent(jsonOutput, "", "  ")
	if err != nil {
		fmt.Println("Error Marshalling JSON")
		fmt.Println(err)
	}

	fmt.Printf("%s", jsonRaw)
}

func (jp *jsonPrinter) buildJSONClusterMetrics() jsonClusterMetrics {
	var response jsonClusterMetrics

	response.ClusterTotals.CPU = jp.buildJSONResourceOutput(jp.cm.cpu)
	response.ClusterTotals.Memory = jp.buildJSONResourceOutput(jp.cm.memory)

	for key, val := range jp.cm.nodeMetrics {
		var node jsonNodeMetric
		node.Name = key
		node.CPU = jp.buildJSONResourceOutput(val.cpu)
		node.Memory = jp.buildJSONResourceOutput(val.memory)
		if jp.showPods {
			for _, val := range val.podMetrics {
				var newNode jsonPod
				newNode.Name = val.name
				newNode.Namespace = val.namespace
				newNode.CPU = jp.buildJSONResourceOutput(val.cpu)
				newNode.Memory = jp.buildJSONResourceOutput(val.memory)
				node.Pods = append(node.Pods, &newNode)
			}
		}
		response.Nodes = append(response.Nodes, &node)
	}

	return response
}

func (jp *jsonPrinter) buildJSONResourceOutput(item *resourceMetric) *jsonResourceOutput {
	valueCalculator := item.valueFunction()
	percentCalculator := item.percentFunction()

	out := jsonResourceOutput{
		Requests:    valueCalculator(item.request),
		RequestsPct: percentCalculator(item.request),
		Limits:      valueCalculator(item.limit),
		LimitsPct:   percentCalculator(item.limit),
	}

	if jp.showUtil {
		out.Utilization = valueCalculator(item.utilization)
		out.UtilizationPct = percentCalculator(item.utilization)
	}
	return &out
}
