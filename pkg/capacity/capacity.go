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
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/robscott/kube-capacity/pkg/kube"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	v1beta1 "k8s.io/metrics/pkg/apis/metrics/v1beta1"
	metrics "k8s.io/metrics/pkg/client/clientset/versioned"
)

// FetchAndPrint gathers cluster resource data and outputs it
func FetchAndPrint(opts Options) {
	clientset, err := kube.NewClientSet(opts.KubeConfig, opts.KubeContext, opts.ImpersonateUser, opts.ImpersonateGroup)
	if err != nil {
		fmt.Printf("Error connecting to Kubernetes: %v\n", err)
		os.Exit(1)
	}

	podList, nodeList := getPodsAndNodes(clientset, opts.ExcludeTainted, opts.PodLabels, opts.NodeLabels, opts.NamespaceLabels, opts.Namespace)
	var pmList *v1beta1.PodMetricsList
	var nmList *v1beta1.NodeMetricsList

	if opts.ShowUtil {
		mClientset, err := kube.NewMetricsClientSet(opts.KubeContext, opts.KubeConfig)
		if err != nil {
			fmt.Printf("Error connecting to Metrics API: %v\n", err)
			os.Exit(4)
		}

		pmList = getPodMetrics(mClientset, opts.Namespace)
		if opts.Namespace == "" && opts.NamespaceLabels == "" {
			nmList = getNodeMetrics(mClientset, opts.NodeList, opts.NodeLabels, opts.NodeTaints)
		}
	}

	cm := buildClusterMetric(podList, pmList, nodeList, nmList)
	showNamespace := opts.Namespace == ""

	printList(&cm, opts.ShowContainers, opts.ShowPods, opts.ShowUtil, opts.ShowPodCount, showNamespace, opts.OutputFormat, opts.SortBy, opts.AvailableFormat)
}

func splitTaint(taint string) (string, string, string) {
	var key, value, effect string
	var parts []string

	if strings.Contains(taint, "=") && strings.Contains(taint, ":") {
		parts = strings.Split(taint, "=")
		key = parts[0]
		parts = strings.Split(parts[1], ":")
		value = parts[0]
		effect = parts[1]
		return key, value, effect
	}

	if strings.Contains(taint, ":") {

		parts = strings.Split(taint, ":")
		key = parts[0]
		effect = parts[1]
		value = ""
		return key, value, effect
	}

	if strings.Contains(taint, "=") {
		parts = strings.Split(taint, "=")
		key = parts[0]
		value = parts[1]
		effect = ""
		return key, value, effect
	}

	return taint, "", ""
}

func removeNodesWithTaints(nodeList corev1.NodeList, taints []string) corev1.NodeList {
	var tempNodeList corev1.NodeList
	var key, value, effect string
	isTainted := false

	for _, node := range nodeList.Items {
		for _, taint := range taints {
			key, value, effect = splitTaint(taint)
			for _, t := range node.Spec.Taints {
				if t.Key == key && t.Value == value && t.Effect == corev1.TaintEffect(effect) {
					isTainted = true
					break
				}
			}
			if isTainted {
				break
			}
		}
		if !isTainted {
			tempNodeList.Items = append(tempNodeList.Items, node)
		}
		isTainted = false
	}

	return tempNodeList
}

func addNodesWithTaints(nodeList corev1.NodeList, taints []string) corev1.NodeList {
	var tempNodeList corev1.NodeList
	var key, value, effect string
	var nodeCount int

	for _, node := range nodeList.Items {
	outer:
		for _, taint := range taints {
			key, value, effect = splitTaint(taint)
			for _, t := range node.Spec.Taints {
				if t.Key == key && t.Value == value && t.Effect == corev1.TaintEffect(effect) {
					tempNodeList.Items = append(tempNodeList.Items, node)
					nodeCount = nodeCount + 1
					break outer
				}
			}
		}
	}

	if nodeCount == 0 {
		fmt.Printf("Error getting nodes with taints.")
		os.Exit(6)
	}

	return tempNodeList
}

