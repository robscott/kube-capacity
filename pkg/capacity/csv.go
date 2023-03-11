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
	"io"
	"os"
	"strings"
)

type csvPrinter struct {
	cm              *clusterMetric
	showPods        bool
	showUtil        bool
	showPodCount    bool
	showContainers  bool
	showNamespace   bool
	sortBy          string
	file            io.Writer
	separator       string
	availableFormat bool
}

type csvLine struct {
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

var csvHeaderStrings = csvLine{
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

func (cp *csvPrinter) Print(outputType string) {

	cp.file = os.Stdout
	cp.separator = outputType

	sortedNodeMetrics := cp.cm.getSortedNodeMetrics(cp.sortBy)

	cp.printLine(&csvHeaderStrings)

	if len(sortedNodeMetrics) > 1 {
		cp.printClusterLine()
	}

	for _, nm := range sortedNodeMetrics {
		if cp.showPods || cp.showContainers {
			cp.printLine(&csvLine{})
		}

		cp.printNodeLine(nm.name, nm)

		if cp.showPods || cp.showContainers {
			podMetrics := nm.getSortedPodMetrics(cp.sortBy)
			for _, pm := range podMetrics {
				cp.printPodLine(nm.name, pm)
				if cp.showContainers {
					containerMetrics := pm.getSortedContainerMetrics(cp.sortBy)
					for _, containerMetric := range containerMetrics {
						cp.printContainerLine(nm.name, pm, containerMetric)
					}
				}
			}
		}
	}
}

func (cp *csvPrinter) printLine(cl *csvLine) {
	separator := ", "
	if cp.separator == TSVOutput {
		separator = "\t "
	}

	lineItems := cp.getLineItems(cl)
	// need to escape or enclose strings that are not numbers only
	fmt.Fprintf(cp.file, strings.Join(lineItems[:], separator)+"\n")
}

func (cp *csvPrinter) getLineItems(cl *csvLine) []string {
	lineItems := []string{cl.node}

	if cp.showContainers || cp.showPods {
		if cp.showNamespace {
			lineItems = append(lineItems, cl.namespace)
		}
		lineItems = append(lineItems, cl.pod)
	}

	if cp.showContainers {
		lineItems = append(lineItems, cl.container)
	}

	lineItems = append(lineItems, cl.cpuRequests)
	lineItems = append(lineItems, cl.cpuLimits)

	if cp.showUtil {
		lineItems = append(lineItems, cl.cpuUtil)
	}

	lineItems = append(lineItems, cl.memoryRequests)
	lineItems = append(lineItems, cl.memoryLimits)

	if cp.showUtil {
		lineItems = append(lineItems, cl.memoryUtil)
	}

	if cp.showPodCount {
		lineItems = append(lineItems, cl.podCount)
	}

	return lineItems
}

func (cp *csvPrinter) printClusterLine() {
	cp.printLine(&csvLine{
		node:           "*",
		namespace:      "*",
		pod:            "*",
		container:      "*",
		cpuRequests:    cp.cm.cpu.requestString(cp.availableFormat),
		cpuLimits:      cp.cm.cpu.limitString(cp.availableFormat),
		cpuUtil:        cp.cm.cpu.utilString(cp.availableFormat),
		memoryRequests: cp.cm.memory.requestString(cp.availableFormat),
		memoryLimits:   cp.cm.memory.limitString(cp.availableFormat),
		memoryUtil:     cp.cm.memory.utilString(cp.availableFormat),
		podCount:       cp.cm.podCount.podCountString(),
	})
}

func (cp *csvPrinter) printNodeLine(nodeName string, nm *nodeMetric) {
	cp.printLine(&csvLine{
		node:           nodeName,
		namespace:      "*",
		pod:            "*",
		container:      "*",
		cpuRequests:    nm.cpu.requestString(cp.availableFormat),
		cpuLimits:      nm.cpu.limitString(cp.availableFormat),
		cpuUtil:        nm.cpu.utilString(cp.availableFormat),
		memoryRequests: nm.memory.requestString(cp.availableFormat),
		memoryLimits:   nm.memory.limitString(cp.availableFormat),
		memoryUtil:     nm.memory.utilString(cp.availableFormat),
		podCount:       nm.podCount.podCountString(),
	})
}

func (cp *csvPrinter) printPodLine(nodeName string, pm *podMetric) {
	cp.printLine(&csvLine{
		node:           nodeName,
		namespace:      pm.namespace,
		pod:            pm.name,
		container:      "*",
		cpuRequests:    pm.cpu.requestString(cp.availableFormat),
		cpuLimits:      pm.cpu.limitString(cp.availableFormat),
		cpuUtil:        pm.cpu.utilString(cp.availableFormat),
		memoryRequests: pm.memory.requestString(cp.availableFormat),
		memoryLimits:   pm.memory.limitString(cp.availableFormat),
		memoryUtil:     pm.memory.utilString(cp.availableFormat),
	})
}

func (cp *csvPrinter) printContainerLine(nodeName string, pm *podMetric, cm *containerMetric) {
	cp.printLine(&csvLine{
		node:           nodeName,
		namespace:      pm.namespace,
		pod:            pm.name,
		container:      cm.name,
		cpuRequests:    cm.cpu.requestString(cp.availableFormat),
		cpuLimits:      cm.cpu.limitString(cp.availableFormat),
		cpuUtil:        cm.cpu.utilString(cp.availableFormat),
		memoryRequests: cm.memory.requestString(cp.availableFormat),
		memoryLimits:   cm.memory.limitString(cp.availableFormat),
		memoryUtil:     cm.memory.utilString(cp.availableFormat),
	})
}
