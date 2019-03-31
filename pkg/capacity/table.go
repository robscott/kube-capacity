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
	"sort"
	"text/tabwriter"
)

type tablePrinter struct {
	cm       *clusterMetric
	showPods bool
	showUtil bool
	w        *tabwriter.Writer
}

func (tp *tablePrinter) Print() {
	tp.w.Init(os.Stdout, 0, 8, 2, ' ', 0)
	names := make([]string, len(tp.cm.nodeMetrics))

	i := 0
	for name := range tp.cm.nodeMetrics {
		names[i] = name
		i++
	}
	sort.Strings(names)

	tp.printHeaders()

	for _, name := range names {
		tp.printNode(name, tp.cm.nodeMetrics[name])
	}

	tp.w.Flush()
}

func (tp *tablePrinter) printHeaders() {
	if tp.showPods && tp.showUtil {
		fmt.Fprintln(tp.w, "NODE\t NAMESPACE\t POD\t CPU REQUESTS \t CPU LIMITS \t CPU UTIL \t MEMORY REQUESTS \t MEMORY LIMITS \t MEMORY UTIL")

		if len(tp.cm.nodeMetrics) > 1 {
			fmt.Fprintf(tp.w, "* \t *\t *\t %s \t %s \t %s \t %s \t %s \t %s \n",
				tp.cm.cpu.requestString(),
				tp.cm.cpu.limitString(),
				tp.cm.cpu.utilString(),
				tp.cm.memory.requestString(),
				tp.cm.memory.limitString(),
				tp.cm.memory.utilString())

			fmt.Fprintln(tp.w, "\t\t\t\t\t\t\t\t")
		}
	} else if tp.showPods {
		fmt.Fprintln(tp.w, "NODE\t NAMESPACE\t POD\t CPU REQUESTS \t CPU LIMITS \t MEMORY REQUESTS \t MEMORY LIMITS")

		fmt.Fprintf(tp.w, "* \t *\t *\t %s \t %s \t %s \t %s \n",
			tp.cm.cpu.requestString(),
			tp.cm.cpu.limitString(),
			tp.cm.memory.requestString(),
			tp.cm.memory.limitString())

		fmt.Fprintln(tp.w, "\t\t\t\t\t\t")

	} else if tp.showUtil {
		fmt.Fprintln(tp.w, "NODE\t CPU REQUESTS \t CPU LIMITS \t CPU UTIL \t MEMORY REQUESTS \t MEMORY LIMITS \t MEMORY UTIL")

		fmt.Fprintf(tp.w, "* \t %s \t %s \t %s \t %s \t %s \t %s \n",
			tp.cm.cpu.requestString(),
			tp.cm.cpu.limitString(),
			tp.cm.cpu.utilString(),
			tp.cm.memory.requestString(),
			tp.cm.memory.limitString(),
			tp.cm.memory.utilString())

	} else {
		fmt.Fprintln(tp.w, "NODE\t CPU REQUESTS \t CPU LIMITS \t MEMORY REQUESTS \t MEMORY LIMITS")

		if len(tp.cm.nodeMetrics) > 1 {
			fmt.Fprintf(tp.w, "* \t %s \t %s \t %s \t %s \n",
				tp.cm.cpu.requestString(), tp.cm.cpu.limitString(),
				tp.cm.memory.requestString(), tp.cm.memory.limitString())
		}
	}
}

func (tp *tablePrinter) printNode(name string, nm *nodeMetric) {
	podNames := make([]string, len(nm.podMetrics))

	i := 0
	for name := range nm.podMetrics {
		podNames[i] = name
		i++
	}
	sort.Strings(podNames)

	if tp.showPods && tp.showUtil {
		fmt.Fprintf(tp.w, "%s \t *\t *\t %s \t %s \t %s \t %s \t %s \t %s \n",
			name,
			nm.cpu.requestString(),
			nm.cpu.limitString(),
			nm.cpu.utilString(),
			nm.memory.requestString(),
			nm.memory.limitString(),
			nm.memory.utilString())

		for _, podName := range podNames {
			pm := nm.podMetrics[podName]
			fmt.Fprintf(tp.w, "%s \t %s \t %s \t %s \t %s \t %s \t %s \t %s \t %s \n",
				name,
				pm.namespace,
				pm.name,
				pm.cpu.requestString(),
				pm.cpu.limitString(),
				pm.cpu.utilString(),
				pm.memory.requestString(),
				pm.memory.limitString(),
				pm.memory.utilString())
		}

		fmt.Fprintln(tp.w, "\t\t\t\t\t\t\t\t")

	} else if tp.showPods {
		fmt.Fprintf(tp.w, "%s \t *\t *\t %s \t %s \t %s \t %s \n",
			name,
			nm.cpu.requestString(),
			nm.cpu.limitString(),
			nm.memory.requestString(),
			nm.memory.limitString())

		for _, podName := range podNames {
			pm := nm.podMetrics[podName]
			fmt.Fprintf(tp.w, "%s \t %s \t %s \t %s \t %s \t %s \t %s \n",
				name,
				pm.namespace,
				pm.name,
				pm.cpu.requestString(),
				pm.cpu.limitString(),
				pm.memory.requestString(),
				pm.memory.limitString())
		}

		fmt.Fprintln(tp.w, "\t\t\t\t\t\t")

	} else if tp.showUtil {
		fmt.Fprintf(tp.w, "%s \t %s \t %s \t %s \t %s \t %s \t %s \n",
			name,
			nm.cpu.requestString(),
			nm.cpu.limitString(),
			nm.cpu.utilString(),
			nm.memory.requestString(),
			nm.memory.limitString(),
			nm.memory.utilString())

	} else {
		fmt.Fprintf(tp.w, "%s \t %s \t %s \t %s \t %s \n", name,
			nm.cpu.requestString(), nm.cpu.limitString(),
			nm.memory.requestString(), nm.memory.limitString())
	}
}
