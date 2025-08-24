// Copyright 2023 Kube Capacity Authors
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
	cm   *clusterMetric
	file io.Writer
	opts Options
}

type csvLine struct {
	node                     string
	namespace                string
	pod                      string
	container                string
	cpuCapacity              string
	cpuRequests              string
	cpuRequestsPercentage    string
	cpuLimits                string
	cpuLimitsPercentage      string
	cpuUtil                  string
	cpuUtilPercentage        string
	memoryCapacity           string
	memoryRequests           string
	memoryRequestsPercentage string
	memoryLimits             string
	memoryLimitsPercentage   string
	memoryUtil               string
	memoryUtilPercentage     string
	podCountCurrent          string
	podCountAllocatable      string
	labels                   string
	labelColumns             []string
}

var staticCsvHeaderStrings = csvLine{
	node:                     "NODE",
	namespace:                "NAMESPACE",
	pod:                      "POD",
	container:                "CONTAINER",
	cpuCapacity:              "CPU CAPACITY (milli)",
	cpuRequests:              "CPU REQUESTS",
	cpuRequestsPercentage:    "CPU REQUESTS %%",
	cpuLimits:                "CPU LIMITS",
	cpuLimitsPercentage:      "CPU LIMITS %%",
	cpuUtil:                  "CPU UTIL",
	cpuUtilPercentage:        "CPU UTIL %%",
	memoryCapacity:           "MEMORY CAPACITY (Mi)",
	memoryRequests:           "MEMORY REQUESTS",
	memoryRequestsPercentage: "MEMORY REQUESTS %%",
	memoryLimits:             "MEMORY LIMITS",
	memoryLimitsPercentage:   "MEMORY LIMITS %%",
	memoryUtil:               "MEMORY UTIL",
	memoryUtilPercentage:     "MEMORY UTIL %%",
	podCountCurrent:          "POD COUNT CURRENT",
	podCountAllocatable:      "POD COUNT ALLOCATABLE",
	labels:                   "LABELS",
	labelColumns:             []string{},
}

func getCsvHeaderStrings(opts Options) csvLine {
	h := staticCsvHeaderStrings

	if opts.LabelColumns != "" {
		h.labelColumns = nodeLabelHeaders(opts.LabelColumns)
	}
	return h
}

func (cp *csvPrinter) Print(outputType string) {

	cp.file = os.Stdout

	sortedNodeMetrics := cp.cm.getSortedNodeMetrics(cp.opts.SortBy)

	hs := getCsvHeaderStrings(cp.opts)
	cp.printLine(&hs)

	// number of custom label columns to print
	labelColumnsCnt := len(hs.labelColumns)

	if len(sortedNodeMetrics) > 1 {
		cp.printClusterLine(labelColumnsCnt)
	}

	for _, nm := range sortedNodeMetrics {
		cp.printNodeLine(nm.name, nm)

		if cp.opts.ShowPods || cp.opts.ShowContainers {
			podMetrics := nm.getSortedPodMetrics(cp.opts.SortBy)
			for _, pm := range podMetrics {
				cp.printPodLine(nm.name, pm, labelColumnsCnt)
				if cp.opts.ShowContainers {
					containerMetrics := pm.getSortedContainerMetrics(cp.opts.SortBy)
					for _, containerMetric := range containerMetrics {
						cp.printContainerLine(nm.name, pm, containerMetric, labelColumnsCnt)
					}
				}
			}
		}
	}
}

func (cp *csvPrinter) printLine(cl *csvLine) {
	separator := ","
	if cp.opts.OutputFormat == TSVOutput {
		separator = "\t"
	}

	lineItems := cp.getLineItems(cl)

	_, _ = fmt.Fprintln(cp.file, strings.Join(lineItems[:], separator))
}

