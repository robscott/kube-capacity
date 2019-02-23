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

	"k8s.io/client-go/kubernetes"
	metrics "k8s.io/metrics/pkg/client/clientset/versioned"

	"github.com/robscott/kube-capacity/pkg/kube"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1beta1 "k8s.io/metrics/pkg/apis/metrics/v1beta1"
)

// List gathers cluster resource data and outputs it
func List(showPods, showUtil bool, podLabels, nodeLabels, namespaceLabels string) {
	clientset, err := kube.NewClientSet()
	if err != nil {
		fmt.Printf("Error connecting to Kubernetes: %v\n", err)
		os.Exit(1)
	}

	podList, nodeList := getPodsAndNodes(clientset, podLabels, nodeLabels, namespaceLabels)
	pmList := &v1beta1.PodMetricsList{}
	if showUtil {
		mClientset, err := kube.NewMetricsClientSet()
		if err != nil {
			fmt.Printf("Error connecting to Metrics API: %v\n", err)
			os.Exit(4)
		}

		pmList = getMetrics(mClientset)
	}
	cm := buildClusterMetric(podList, pmList, nodeList)
	printList(&cm, showPods, showUtil)
}

func getPodsAndNodes(clientset kubernetes.Interface, podLabels, nodeLabels, namespaceLabels string) (*corev1.PodList, *corev1.NodeList) {
	nodeList, err := clientset.CoreV1().Nodes().List(metav1.ListOptions{
		LabelSelector: nodeLabels,
	})
	if err != nil {
		fmt.Printf("Error listing Nodes: %v\n", err)
		os.Exit(2)
	}

	podList, err := clientset.CoreV1().Pods("").List(metav1.ListOptions{
		LabelSelector: podLabels,
	})
	if err != nil {
		fmt.Printf("Error listing Pods: %v\n", err)
		os.Exit(3)
	}

	newPodItems := []corev1.Pod{}

	nodes := map[string]bool{}
	for _, node := range nodeList.Items {
		nodes[node.GetName()] = true
	}

	for _, pod := range podList.Items {
		if ! nodes[pod.Spec.NodeName] {
			continue
		}

		newPodItems = append(newPodItems, pod)
	}

	podList.Items = newPodItems

	if namespaceLabels != "" {
		namespaceList, err := clientset.CoreV1().Namespaces().List(metav1.ListOptions{
			LabelSelector: namespaceLabels,
		})
		if err != nil {
			fmt.Printf("Error listing Namespaces: %v\n", err)
			os.Exit(3)
		}

		namespaces := map[string]bool{}
		for _, ns := range namespaceList.Items {
			namespaces[ns.GetName()] = true
		}

		newPodItems := []corev1.Pod{}

		for _, pod := range podList.Items {
			if ! namespaces[pod.GetNamespace()] {
				continue
			}

			newPodItems = append(newPodItems, pod)
		}

		podList.Items = newPodItems
	}

	return podList, nodeList
}

func getMetrics(mClientset *metrics.Clientset) *v1beta1.PodMetricsList {
	pmList, err := mClientset.MetricsV1beta1().PodMetricses("").List(metav1.ListOptions{})
	if err != nil {
		fmt.Printf("Error getting Pod Metrics: %v\n", err)
		fmt.Println("For this to work, metrics-server needs to be running in your cluster")
		os.Exit(6)
	}

	return pmList
}

func buildClusterMetric(podList *corev1.PodList, pmList *v1beta1.PodMetricsList, nodeList *corev1.NodeList) clusterMetric {
	cm := clusterMetric{
		cpu:         &resourceMetric{resourceType: "cpu"},
		memory:      &resourceMetric{resourceType: "memory"},
		nodeMetrics: map[string]*nodeMetric{},
		podMetrics:  map[string]*podMetric{},
	}

	for _, node := range nodeList.Items {
		cm.nodeMetrics[node.Name] = &nodeMetric{
			cpu: &resourceMetric{
				resourceType: "cpu",
				allocatable:  node.Status.Allocatable["cpu"],
			},
			memory: &resourceMetric{
				resourceType: "memory",
				allocatable:  node.Status.Allocatable["memory"],
			},
			podMetrics: map[string]*podMetric{},
		}

		cm.cpu.allocatable.Add(node.Status.Allocatable["cpu"])
		cm.memory.allocatable.Add(node.Status.Allocatable["memory"])
	}

	podMetrics := map[string]v1beta1.PodMetrics{}
	for _, pm := range pmList.Items {
		podMetrics[fmt.Sprintf("%s-%s", pm.GetNamespace(), pm.GetName())] = pm
	}

	for _, pod := range podList.Items {
		if pod.Status.Phase != corev1.PodSucceeded && pod.Status.Phase != corev1.PodFailed {
			cm.addPodMetric(&pod, podMetrics[fmt.Sprintf("%s-%s", pod.GetNamespace(), pod.GetName())])
		}
	}

	return cm
}
