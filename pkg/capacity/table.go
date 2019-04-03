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
	"strings"
	"text/tabwriter"
)

type tablePrinter struct {
	cm             *clusterMetric
	showPods       bool
	showUtil       bool
	showContainers bool
	sortBy         string
	w              *tabwriter.Writer
}

type tableLine struct {
	node           string
	namespace      string
	pod            string
	container      string
	cpuRequests    string
	cpuLimits      string
	cpuUtil        string
	memoryRequests string
	memoryLimits   string
	memoryUtil     string
}

var headerStrings = tableLine{
	node:           "NODE",
	namespace:      "NAMESPACE",
	pod:            "POD",
	container:      "CONTAINER",
	cpuRequests:    "CPU REQUESTS",
	cpuLimits:      "CPU LIMITS",
	cpuUtil:        "CPU UTIL",
	memoryRequests: "MEMORY REQUESTS",
	memoryLimits:   "MEMORY LIMITS",
	memoryUtil:     "MEMORY UTIL",
}

func (tp *tablePrinter) Print() {
	tp.w.Init(os.Stdout, 0, 8, 2, ' ', 0)
	sortedNodeMetrics := tp.cm.getSortedNodeMetrics(tp.sortBy)

	tp.printLine(&headerStrings)

	if len(sortedNodeMetrics) > 1 {
		tp.printClusterLine()
		tp.printLine(&tableLine{})
	}

	for _, nm := range sortedNodeMetrics {
		tp.printNodeLine(nm.name, nm)
		tp.printLine(&tableLine{})

		if tp.showPods || tp.showContainers {
			podMetrics := nm.getSortedPodMetrics(tp.sortBy)
			for _, pm := range podMetrics {
				tp.printPodLine(nm.name, pm)
				if tp.showContainers {
					containerMetrics := pm.getSortedContainerMetrics(tp.sortBy)
					for _, containerMetric := range containerMetrics {
						tp.printContainerLine(nm.name, pm, containerMetric)
					}
				}
			}
		}
	}

	tp.w.Flush()
}

func (tp *tablePrinter) printLine(tl *tableLine) {
	lineItems := []string{tl.node, tl.namespace}

	if tp.showContainers || tp.showPods {
		lineItems = append(lineItems, tl.pod)
	}

	if tp.showContainers {
		lineItems = append(lineItems, tl.container)
	}

	lineItems = append(lineItems, tl.cpuRequests)
	lineItems = append(lineItems, tl.cpuLimits)

	if tp.showUtil {
		lineItems = append(lineItems, tl.cpuUtil)
	}

	lineItems = append(lineItems, tl.memoryRequests)
	lineItems = append(lineItems, tl.memoryLimits)

	if tp.showUtil {
		lineItems = append(lineItems, tl.memoryUtil)
	}

	fmt.Fprintf(tp.w, strings.Join(lineItems[:], "\t ")+"\n")
}

func (tp *tablePrinter) printClusterLine() {
	tp.printLine(&tableLine{
		node:           "*",
		namespace:      "*",
		pod:            "*",
		container:      "*",
		cpuRequests:    tp.cm.cpu.requestString(),
		cpuLimits:      tp.cm.cpu.limitString(),
		cpuUtil:        tp.cm.cpu.utilString(),
		memoryRequests: tp.cm.memory.requestString(),
		memoryLimits:   tp.cm.memory.limitString(),
		memoryUtil:     tp.cm.memory.utilString(),
	})
}

func (tp *tablePrinter) printNodeLine(nodeName string, nm *nodeMetric) {
	tp.printLine(&tableLine{
		node:           nodeName,
		namespace:      "*",
		pod:            "*",
		container:      "*",
		cpuRequests:    nm.cpu.requestString(),
		cpuLimits:      nm.cpu.limitString(),
		cpuUtil:        nm.cpu.utilString(),
		memoryRequests: nm.memory.requestString(),
		memoryLimits:   nm.memory.limitString(),
		memoryUtil:     nm.memory.utilString(),
	})
}

func (tp *tablePrinter) printPodLine(nodeName string, pm *podMetric) {
	tp.printLine(&tableLine{
		node:           nodeName,
		namespace:      pm.namespace,
		pod:            pm.name,
		container:      "*",
		cpuRequests:    pm.cpu.requestString(),
		cpuLimits:      pm.cpu.limitString(),
		cpuUtil:        pm.cpu.utilString(),
		memoryRequests: pm.memory.requestString(),
		memoryLimits:   pm.memory.limitString(),
		memoryUtil:     pm.memory.utilString(),
	})
}

func (tp *tablePrinter) printContainerLine(nodeName string, pm *podMetric, cm *containerMetric) {
	tp.printLine(&tableLine{
		node:           nodeName,
		namespace:      pm.namespace,
		pod:            pm.name,
		container:      cm.name,
		cpuRequests:    cm.cpu.requestString(),
		cpuLimits:      cm.cpu.limitString(),
		cpuUtil:        cm.cpu.utilString(),
		memoryRequests: cm.memory.requestString(),
		memoryLimits:   cm.memory.limitString(),
		memoryUtil:     cm.memory.utilString(),
	})
}
