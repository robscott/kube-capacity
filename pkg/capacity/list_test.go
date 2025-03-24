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
	"testing"

	"github.com/stretchr/testify/assert"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1beta1 "k8s.io/metrics/pkg/apis/metrics/v1beta1"
)

func TestBuildListClusterMetricsNoOptions(t *testing.T) {
	cm := getTestClusterMetric()

	lp := listPrinter{
		cm: &cm,
	}

	lcm := lp.buildListClusterMetrics()

	assert.EqualValues(t, &listClusterTotals{
		CPU: &listResourceOutput{
			Requests:    "650m",
			RequestsPct: "65%",
			Limits:      "810m",
			LimitsPct:   "81%",
		},
		Memory: &listResourceOutput{
			Requests:    "410Mi",
			RequestsPct: "10%",
			Limits:      "580Mi",
			LimitsPct:   "14%",
		},
	}, lcm.ClusterTotals)

	assert.EqualValues(t, &listNodeMetric{
		Name: "example-node-1",
		CPU: &listResourceOutput{
			Requests:    "650m",
			RequestsPct: "65%",
			Limits:      "810m",
			LimitsPct:   "81%",
		},
		Memory: &listResourceOutput{
			Requests:    "410Mi",
			RequestsPct: "10%",
			Limits:      "580Mi",
			LimitsPct:   "14%",
		},
	}, lcm.Nodes[0])

}

func TestBuildListClusterMetricsAllOptions(t *testing.T) {
	cm := getTestClusterMetric()

	lp := listPrinter{
		cm: &cm,
		opts: Options{
			ShowUtil:       true,
			ShowPods:       true,
			ShowContainers: true,
			ShowPodCount:   true,
			ShowLabels:     true,
		},
	}

	lcm := lp.buildListClusterMetrics()

	assert.EqualValues(t, &listClusterTotals{
		CPU: &listResourceOutput{
			Requests:       "650m",
			RequestsPct:    "65%",
			Limits:         "810m",
			LimitsPct:      "81%",
			Utilization:    "63m",
			UtilizationPct: "6%",
		},
		Memory: &listResourceOutput{
			Requests:       "410Mi",
			RequestsPct:    "10%",
			Limits:         "580Mi",
			LimitsPct:      "14%",
			Utilization:    "439Mi",
			UtilizationPct: "10%",
		},
		PodCount: "1/110",
	}, lcm.ClusterTotals)

	assert.EqualValues(t, &listNodeMetric{
		Name:     "example-node-1",
		PodCount: "1/110",
		Labels:   map[string]string{"example.io/os": "example-os-1", "zone": "example-zone-1"},
		CPU: &listResourceOutput{
			Requests:       "650m",
			RequestsPct:    "65%",
			Limits:         "810m",
			LimitsPct:      "81%",
			Utilization:    "63m",
			UtilizationPct: "6%",
		},
		Memory: &listResourceOutput{
			Requests:       "410Mi",
			RequestsPct:    "10%",
			Limits:         "580Mi",
			LimitsPct:      "14%",
			Utilization:    "439Mi",
			UtilizationPct: "10%",
		},
		Pods: []*listPod{
			{
				Name:      "example-pod",
				Namespace: "default",
				CPU: &listResourceOutput{
					Requests:       "650m",
					RequestsPct:    "65%",
					Limits:         "810m",
					LimitsPct:      "81%",
					Utilization:    "63m",
					UtilizationPct: "6%",
				},
				Memory: &listResourceOutput{
					Requests:       "410Mi",
					RequestsPct:    "10%",
					Limits:         "580Mi",
					LimitsPct:      "14%",
					Utilization:    "439Mi",
					UtilizationPct: "10%",
				},
				Containers: []listContainer{
					{
						Name: "example-container-1",
						CPU: &listResourceOutput{
							Requests:       "450m",
							RequestsPct:    "45%",
							Limits:         "560m",
							LimitsPct:      "56%",
							Utilization:    "40m",
							UtilizationPct: "4%",
						},
						Memory: &listResourceOutput{
							Requests:       "160Mi",
							RequestsPct:    "4%",
							Limits:         "280Mi",
							LimitsPct:      "7%",
							Utilization:    "288Mi",
							UtilizationPct: "7%",
						},
					}, {
						Name: "example-container-2",
						CPU: &listResourceOutput{
							Requests:       "200m",
							RequestsPct:    "20%",
							Limits:         "250m",
							LimitsPct:      "25%",
							Utilization:    "23m",
							UtilizationPct: "2%",
						},
						Memory: &listResourceOutput{
							Requests:       "250Mi",
							RequestsPct:    "6%",
							Limits:         "300Mi",
							LimitsPct:      "7%",
							Utilization:    "151Mi",
							UtilizationPct: "3%",
						},
					},
				},
			},
		}}, lcm.Nodes[0])
}

func getTestClusterMetric() clusterMetric {
	return buildClusterMetric(
		&corev1.PodList{
			Items: []corev1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "example-pod",
						Namespace: "default",
					},
					Spec: corev1.PodSpec{
						NodeName: "example-node-1",
						Containers: []corev1.Container{
							{
								Name: "example-container-1",
								Resources: corev1.ResourceRequirements{
									Requests: corev1.ResourceList{
										"cpu":    resource.MustParse("450m"),
										"memory": resource.MustParse("160Mi"),
									},
									Limits: corev1.ResourceList{
										"cpu":    resource.MustParse("560m"),
										"memory": resource.MustParse("280Mi"),
									},
								},
							},
							{
								Name: "example-container-2",
								Resources: corev1.ResourceRequirements{
									Requests: corev1.ResourceList{
										"cpu":    resource.MustParse("200m"),
										"memory": resource.MustParse("250Mi"),
									},
									Limits: corev1.ResourceList{
										"cpu":    resource.MustParse("250m"),
										"memory": resource.MustParse("300Mi"),
									},
								},
							},
						},
					},
				},
			},
		}, &v1beta1.PodMetricsList{
			Items: []v1beta1.PodMetrics{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "example-pod",
						Namespace: "default",
					},
					Containers: []v1beta1.ContainerMetrics{
						{
							Name: "example-container-1",
							Usage: corev1.ResourceList{
								"cpu":    resource.MustParse("40m"),
								"memory": resource.MustParse("288Mi"),
							},
						},
						{
							Name: "example-container-2",
							Usage: corev1.ResourceList{
								"cpu":    resource.MustParse("23m"),
								"memory": resource.MustParse("151Mi"),
							},
						},
					},
				},
			},
		}, &corev1.NodeList{
			Items: []corev1.Node{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "example-node-1",
						Labels: map[string]string{
							"example.io/os": "example-os-1",
							"zone":          "example-zone-1",
						},
					},
					Status: corev1.NodeStatus{
						Allocatable: corev1.ResourceList{
							"cpu":    resource.MustParse("1000m"),
							"memory": resource.MustParse("4000Mi"),
							"pods":   resource.MustParse("110"),
						},
					},
				},
			},
		}, &v1beta1.NodeMetricsList{
			Items: []v1beta1.NodeMetrics{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "example-node-1",
						Labels: map[string]string{
							"example.io/os": "example-os-1",
							"zone":          "example-zone-1",
						},
					},
					Usage: corev1.ResourceList{
						"cpu":    resource.MustParse("63m"),
						"memory": resource.MustParse("439Mi"),
					},
				},
			},
		},
	)
}
