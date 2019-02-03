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

	"github.com/robscott/kube-capacity/pkg/kube"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func List(args []string, showPods bool, showUtil bool) {
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

	printList(&cm, showPods, showUtil)
}
