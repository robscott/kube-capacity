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
	cm   *clusterMetric
	w    *tabwriter.Writer
	opts Options
}

func (tp *tablePrinter) hasVisibleColumns() bool {
	// Check if any data columns will be shown
	return !tp.opts.HideRequests || !tp.opts.HideLimits || tp.opts.ShowUtil || tp.opts.ShowPodCount || tp.opts.ShowLabels
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
	labels         string
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
	labels:         "LABELS",
}

func (tp *tablePrinter) Print() {
	tp.w.Init(os.Stdout, 0, 8, 2, ' ', 0)
	sortedNodeMetrics := tp.cm.getSortedNodeMetrics(tp.opts.SortBy)

	tp.printLine(&headerStrings)

	if len(sortedNodeMetrics) > 1 {
		tp.printClusterLine()
	}

	for _, nm := range sortedNodeMetrics {
		if tp.opts.ShowPods || tp.opts.ShowContainers {
			tp.printLine(&tableLine{})
		}

		tp.printNodeLine(nm.name, nm)

		if tp.opts.ShowPods || tp.opts.ShowContainers {
			podMetrics := nm.getSortedPodMetrics(tp.opts.SortBy)
			for _, pm := range podMetrics {
				tp.printPodLine(nm.name, pm)
				if tp.opts.ShowContainers {
					containerMetrics := pm.getSortedContainerMetrics(tp.opts.SortBy)
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
	_, _ = fmt.Fprintln(tp.w, strings.Join(lineItems[:], "\t "))
}

func (tp *tablePrinter) getLineItems(tl *tableLine) []string {
	lineItems := []string{tl.node}

	if tp.opts.ShowContainers || tp.opts.ShowPods {
		if tp.opts.Namespace == "" {
			lineItems = append(lineItems, tl.namespace)
		}
		lineItems = append(lineItems, tl.pod)
	}

	if tp.opts.ShowContainers {
		lineItems = append(lineItems, tl.container)
	}

	if !tp.opts.HideRequests {
		lineItems = append(lineItems, tl.cpuRequests)
	}
	if !tp.opts.HideLimits {
		lineItems = append(lineItems, tl.cpuLimits)
	}

	if tp.opts.ShowUtil {
		lineItems = append(lineItems, tl.cpuUtil)
	}

	if !tp.opts.HideRequests {
		lineItems = append(lineItems, tl.memoryRequests)
	}
	if !tp.opts.HideLimits {
		lineItems = append(lineItems, tl.memoryLimits)
	}

	if tp.opts.ShowUtil {
		lineItems = append(lineItems, tl.memoryUtil)
	}

	if tp.opts.ShowPodCount {
		lineItems = append(lineItems, tl.podCount)
	}

	if tp.opts.ShowLabels {
		lineItems = append(lineItems, tl.labels)
	}

	return lineItems
}

func (tp *tablePrinter) printClusterLine() {
	tp.printLine(&tableLine{
		node:           VoidValue,
		namespace:      VoidValue,
		pod:            VoidValue,
		container:      VoidValue,
		cpuRequests:    tp.cm.cpu.requestString(tp.opts.AvailableFormat),
		cpuLimits:      tp.cm.cpu.limitString(tp.opts.AvailableFormat),
		cpuUtil:        tp.cm.cpu.utilString(tp.opts.AvailableFormat),
		memoryRequests: tp.cm.memory.requestString(tp.opts.AvailableFormat),
		memoryLimits:   tp.cm.memory.limitString(tp.opts.AvailableFormat),
		memoryUtil:     tp.cm.memory.utilString(tp.opts.AvailableFormat),
		podCount:       tp.cm.podCount.podCountString(),
		labels:         VoidValue,
	})
}

func (tp *tablePrinter) printNodeLine(nodeName string, nm *nodeMetric) {
	tp.printLine(&tableLine{
		node:           nodeName,
		namespace:      VoidValue,
		pod:            VoidValue,
		container:      VoidValue,
		cpuRequests:    nm.cpu.requestString(tp.opts.AvailableFormat),
		cpuLimits:      nm.cpu.limitString(tp.opts.AvailableFormat),
		cpuUtil:        nm.cpu.utilString(tp.opts.AvailableFormat),
		memoryRequests: nm.memory.requestString(tp.opts.AvailableFormat),
		memoryLimits:   nm.memory.limitString(tp.opts.AvailableFormat),
		memoryUtil:     nm.memory.utilString(tp.opts.AvailableFormat),
		podCount:       nm.podCount.podCountString(),
		labels:         nodeLabelsString(nm.labels),
	})
}

func (tp *tablePrinter) printPodLine(nodeName string, pm *podMetric) {
	tp.printLine(&tableLine{
		node:           nodeName,
		namespace:      pm.namespace,
		pod:            pm.name,
		container:      VoidValue,
		cpuRequests:    pm.cpu.requestString(tp.opts.AvailableFormat),
		cpuLimits:      pm.cpu.limitString(tp.opts.AvailableFormat),
		cpuUtil:        pm.cpu.utilString(tp.opts.AvailableFormat),
		memoryRequests: pm.memory.requestString(tp.opts.AvailableFormat),
		memoryLimits:   pm.memory.limitString(tp.opts.AvailableFormat),
		memoryUtil:     pm.memory.utilString(tp.opts.AvailableFormat),
	})
}

func (tp *tablePrinter) printContainerLine(nodeName string, pm *podMetric, cm *containerMetric) {
	tp.printLine(&tableLine{
		node:           nodeName,
		namespace:      pm.namespace,
		pod:            pm.name,
		container:      cm.name,
		cpuRequests:    cm.cpu.requestString(tp.opts.AvailableFormat),
		cpuLimits:      cm.cpu.limitString(tp.opts.AvailableFormat),
		cpuUtil:        cm.cpu.utilString(tp.opts.AvailableFormat),
		memoryRequests: cm.memory.requestString(tp.opts.AvailableFormat),
		memoryLimits:   cm.memory.limitString(tp.opts.AvailableFormat),
		memoryUtil:     cm.memory.utilString(tp.opts.AvailableFormat),
	})
}