func (cp *csvPrinter) getLineItems(cl *csvLine) []string {
	lineItems := []string{CSVStringTerminator + cl.node + CSVStringTerminator}

	if cp.opts.ShowContainers || cp.opts.ShowPods {
		if cp.opts.Namespace == "" {
			lineItems = append(lineItems, CSVStringTerminator+cl.namespace+CSVStringTerminator)
		}
		lineItems = append(lineItems, CSVStringTerminator+cl.pod+CSVStringTerminator)
	}

	if cp.opts.ShowContainers {
		lineItems = append(lineItems, CSVStringTerminator+cl.container+CSVStringTerminator)
	}

	lineItems = append(lineItems, cl.cpuCapacity)
	if !cp.opts.HideRequests {
		lineItems = append(lineItems, cl.cpuRequests)
		lineItems = append(lineItems, cl.cpuRequestsPercentage)
	}
	if !cp.opts.HideLimits {
		lineItems = append(lineItems, cl.cpuLimits)
		lineItems = append(lineItems, cl.cpuLimitsPercentage)
	}

	if cp.opts.ShowUtil {
		lineItems = append(lineItems, cl.cpuUtil)
		lineItems = append(lineItems, cl.cpuUtilPercentage)
	}

	lineItems = append(lineItems, cl.memoryCapacity)
	if !cp.opts.HideRequests {
		lineItems = append(lineItems, cl.memoryRequests)
		lineItems = append(lineItems, cl.memoryRequestsPercentage)
	}
	if !cp.opts.HideLimits {
		lineItems = append(lineItems, cl.memoryLimits)
		lineItems = append(lineItems, cl.memoryLimitsPercentage)
	}

	if cp.opts.ShowUtil {
		lineItems = append(lineItems, cl.memoryUtil)
		lineItems = append(lineItems, cl.memoryUtilPercentage)
	}

	if cp.opts.ShowPodCount {
		lineItems = append(lineItems, cl.podCountCurrent)
		lineItems = append(lineItems, cl.podCountAllocatable)
	}

	if cp.opts.ShowLabels {
		lineItems = append(lineItems, cl.labels)
	}

	if cp.opts.LabelColumns != "" {
		lineItems = append(lineItems, cl.labelColumns...)
	}

	return lineItems
}

func (cp *csvPrinter) printClusterLine(labelColumnsCnt int) {
	cp.printLine(&csvLine{
		node:                     VoidValue,
		namespace:                VoidValue,
		pod:                      VoidValue,
		container:                VoidValue,
		cpuCapacity:              cp.cm.cpu.capacityString(),
		cpuRequests:              cp.cm.cpu.requestActualString(),
		cpuRequestsPercentage:    cp.cm.cpu.requestPercentageString(),
		cpuLimits:                cp.cm.cpu.limitActualString(),
		cpuLimitsPercentage:      cp.cm.cpu.limitPercentageString(),
		cpuUtil:                  cp.cm.cpu.utilActualString(),
		cpuUtilPercentage:        cp.cm.cpu.utilPercentageString(),
		memoryCapacity:           cp.cm.memory.capacityString(),
		memoryRequests:           cp.cm.memory.requestActualString(),
		memoryRequestsPercentage: cp.cm.memory.requestPercentageString(),
		memoryLimits:             cp.cm.memory.limitActualString(),
		memoryLimitsPercentage:   cp.cm.memory.limitPercentageString(),
		memoryUtil:               cp.cm.memory.utilActualString(),
		memoryUtilPercentage:     cp.cm.memory.utilPercentageString(),
		podCountCurrent:          cp.cm.podCount.podCountCurrentString(),
		podCountAllocatable:      cp.cm.podCount.podCountAllocatableString(),
		labels:                   VoidValue,
		labelColumns:             make([]string, labelColumnsCnt),
	})
}

