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
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func List(args []string, outputFormat string) {
	clientset, err := kube.NewClientSet()
	if err != nil {
		fmt.Println("Error connecting to Kubernetes")
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

	allocByNode := map[string]*nodeResource{}

	for _, node := range nodeList.Items {
		allocByNode[node.Name] = &nodeResource{
			cpuAllocatable: node.Status.Allocatable["cpu"],
			cpuRequest:     resource.Quantity{},
			cpuLimit:       resource.Quantity{},
			memAllocatable: node.Status.Allocatable["memory"],
			memRequest:     resource.Quantity{},
			memLimit:       resource.Quantity{},
			podResources:   []podResource{},
		}
	}

	for _, pod := range podList.Items {
		n, ok := allocByNode[pod.Spec.NodeName]
		if ok {
			n.addPodResources(&pod)
		}
	}

	printList(allocByNode)
}

func printList(allocByNode map[string]*nodeResource) {
	names := make([]string, len(allocByNode))

	i := 0
	for name := range allocByNode {
		names[i] = name
		i++
	}
	sort.Strings(names)

	w := new(tabwriter.Writer)
	w.Init(os.Stdout, 0, 8, 2, ' ', 0)
	fmt.Fprintln(w, "NODE\t CPU REQUESTS \t CPU LIMITS \t MEMORY REQUESTS \t MEMORY LIMITS")

	for _, name := range names {
		alloc := allocByNode[name]
		fmt.Fprintf(w, "%s \t %s \t %s \t %s \t %s \n", name, alloc.cpuRequestString(), alloc.cpuLimitString(), alloc.memRequestString(), alloc.memLimitString())
	}

	w.Flush()
}
