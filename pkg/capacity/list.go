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
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"text/tabwriter"
	"time"

	"github.com/robscott/kube-capacity/pkg/kube"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
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

	cr := clusterResource{
		cpuAllocatable: resource.Quantity{},
		cpuRequest:     resource.Quantity{},
		cpuLimit:       resource.Quantity{},
		memAllocatable: resource.Quantity{},
		memRequest:     resource.Quantity{},
		memLimit:       resource.Quantity{},
		capacityByNode: map[string]*nodeResource{},
	}

	for _, node := range nodeList.Items {
		cr.capacityByNode[node.Name] = &nodeResource{
			cpuAllocatable: node.Status.Allocatable["cpu"],
			cpuRequest:     resource.Quantity{},
			cpuLimit:       resource.Quantity{},
			memAllocatable: node.Status.Allocatable["memory"],
			memRequest:     resource.Quantity{},
			memLimit:       resource.Quantity{},
			podResources:   []podResource{},
		}

		for _, pod := range podList.Items {
			n, ok := cr.capacityByNode[pod.Spec.NodeName]
			if ok {
				n.addPodResources(&pod)
			}
		}

		cr.addNodeCapacity(cr.capacityByNode[node.Name])
	}

	nmList, err := mClientset.MetricsV1beta1().NodeMetricses().List(metav1.ListOptions{})
	if err != nil {
		fmt.Println("Error getting metrics")
		panic(err.Error())
	}

	fmt.Printf("========> %#v\n", nmList)

	for _, nm := range nmList.Items {
		fmt.Printf("nm =============> %#v\n", nm)
	}

	pmList, err := mClientset.MetricsV1beta1().PodMetricses("").List(metav1.ListOptions{})
	if err != nil {
		fmt.Println("Error getting metrics")
		panic(err.Error())
	}

	fmt.Printf("pmList ========> %#v\n", pmList)

	for _, pm := range pmList.Items {
		fmt.Printf("nm =============> %#v\n", pm)
	}

	printList(cr, outputFormat)
}

func printList(cr clusterResource, outputFormat string) {
	names := make([]string, len(cr.capacityByNode))

	i := 0
	for name := range cr.capacityByNode {
		names[i] = name
		i++
	}
	sort.Strings(names)

	w := new(tabwriter.Writer)
	w.Init(os.Stdout, 0, 8, 2, ' ', 0)

	if outputFormat == "wide" {
		fmt.Fprintln(w, "NODE\t NAMESPACE\t POD\t CPU REQUESTS \t CPU LIMITS \t MEMORY REQUESTS \t MEMORY LIMITS")

		if len(names) > 1 {
			fmt.Fprintf(w, "* \t *\t *\t %s \t %s \t %s \t %s \n",
				cr.cpuRequestString(), cr.cpuLimitString(),
				cr.memRequestString(), cr.memLimitString())
		}
	} else {
		fmt.Fprintln(w, "NODE\t CPU REQUESTS \t CPU LIMITS \t MEMORY REQUESTS \t MEMORY LIMITS")

		if len(names) > 1 {
			fmt.Fprintf(w, "* \t %s \t %s \t %s \t %s \n",
				cr.cpuRequestString(), cr.cpuLimitString(),
				cr.memRequestString(), cr.memLimitString())
		}
	}

	for _, name := range names {
		cap := cr.capacityByNode[name]

		if outputFormat == "wide" {
			fmt.Fprintf(w, "%s \t *\t *\t %s \t %s \t %s \t %s \n", name,
				cap.cpuRequestString(), cap.cpuLimitString(),
				cap.memRequestString(), cap.memLimitString())

			for _, pod := range cap.podResources {
				fmt.Fprintf(w, "%s \t %s \t %s \t %s \t %s \t %s \t %s \n", name,
					pod.namespace, pod.name,
					pod.cpuRequestString(cap), pod.cpuLimitString(cap),
					pod.memRequestString(cap), pod.memLimitString(cap))
			}
			fmt.Fprintln(w)
		} else {
			fmt.Fprintf(w, "%s \t %s \t %s \t %s \t %s \n", name,
				cap.cpuRequestString(), cap.cpuLimitString(),
				cap.memRequestString(), cap.memLimitString())
		}
	}

	w.Flush()
}

// PodMetricsList : PodMetricsList
type PodMetricsList struct {
	Kind       string `json:"kind"`
	APIVersion string `json:"apiVersion"`
	Metadata   struct {
		SelfLink string `json:"selfLink"`
	} `json:"metadata"`
	Items []struct {
		Metadata struct {
			Name              string    `json:"name"`
			Namespace         string    `json:"namespace"`
			SelfLink          string    `json:"selfLink"`
			CreationTimestamp time.Time `json:"creationTimestamp"`
		} `json:"metadata"`
		Timestamp  time.Time `json:"timestamp"`
		Window     string    `json:"window"`
		Containers []struct {
			Name  string `json:"name"`
			Usage struct {
				CPU    string `json:"cpu"`
				Memory string `json:"memory"`
			} `json:"usage"`
		} `json:"containers"`
	} `json:"items"`
}

func getMetrics(clientset *kubernetes.Clientset) (*PodMetricsList, error) {
	data, err := clientset.RESTClient().Get().AbsPath("apis/metrics.k8s.io/v1beta1/pods").DoRaw()
	if err != nil {
		return nil, err
	}
	pods := &PodMetricsList{}
	err = json.Unmarshal(data, pods)
	return pods, err
}