func (cp *csvPrinter) printNodeLine(nodeName string, nm *nodeMetric) {
	cp.printLine(&csvLine{
		node:                     nodeName,
		namespace:                VoidValue,
		pod:                      VoidValue,
		container:                VoidValue,
		cpuCapacity:              nm.cpu.capacityString(),
		cpuRequests:              nm.cpu.requestActualString(),
		cpuRequestsPercentage:    nm.cpu.requestPercentageString(),
		cpuLimits:                nm.cpu.limitActualString(),
		cpuLimitsPercentage:      nm.cpu.limitPercentageString(),
		cpuUtil:                  nm.cpu.utilActualString(),
		cpuUtilPercentage:        nm.cpu.utilPercentageString(),
		memoryCapacity:           nm.memory.capacityString(),
		memoryRequests:           nm.memory.requestActualString(),
		memoryRequestsPercentage: nm.memory.requestPercentageString(),
		memoryLimits:             nm.memory.limitActualString(),
		memoryLimitsPercentage:   nm.memory.limitPercentageString(),
		memoryUtil:               nm.memory.utilActualString(),
		memoryUtilPercentage:     nm.memory.utilPercentageString(),
		podCountCurrent:          nm.podCount.podCountCurrentString(),
		podCountAllocatable:      nm.podCount.podCountAllocatableString(),
		labels:                   fmt.Sprintf("%q", nodeLabelsString(nm.labels)), // quote the labels to avoid CSV parsing issues
		labelColumns:             nodeLabelValues(nm.labels, cp.opts.LabelColumns),
	})
}

func (cp *csvPrinter) printPodLine(nodeName string, pm *podMetric, labelColumnsCnt int) {
	cp.printLine(&csvLine{
		node:                     nodeName,
		namespace:                pm.namespace,
		pod:                      pm.name,
		container:                VoidValue,
		cpuCapacity:              pm.cpu.capacityString(),
		cpuRequests:              pm.cpu.requestActualString(),
		cpuRequestsPercentage:    pm.cpu.requestPercentageString(),
		cpuLimits:                pm.cpu.limitActualString(),
		cpuLimitsPercentage:      pm.cpu.limitPercentageString(),
		cpuUtil:                  pm.cpu.utilActualString(),
		cpuUtilPercentage:        pm.cpu.utilPercentageString(),
		memoryCapacity:           pm.memory.capacityString(),
		memoryRequests:           pm.memory.requestActualString(),
		memoryRequestsPercentage: pm.memory.requestPercentageString(),
		memoryLimits:             pm.memory.limitActualString(),
		memoryLimitsPercentage:   pm.memory.limitPercentageString(),
		memoryUtil:               pm.memory.utilActualString(),
		memoryUtilPercentage:     pm.memory.utilPercentageString(),
		labelColumns:             make([]string, labelColumnsCnt),
	})
}

func (cp *csvPrinter) printContainerLine(nodeName string, pm *podMetric, cm *containerMetric, labelColumnsCnt int) {
	cp.printLine(&csvLine{
		node:                     nodeName,
		namespace:                pm.namespace,
		pod:                      pm.name,
		container:                cm.name,
		cpuCapacity:              cm.cpu.capacityString(),
		cpuRequests:              cm.cpu.requestActualString(),
		cpuRequestsPercentage:    cm.cpu.requestPercentageString(),
		cpuLimits:                cm.cpu.limitActualString(),
		cpuLimitsPercentage:      cm.cpu.limitPercentageString(),
		cpuUtil:                  cm.cpu.utilActualString(),
		cpuUtilPercentage:        cm.cpu.utilPercentageString(),
		memoryCapacity:           cm.memory.capacityString(),
		memoryRequests:           cm.memory.requestActualString(),
		memoryRequestsPercentage: cm.memory.requestPercentageString(),
		memoryLimits:             cm.memory.limitActualString(),
		memoryLimitsPercentage:   cm.memory.limitPercentageString(),
		memoryUtil:               cm.memory.utilActualString(),
		memoryUtilPercentage:     cm.memory.utilPercentageString(),
		labelColumns:             make([]string, labelColumnsCnt),
	})
}