func splitTaintsByAddRemove(taints string) (taintsToAdd []string, taintsToRemove []string) {
	taintSlice := strings.Split(taints, ",")
	for _, taint := range taintSlice {
		trimmedTaint := strings.TrimSpace(taint)
		if strings.HasPrefix(trimmedTaint, "!") {
			taintsToRemove = append(taintsToRemove, trimmedTaint[1:])
		} else {
			taintsToAdd = append(taintsToAdd, trimmedTaint)
		}
	}
	return taintsToAdd, taintsToRemove
}

func getPodsAndNodes(clientset kubernetes.Interface, excludeTainted bool, podLabels, nodeLabels, nodeTaints, namespaceLabels, namespace string) (*corev1.PodList, *corev1.NodeList) {
	nodeList, err := clientset.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{
		LabelSelector: nodeLabels,
	})
	if err != nil {
		fmt.Printf("Error listing Nodes: %v\n", err)
		os.Exit(2)
	}
	if excludeTainted {
		filteredNodeList := []corev1.Node{}
		for _, node := range nodeList.Items {
			if len(node.Spec.Taints) == 0 {
				filteredNodeList = append(filteredNodeList, node)
			}
		}
		nodeList.Items = filteredNodeList
	}

	if nodeTaints != "" {
		taintedNodes := *nodeList
		taintsToAdd, taintsToRemove := splitTaintsByAddRemove(nodeTaints)

		if len(taintsToAdd) > 0 {
			taintedNodes = addNodesWithTaints(taintedNodes, taintsToAdd)
		}

		if len(taintsToRemove) > 0 {
			taintedNodes = removeNodesWithTaints(taintedNodes, taintsToRemove)
		}
		if err != nil {
			fmt.Printf("Error removing tained Nodes: %v\n", err)
			os.Exit(2)
		}
		*nodeList = taintedNodes
	}

	podList, err := clientset.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{
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
		if !nodes[pod.Spec.NodeName] {
			continue
		}

		newPodItems = append(newPodItems, pod)
	}

	podList.Items = newPodItems

	if namespace == "" && namespaceLabels != "" {
		namespaceList, err := clientset.CoreV1().Namespaces().List(context.TODO(), metav1.ListOptions{
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
			if !namespaces[pod.GetNamespace()] {
				continue
			}

			newPodItems = append(newPodItems, pod)
		}

		podList.Items = newPodItems
	}

	return podList, nodeList
}

func removeNodeMetricsWithTaints(nmList *v1beta1.NodeMetricsList, nodeList corev1.NodeList) v1beta1.NodeMetricsList {
	var tempNmList v1beta1.NodeMetricsList

	for _, node := range nodeList.Items {
		for _, nm := range nmList.Items {
			if node.Name == nm.Name {
				tempNmList.Items = append(tempNmList.Items, nm)
			}
		}
	}

	return tempNmList
}

func getPodMetrics(mClientset *metrics.Clientset, namespace string) *v1beta1.PodMetricsList {
	pmList, err := mClientset.MetricsV1beta1().PodMetricses(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		fmt.Printf("Error getting Pod Metrics: %v\n", err)
		fmt.Println("For this to work, metrics-server needs to be running in your cluster")
		os.Exit(6)
	}

	return pmList
}

func getNodeMetrics(mClientset *metrics.Clientset, nodeList *corev1.NodeList, nodeLabels string, nodeTaints string) *v1beta1.NodeMetricsList {
	nmList, err := mClientset.MetricsV1beta1().NodeMetricses().List(context.TODO(), metav1.ListOptions{
		LabelSelector: nodeLabels,
	})

	if nodeTaints != "" {
		taintedNodeList := *nodeList
		taintedNmList := removeNodeMetricsWithTaints(nmList, taintedNodeList)
		if err != nil {
			fmt.Printf("Error removing tainted Nodes: %v\n", err)
			os.Exit(2)
		}
		*nmList = taintedNmList
	}

	if err != nil {
		fmt.Printf("Error getting Node Metrics: %v\n", err)
		fmt.Println("For this to work, metrics-server needs to be running in your cluster")
		os.Exit(7)
	}

	return nmList
}
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
	"context"
	"fmt"
	"os"
	"strings"

	"k8s.io/client-go/kubernetes"
	metrics "k8s.io/metrics/pkg/client/clientset/versioned"

	"github.com/robscott/kube-capacity/pkg/kube"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1beta1 "k8s.io/metrics/pkg/apis/metrics/v1beta1"
)

