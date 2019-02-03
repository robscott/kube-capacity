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
	"sort"
	"text/tabwriter"

	"github.com/robscott/kube-capacity/pkg/kube"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func List(args []string, outputFormat string) {
	clientset, err := kube.NewClientSet()
	if err != nil {
		fmt.Println("Error connecting to Kubernetes")
		panic(err.Error())
	}

	mClientset, err := kube.NewMetricsClientSet()
	if err != nil {
		fmt.Println("Error connecting to Metrics Server")
		panic(err.Error())
	}

	nodeList, err := clientset.CoreV1().Nodes().List(metav1.ListOptions{})
	if err != nil {
		fmt.Println("Error listing Nodes")
		panic(err.Error())
	}

	podList, err := clientset.CoreV1().Pods("").List(metav1.ListOptions{})
	if err != nil {
		fmt.Println("Error listing Nodes")
		panic(err.Error())
	}

	cm := clusterMetric{
		cpu:         &resourceMetric{},
		memory:      &resourceMetric{},
		nodeMetrics: map[string]*nodeMetric{},
		podMetrics:  map[string]*podMetric{},
	}

	for _, node := range nodeList.Items {
		cm.nodeMetrics[node.Name] = &nodeMetric{
			cpu: &resourceMetric{
				allocatable: node.Status.Allocatable["cpu"],
			},
			memory: &resourceMetric{
				allocatable: node.Status.Allocatable["memory"],
			},
			podMetrics: map[string]*podMetric{},
		}
	}

	for _, pod := range podList.Items {
		cm.addPodMetric(&pod)
	}

	for _, node := range nodeList.Items {
		cm.addNodeMetric(cm.nodeMetrics[node.Name])
	}

	nmList, err := mClientset.MetricsV1beta1().NodeMetricses().List(metav1.ListOptions{})
	if err != nil {
		fmt.Println("Error getting metrics")
		panic(err.Error())
	}

	for _, node := range nmList.Items {
		nm := cm.nodeMetrics[node.GetName()]
		cm.cpu.utilization.Add(node.Usage["cpu"])
		cm.memory.utilization.Add(node.Usage["memory"])
		nm.cpu.utilization = node.Usage["cpu"]
		nm.memory.utilization = node.Usage["memory"]
	}

	pmList, err := mClientset.MetricsV1beta1().PodMetricses("").List(metav1.ListOptions{})
	if err != nil {
		fmt.Println("Error getting metrics")
		panic(err.Error())
	}

	for _, pod := range pmList.Items {
		pm := cm.podMetrics[fmt.Sprintf("%s-%s", pod.GetNamespace(), pod.GetName())]
		for _, container := range pod.Containers {
			pm.cpu.utilization.Add(container.Usage["cpu"])
			pm.memory.utilization.Add(container.Usage["memory"])
		}
	}

	printList(cm, outputFormat)
}

func printList(cm clusterMetric, outputFormat string) {
	names := make([]string, len(cm.nodeMetrics))

	i := 0
	for name := range cm.nodeMetrics {
		names[i] = name
		i++
	}
	sort.Strings(names)

	w := new(tabwriter.Writer)
	w.Init(os.Stdout, 0, 8, 2, ' ', 0)

	if outputFormat == "wide" {
		fmt.Fprintln(w, "NODE\t NAMESPACE\t POD\t CPU REQUESTS \t CPU LIMITS \t CPU UTIL \t MEMORY REQUESTS \t MEMORY LIMITS \t MEMORY UTIL")

		if len(names) > 1 {
			fmt.Fprintf(w, "* \t *\t *\t %s \t %s \t %s \t %s \t %s \t %s \n",
				cm.cpu.requestString(), cm.cpu.limitString(), cm.cpu.utilString(),
				cm.memory.requestString(), cm.memory.limitString(), cm.memory.utilString())
			fmt.Fprintln(w, "\t\t\t\t\t\t\t\t")
		}
	} else {
		fmt.Fprintln(w, "NODE\t CPU REQUESTS \t CPU LIMITS \t MEMORY REQUESTS \t MEMORY LIMITS")

		if len(names) > 1 {
			fmt.Fprintf(w, "* \t %s \t %s \t %s \t %s \n",
				cm.cpu.requestString(), cm.cpu.limitString(),
				cm.memory.requestString(), cm.memory.limitString())
		}
	}

	for _, name := range names {
		nm := cm.nodeMetrics[name]

		if outputFormat == "wide" {
			fmt.Fprintf(w, "%s \t *\t *\t %s \t %s \t %s \t %s \t %s \t %s \n", name,
				nm.cpu.requestString(), nm.cpu.limitString(), nm.cpu.utilString(),
				nm.memory.requestString(), nm.memory.limitString(), nm.memory.utilString())

			for _, pm := range nm.podMetrics {
				fmt.Fprintf(w, "%s \t %s \t %s \t %s \t %s \t %s \t %s \t %s \t %s \n", name,
					pm.namespace, pm.name,
					pm.cpu.requestStringPar(nm.cpu), pm.cpu.limitStringPar(nm.cpu), pm.cpu.utilStringPar(nm.cpu),
					pm.memory.requestStringPar(nm.memory), pm.memory.limitStringPar(nm.memory), pm.memory.utilStringPar(nm.memory))
			}
			fmt.Fprintln(w, "\t\t\t\t\t\t\t\t")
		} else {
			fmt.Fprintf(w, "%s \t %s \t %s \t %s \t %s \n", name,
				nm.cpu.requestString(), nm.cpu.limitString(),
				nm.memory.requestString(), nm.memory.limitString())
		}
	}

	w.Flush()
}
