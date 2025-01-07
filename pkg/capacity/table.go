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
	cm              *clusterMetric
	showPods        bool
	showUtil        bool
	showPodCount    bool
	showContainers  bool
	showNamespace   bool
	hideRequests    bool
	hideLimits     bool
	sortBy          string
	w               *tabwriter.Writer
	availableFormat bool
}

func (tp *tablePrinter) hasVisibleColumns() bool {
	// Check if any data columns will be shown
	return !tp.hideRequests || !tp.hideLimits || tp.showUtil || tp.showPodCount
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
	podCount       string
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
	podCount:       "POD COUNT",
}

func (tp *tablePrinter) Print() {
	tp.w.Init(os.Stdout, 0, 8, 2, ' ', 0)
	sortedNodeMetrics := tp.cm.getSortedNodeMetrics(tp.sortBy)

	tp.printLine(&headerStrings)

	if len(sortedNodeMetrics) > 1 {
		tp.printClusterLine()
	}

	for _, nm := range sortedNodeMetrics {
		if tp.showPods || tp.showContainers {
			tp.printLine(&tableLine{})
		}

		tp.printNodeLine(nm.name, nm)

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

	err := tp.w.Flush()
	if err != nil {
		fmt.Printf("Error writing to table: %s", err)
	}
}

func (tp *tablePrinter) printLine(tl *tableLine) {
	lineItems := tp.getLineItems(tl)
	fmt.Fprintf(tp.w, strings.Join(lineItems[:], "\t ")+"\n")
}

func (tp *tablePrinter) getLineItems(tl *tableLine) []string {
	lineItems := []string{tl.node}

	if tp.showContainers || tp.showPods {
		if tp.showNamespace {
			lineItems = append(lineItems, tl.namespace)
		}
		lineItems = append(lineItems, tl.pod)
	}

	if tp.showContainers {
		lineItems = append(lineItems, tl.container)
	}

	if !tp.hideRequests {
		lineItems = append(lineItems, tl.cpuRequests)
	}
	if !tp.hideLimits {
		lineItems = append(lineItems, tl.cpuLimits)
	}

	if tp.showUtil {
		lineItems = append(lineItems, tl.cpuUtil)
	}

	if !tp.hideRequests {
		lineItems = append(lineItems, tl.memoryRequests)
	}
	if !tp.hideLimits {
		lineItems = append(lineItems, tl.memoryLimits)
	}

	if tp.showUtil {
		lineItems = append(lineItems, tl.memoryUtil)
	}

	if tp.showPodCount {
		lineItems = append(lineItems, tl.podCount)
	}

	return lineItems
}

func (tp *tablePrinter) printClusterLine() {
	tp.printLine(&tableLine{
		node:           VoidValue,
		namespace:      VoidValue,
		pod:            VoidValue,
		container:      VoidValue,
		cpuRequests:    tp.cm.cpu.requestString(tp.availableFormat),
		cpuLimits:      tp.cm.cpu.limitString(tp.availableFormat),
		cpuUtil:        tp.cm.cpu.utilString(tp.availableFormat),
		memoryRequests: tp.cm.memory.requestString(tp.availableFormat),
		memoryLimits:   tp.cm.memory.limitString(tp.availableFormat),
		memoryUtil:     tp.cm.memory.utilString(tp.availableFormat),
		podCount:       tp.cm.podCount.podCountString(),
	})
}

func (tp *tablePrinter) printNodeLine(nodeName string, nm *nodeMetric) {
	tp.printLine(&tableLine{
		node:           nodeName,
		namespace:      VoidValue,
		pod:            VoidValue,
		container:      VoidValue,
		cpuRequests:    nm.cpu.requestString(tp.availableFormat),
		cpuLimits:      nm.cpu.limitString(tp.availableFormat),
		cpuUtil:        nm.cpu.utilString(tp.availableFormat),
		memoryRequests: nm.memory.requestString(tp.availableFormat),
		memoryLimits:   nm.memory.limitString(tp.availableFormat),
		memoryUtil:     nm.memory.utilString(tp.availableFormat),
		podCount:       nm.podCount.podCountString(),
	})
}

func (tp *tablePrinter) printPodLine(nodeName string, pm *podMetric) {
	tp.printLine(&tableLine{
		node:           nodeName,
		namespace:      pm.namespace,
		pod:            pm.name,
		container:      VoidValue,
		cpuRequests:    pm.cpu.requestString(tp.availableFormat),
		cpuLimits:      pm.cpu.limitString(tp.availableFormat),
		cpuUtil:        pm.cpu.utilString(tp.availableFormat),
		memoryRequests: pm.memory.requestString(tp.availableFormat),
		memoryLimits:   pm.memory.limitString(tp.availableFormat),
		memoryUtil:     pm.memory.utilString(tp.availableFormat),
	})
}

func (tp *tablePrinter) printContainerLine(nodeName string, pm *podMetric, cm *containerMetric) {
	tp.printLine(&tableLine{
		node:           nodeName,
		namespace:      pm.namespace,
		pod:            pm.name,
		container:      cm.name,
		cpuRequests:    cm.cpu.requestString(tp.availableFormat),
		cpuLimits:      cm.cpu.limitString(tp.availableFormat),
		cpuUtil:        cm.cpu.utilString(tp.availableFormat),
		memoryRequests: cm.memory.requestString(tp.availableFormat),
		memoryLimits:   cm.memory.limitString(tp.availableFormat),
		memoryUtil:     cm.memory.utilString(tp.availableFormat),
	})
}