// FetchAndPrint gathers cluster resource data and outputs it
func FetchAndPrint(showContainers, showPods, showUtil, showPodCount, excludeTainted, availableFormat bool, podLabels, nodeLabels, nodeTaints, namespaceLabels, namespace, kubeContext, kubeConfig, output, sortBy string) {
	clientset, err := kube.NewClientSet(kubeContext, kubeConfig)
	if err != nil {
		fmt.Printf("Error connecting to Kubernetes: %v\n", err)
		os.Exit(1)
	}

	podList, nodeList := getPodsAndNodes(clientset, excludeTainted, podLabels, nodeLabels, nodeTaints, namespaceLabels, namespace)
	var pmList *v1beta1.PodMetricsList
	var nmList *v1beta1.NodeMetricsList

	if showUtil {
		mClientset, err := kube.NewMetricsClientSet(kubeContext, kubeConfig)
		if err != nil {
			fmt.Printf("Error connecting to Metrics API: %v\n", err)
			os.Exit(4)
		}

		pmList = getPodMetrics(mClientset, namespace)
		if namespace == "" && namespaceLabels == "" {
			nmList = getNodeMetrics(mClientset, nodeList, nodeLabels, nodeTaints)
		}
	}

	cm := buildClusterMetric(podList, pmList, nodeList, nmList)
	showNamespace := namespace == ""

	printList(&cm, showContainers, showPods, showUtil, showPodCount, showNamespace, output, sortBy, availableFormat)
}

// Taints can have two possible formats, key1:effect or key1=value1:effect. This function handles both formats and returns each part as a separate string for comparison.
func splitTaint(taint string) (string, string, string) {
	var key, value, effect string
	var parts []string

	if strings.Contains(taint, "=") && strings.Contains(taint, ":") {
		parts = strings.Split(taint, "=")
		key = parts[0]
		parts = strings.Split(parts[1], ":")
		value = parts[0]
		effect = parts[1]
		return key, value, effect
	}

	if strings.Contains(taint, ":") {

		parts = strings.Split(taint, ":")
		key = parts[0]
		effect = parts[1]
		value = ""
		return key, value, effect
	}
}

// This loops through the original nodeList from getPodsAndNodes and checks each node individually for the list of taints. If a node contains any taint in the list, it is removedfrom nodeList, otherwise it is remains.
func removeNodesWithTaints(nodeList corev1.NodeList, taints []string) corev1.NodeList {
	var tempNodeList corev1.NodeList
	var key, value, effect string
	isTainted := false

	for _, node := range nodeList.Items {
		for _, taint := range taints {
			key, value, effect = splitTaint(taint)
			for _, t := range node.Spec.Taints {
				if t.Key == key && t.Value == value && t.Effect == corev1.TaintEffect(effect) {
					isTainted = true
					break
				}
			}
			if isTainted {
				break
			}
		}
		if !isTainted {
			tempNodeList.Items = append(tempNodeList.Items, node)
		}
		isTainted = false
	}

	return tempNodeList
}

// This loops through the original nodeList from getPodsAndNodes and checks each node individually for the list of taints. If a node contains any taint in the list, it remains as part of nodeList, otherwise it is removed.
func addNodesWithTaints(nodeList corev1.NodeList, taints []string) corev1.NodeList {
	var tempNodeList corev1.NodeList
	var key, value, effect string
	var nodeCount int

	for _, node := range nodeList.Items {
	outer:
		for _, taint := range taints {
			key, value, effect = splitTaint(taint)
			for _, t := range node.Spec.Taints {
				if t.Key == key && t.Value == value && t.Effect == corev1.TaintEffect(effect) {
					tempNodeList.Items = append(tempNodeList.Items, node)
					nodeCount = nodeCount + 1
					break outer
				}
			}
		}
	}

	if nodeCount == 0 {
		fmt.Printf("Error getting nodes with taints.")
		os.Exit(6)
	}

	return tempNodeList
}

