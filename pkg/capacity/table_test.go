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

func TestGetLineItemsWithColumns(t *testing.T) {
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
			name: "single column: mem.util",
			tp: &tablePrinter{
				opts: Options{
					Columns: "mem.util",
				},
			},
			tl: tl,
			expected: []string{
				"example-node-1",
				"326Mi",
			},
		},
		{
			name: "multiple columns: cpu.util,mem.util",
			tp: &tablePrinter{
				opts: Options{
					Columns: "cpu.util,mem.util",
				},
			},
			tl: tl,
			expected: []string{
				"example-node-1",
				"14m",
				"326Mi",
			},
		},
		{
			name: "all columns via --columns",
			tp: &tablePrinter{
				opts: Options{
					Columns: "cpu.request,cpu.limit,cpu.util,mem.request,mem.limit,mem.util,pod.count,labels",
				},
			},
			tl: tl,
			expected: []string{
				"example-node-1",
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
		{
			name: "columns with pods view",
			tp: &tablePrinter{
				opts: Options{
					Columns:   "cpu.util,mem.util",
					ShowPods:  true,
					Namespace: "",
				},
			},
			tl: tl,
			expected: []string{
				"example-node-1",
				"example-namespace",
				"nginx-fsde",
				"14m",
				"326Mi",
			},
		},
		{
			name: "columns with containers view",
			tp: &tablePrinter{
				opts: Options{
					Columns:        "cpu.util",
					ShowContainers: true,
					Namespace:      "",
				},
			},
			tl: tl,
			expected: []string{
				"example-node-1",
				"example-namespace",
				"nginx-fsde",
				"nginx",
				"14m",
			},
		},
		{
			name: "columns with namespace filter omits namespace column",
			tp: &tablePrinter{
				opts: Options{
					Columns:   "mem.util",
					ShowPods:  true,
					Namespace: "example",
				},
			},
			tl: tl,
			expected: []string{
				"example-node-1",
				"nginx-fsde",
				"326Mi",
			},
		},
		{
			name: "columns override hide-requests and hide-limits",
			tp: &tablePrinter{
				opts: Options{
					Columns:      "cpu.request,mem.limit",
					HideRequests: true,
					HideLimits:   true,
				},
			},
			tl: tl,
			expected: []string{
				"example-node-1",
				"100m",
				"2000Mi",
			},
		},
		{
			name: "columns with spaces around comma",
			tp: &tablePrinter{
				opts: Options{
					Columns: "cpu.util , mem.util",
				},
			},
			tl: tl,
			expected: []string{
				"example-node-1",
				"14m",
				"326Mi",
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

func TestValidateColumns(t *testing.T) {
	var testCases = []struct {
		name    string
		columns string
		wantErr bool
	}{
		{
			name:    "empty is valid",
			columns: "",
			wantErr: false,
		},
		{
			name:    "single valid column",
			columns: "cpu.util",
			wantErr: false,
		},
		{
			name:    "multiple valid columns",
			columns: "cpu.util,mem.util,pod.count",
			wantErr: false,
		},
		{
			name:    "all valid columns",
			columns: "cpu.request,cpu.limit,cpu.util,mem.request,mem.limit,mem.util,pod.count,labels",
			wantErr: false,
		},
		{
			name:    "invalid column name",
			columns: "cpu.util,badcolumn",
			wantErr: true,
		},
		{
			name:    "typo in column name",
			columns: "cpu.utl",
			wantErr: true,
		},
		{
			name:    "empty column in list",
			columns: "cpu.util,,mem.util",
			wantErr: false,
		},
		{
			name:    "column with spaces",
			columns: " cpu.util , mem.util ",
			wantErr: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateColumns(tc.columns)
			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestHasVisibleColumns(t *testing.T) {
	var testCases = []struct {
		name     string
		opts     Options
		expected bool
	}{
		{
			name:     "default options have visible columns",
			opts:     Options{},
			expected: true,
		},
		{
			name: "all hidden has no visible columns",
			opts: Options{
				HideRequests: true,
				HideLimits:   true,
			},
			expected: false,
		},
		{
			name: "columns flag always has visible columns",
			opts: Options{
				HideRequests: true,
				HideLimits:   true,
				Columns:      "cpu.util",
			},
			expected: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tp := &tablePrinter{opts: tc.opts}
			assert.Equal(t, tc.expected, tp.hasVisibleColumns())
		})
	}
}
