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
)

func TestGetLineItems(t *testing.T) {
	tpNone := &tablePrinter{
		opts: Options{
			ShowPods:       false,
			ShowUtil:       false,
			ShowPodCount:   false,
			ShowContainers: false,
			Namespace:      "example",
		},
	}

	tpSome := &tablePrinter{
		opts: Options{
			ShowPods:       false,
			ShowUtil:       false,
			ShowPodCount:   false,
			ShowLabels:     false,
			ShowContainers: true,
			Namespace:      "",
		},
	}

	tpAll := &tablePrinter{
		opts: Options{
			ShowPods:       true,
			ShowUtil:       true,
			ShowContainers: true,
			Namespace:      "",
			ShowPodCount:   true,
			ShowLabels:     true,
		},
	}

	tl := &tableLine{
		node:           "example-node-1",
		namespace:      "example-namespace",
		pod:            "nginx-fsde",
		container:      "nginx",
		cpuRequests:    "100m",
		cpuLimits:      "200m",
		cpuUtil:        "14m",
		memoryRequests: "1000Mi",
		memoryLimits:   "2000Mi",
		memoryUtil:     "326Mi",
		podCount:       "1/110",
		labels:         "zone=example-zone-1",
	}

	var testCases = []struct {
		name     string
		tp       *tablePrinter
		tl       *tableLine
		expected []string
	}{
		{
			name: "all false",
			tp:   tpNone,
			tl:   tl,
			expected: []string{
				"example-node-1",
				"100m",
				"200m",
				"1000Mi",
				"2000Mi",
			},
		}, {
			name: "some true",
			tp:   tpSome,
			tl:   tl,
			expected: []string{
				"example-node-1",
				"example-namespace",
				"nginx-fsde",
				"nginx",
				"100m",
				"200m",
				"1000Mi",
				"2000Mi",
			},
		}, {
			name: "all true",
			tp:   tpAll,
			tl:   tl,
			expected: []string{
				"example-node-1",
				"example-namespace",
				"nginx-fsde",
				"nginx",
				"100m",
				"200m",
				"14m",
				"1000Mi",
				"2000Mi",
				"326Mi",
				"1/110",
				"zone=example-zone-1",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			lineItems := tc.tp.getLineItems(tl)
			assert.ElementsMatchf(t, lineItems, tc.expected, "Expected: %+v\nGot:      %+v", tc.expected, lineItems)
		})
	}
}