// The initial list of taints is supplied as a comma delimited string. Taints that need to be excluded are prefixed with an !, otherwise they are included.
func splitTaintsByAddRemove(taints string) (taintsToAdd []string, taintsToRemove []string) {
	taintSlice := strings.Split(taints, ",")
	for _, taint := range taintSlice {
		trimmedTaint := strings.TrimSpace(taint)
		if strings.HasPrefix(trimmedTaint, "!") {
			taintsToRemove = append(taintsToRemove, trimmedTaint[1:])
		} else {
			taintsToAdd = append(taintsToAdd, trimmedTaint)
		}
	}
	return taintsToAdd, taintsToRemove
}

func getPodsAndNodes(clientset kubernetes.Interface, excludeTainted bool, podLabels, nodeLabels, nodeTaints, namespaceLabels, namespace string) (*corev1.PodList, *corev1.NodeList) {
	nodeList, err := clientset.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{
		LabelSelector: nodeLabels,
	})
	if err != nil {
		fmt.Printf("Error listing Nodes: %v\n", err)
		os.Exit(2)
	}
	if excludeTainted {
		filteredNodeList := []corev1.Node{}
		for _, node := range nodeList.Items {
			if len(node.Spec.Taints) == 0 {
				filteredNodeList = append(filteredNodeList, node)
			}
		}
		nodeList.Items = filteredNodeList
	}

	// It is not possible to filter nodeList based on taints, so if we want to filter based on taints, we need to check each node individually and check it's taints.
	if nodeTaints != "" {
		taintedNodes := *nodeList
		taintsToAdd, taintsToRemove := splitTaintsByAddRemove(nodeTaints)

		if len(taintsToAdd) > 0 {
			taintedNodes = addNodesWithTaints(taintedNodes, taintsToAdd)
		}

		if len(taintsToRemove) > 0 {
			taintedNodes = removeNodesWithTaints(taintedNodes, taintsToRemove)
		}
		if err != nil {
			fmt.Printf("Error removing tained Nodes: %v\n", err)
			os.Exit(2)
		}
		*nodeList = taintedNodes
	}

	podList, err := clientset.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{
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
		if !nodes[pod.Spec.NodeName] {
			continue
		}

		newPodItems = append(newPodItems, pod)
	}

	podList.Items = newPodItems

	if namespace == "" && namespaceLabels != "" {
		namespaceList, err := clientset.CoreV1().Namespaces().List(context.TODO(), metav1.ListOptions{
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
			if !namespaces[pod.GetNamespace()] {
				continue
			}

			newPodItems = append(newPodItems, pod)
		}

		podList.Items = newPodItems
	}

	return podList, nodeList
}

func removeNodeMetricsWithTaints(nmList *v1beta1.NodeMetricsList, nodeList corev1.NodeList) v1beta1.NodeMetricsList {
	var tempNmList v1beta1.NodeMetricsList

	for _, node := range nodeList.Items {
		for _, nm := range nmList.Items {
			if node.Name == nm.Name {
				tempNmList.Items = append(tempNmList.Items, nm)
			}
		}
	}

	return tempNmList
}

func getPodMetrics(mClientset *metrics.Clientset, namespace string) *v1beta1.PodMetricsList {
	pmList, err := mClientset.MetricsV1beta1().PodMetricses(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		fmt.Printf("Error getting Pod Metrics: %v\n", err)
		fmt.Println("For this to work, metrics-server needs to be running in your cluster")
		os.Exit(6)
	}

	return pmList
}

func getNodeMetrics(mClientset *metrics.Clientset, nodeList *corev1.NodeList, nodeLabels string, nodeTaints string) *v1beta1.NodeMetricsList {
	nmList, err := mClientset.MetricsV1beta1().NodeMetricses().List(context.TODO(), metav1.ListOptions{
		LabelSelector: nodeLabels,
	})

	if nodeTaints != "" {
		taintedNodeList := *nodeList
		taintedNmList := removeNodeMetricsWithTaints(nmList, taintedNodeList)
		if err != nil {
			fmt.Printf("Error removing tainted Nodes: %v\n", err)
			os.Exit(2)
		}
		*nmList = taintedNmList
	}

	if err != nil {
		fmt.Printf("Error getting Node Metrics: %v\n", err)
		fmt.Println("For this to work, metrics-server needs to be running in your cluster")
		os.Exit(7)
	}

	return nmList
}
