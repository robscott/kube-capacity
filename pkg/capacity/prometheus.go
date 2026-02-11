package capacity

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	v1beta1 "k8s.io/metrics/pkg/apis/metrics/v1beta1"
)

type prometheusResponse struct {
	Status string         `json:"status"`
	Data   prometheusData `json:"data"`
}

type prometheusData struct {
	ResultType string             `json:"resultType"`
	Result     []prometheusResult `json:"result"`
}

type prometheusResult struct {
	Metric map[string]string `json:"metric"`
	Value  []interface{}     `json:"value"`
}

func containerCPUQuery(agg, window string) string {
	return fmt.Sprintf(`%s_over_time(sum by (namespace, pod, container) (rate(container_cpu_usage_seconds_total{container!="",container!="POD"}[5m]))[%s:])`, agg, window)
}

func containerMemQuery(agg, window string) string {
	return fmt.Sprintf(`%s_over_time(sum by (namespace, pod, container) (container_memory_working_set_bytes{container!="",container!="POD"})[%s:])`, agg, window)
}

func nodeCPUQuery(agg, window string) string {
	return fmt.Sprintf(`%s_over_time(sum by (node) (rate(container_cpu_usage_seconds_total{container!=""}[5m]))[%s:])`, agg, window)
}

func nodeMemQuery(agg, window string) string {
	return fmt.Sprintf(`%s_over_time(sum by (node) (container_memory_working_set_bytes{container!=""})[%s:])`, agg, window)
}

var prometheusLabelSelectors = []string{
	"app.kubernetes.io/name=prometheus",
	"app=kube-prometheus-stack-prometheus",
	"operated-prometheus=true",
}

type promCandidate struct {
	namespace string
	name      string
	port      int32
}

func discoverPrometheusEndpoint(clientset kubernetes.Interface) (string, error) {
	seen := map[string]bool{}
	var candidates []promCandidate

	for _, selector := range prometheusLabelSelectors {
		svcList, err := clientset.CoreV1().Services("").List(context.TODO(), metav1.ListOptions{
			LabelSelector: selector,
		})
		if err != nil {
			continue
		}
		for _, svc := range svcList.Items {
			key := fmt.Sprintf("%s/%s", svc.Namespace, svc.Name)
			if seen[key] {
				continue
			}
			seen[key] = true

			port := int32(9090)
			found9090 := false
			for _, p := range svc.Spec.Ports {
				if p.Port == 9090 {
					found9090 = true
					break
				}
			}
			if !found9090 {
				if len(svc.Spec.Ports) > 0 {
					port = svc.Spec.Ports[0].Port
				} else {
					continue
				}
			}
			candidates = append(candidates, promCandidate{
				namespace: svc.Namespace,
				name:      svc.Name,
				port:      port,
			})
		}
	}

	if len(candidates) == 0 {
		return "", fmt.Errorf("no Prometheus service found (searched labels: %v)", prometheusLabelSelectors)
	}

	if len(candidates) > 1 {
		fmt.Fprintf(os.Stderr, "Warning: found %d Prometheus services, using first match. Use --prometheus-endpoint to specify explicitly:\n", len(candidates))
		for _, c := range candidates {
			fmt.Fprintf(os.Stderr, "  - %s/%s:%d\n", c.namespace, c.name, c.port)
		}
	}

	c := candidates[0]
	return fmt.Sprintf("%s/%s:%d", c.namespace, c.name, c.port), nil
}

func getPrometheusMetrics(clientset kubernetes.Interface, opts Options) (*v1beta1.PodMetricsList, *v1beta1.NodeMetricsList, error) {
	endpoint := opts.PrometheusEndpoint
	if endpoint == "" {
		var err error
		endpoint, err = discoverPrometheusEndpoint(clientset)
		if err != nil {
			return nil, nil, fmt.Errorf("auto-discovering Prometheus: %w", err)
		}
		fmt.Printf("Discovered Prometheus at %s\n", endpoint)
	}

	queryFn := func(query string) (*prometheusResponse, error) {
		return queryPrometheus(clientset, endpoint, query)
	}

	window := opts.PrometheusWindow
	agg := opts.PrometheusAggregation

	// Query container-level CPU and memory
	cpuResp, err := queryFn(containerCPUQuery(agg, window))
	if err != nil {
		return nil, nil, fmt.Errorf("querying container CPU: %w", err)
	}

	memResp, err := queryFn(containerMemQuery(agg, window))
	if err != nil {
		return nil, nil, fmt.Errorf("querying container memory: %w", err)
	}

	pmList := buildPodMetricsList(cpuResp, memResp)

	// Query node-level CPU and memory
	nodeCPUResp, err := queryFn(nodeCPUQuery(agg, window))
	if err != nil {
		return nil, nil, fmt.Errorf("querying node CPU: %w", err)
	}

	nodeMemResp, err := queryFn(nodeMemQuery(agg, window))
	if err != nil {
		return nil, nil, fmt.Errorf("querying node memory: %w", err)
	}

	nmList := buildNodeMetricsList(nodeCPUResp, nodeMemResp)

	return pmList, nmList, nil
}

func queryPrometheus(clientset kubernetes.Interface, endpoint, query string) (*prometheusResponse, error) {
	var body []byte
	var err error

	if strings.HasPrefix(endpoint, "http://") || strings.HasPrefix(endpoint, "https://") {
		body, err = queryPrometheusDirectHTTP(endpoint, query)
	} else {
		body, err = queryPrometheusViaProxy(clientset, endpoint, query)
	}
	if err != nil {
		return nil, err
	}

	var resp prometheusResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parsing Prometheus response: %w", err)
	}

	if resp.Status != "success" {
		return nil, fmt.Errorf("Prometheus query failed with status: %s", resp.Status)
	}

	return &resp, nil
}

func queryPrometheusDirectHTTP(endpoint, query string) ([]byte, error) {
	u := fmt.Sprintf("%s/api/v1/query?query=%s", strings.TrimRight(endpoint, "/"), url.QueryEscape(query))
	resp, err := http.Get(u) //nolint:gosec // user-provided endpoint
	if err != nil {
		return nil, fmt.Errorf("HTTP request to Prometheus: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading Prometheus response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Prometheus returned HTTP %d: %s", resp.StatusCode, string(body))
	}

	return body, nil
}

func queryPrometheusViaProxy(clientset kubernetes.Interface, endpoint, query string) ([]byte, error) {
	// Parse namespace/service:port
	parts := strings.SplitN(endpoint, "/", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid Prometheus endpoint format %q, expected namespace/service:port", endpoint)
	}
	ns := parts[0]
	svcPort := parts[1]

	// Split service:port
	svcParts := strings.SplitN(svcPort, ":", 2)
	if len(svcParts) != 2 {
		return nil, fmt.Errorf("invalid Prometheus endpoint format %q, expected namespace/service:port", endpoint)
	}
	svc := svcParts[0]
	port := svcParts[1]

	body, err := clientset.CoreV1().RESTClient().Get().
		Namespace(ns).
		Resource("services").
		Name(svc+":"+port).
		SubResource("proxy").
		Suffix("api", "v1", "query").
		Param("query", query).
		DoRaw(context.TODO())
	if err != nil {
		return nil, fmt.Errorf("K8s API proxy request to Prometheus: %w", err)
	}

	return body, nil
}

func parseValue(val []interface{}) (float64, error) {
	if len(val) < 2 {
		return 0, fmt.Errorf("unexpected value format")
	}
	s, ok := val[1].(string)
	if !ok {
		return 0, fmt.Errorf("value is not a string: %v", val[1])
	}
	return strconv.ParseFloat(s, 64)
}

func buildPodMetricsList(cpuResp, memResp *prometheusResponse) *v1beta1.PodMetricsList {
	// Key: namespace/pod/container
	type containerUsage struct {
		cpu    *resource.Quantity
		memory *resource.Quantity
	}
	type podKey struct {
		namespace string
		pod       string
	}

	containers := map[string]*containerUsage{}

	for _, r := range cpuResp.Data.Result {
		ns := r.Metric["namespace"]
		pod := r.Metric["pod"]
		container := r.Metric["container"]
		key := ns + "/" + pod + "/" + container

		val, err := parseValue(r.Value)
		if err != nil {
			continue
		}
		milliCores := int64(math.Round(val * 1000))
		q := resource.NewMilliQuantity(milliCores, resource.DecimalSI)

		if _, ok := containers[key]; !ok {
			containers[key] = &containerUsage{}
		}
		containers[key].cpu = q
	}

	for _, r := range memResp.Data.Result {
		ns := r.Metric["namespace"]
		pod := r.Metric["pod"]
		container := r.Metric["container"]
		key := ns + "/" + pod + "/" + container

		val, err := parseValue(r.Value)
		if err != nil {
			continue
		}
		bytes := int64(math.Round(val))
		q := resource.NewQuantity(bytes, resource.BinarySI)

		if _, ok := containers[key]; !ok {
			containers[key] = &containerUsage{}
		}
		containers[key].memory = q
	}

	// Group by pod
	pods := map[podKey][]v1beta1.ContainerMetrics{}
	for key, usage := range containers {
		parts := strings.SplitN(key, "/", 3)
		if len(parts) != 3 {
			continue
		}
		pk := podKey{namespace: parts[0], pod: parts[1]}
		containerName := parts[2]

		cm := v1beta1.ContainerMetrics{
			Name:  containerName,
			Usage: make(corev1.ResourceList),
		}
		if usage.cpu != nil {
			cm.Usage[corev1.ResourceCPU] = *usage.cpu
		}
		if usage.memory != nil {
			cm.Usage[corev1.ResourceMemory] = *usage.memory
		}
		pods[pk] = append(pods[pk], cm)
	}

	pmList := &v1beta1.PodMetricsList{}
	for pk, cms := range pods {
		pmList.Items = append(pmList.Items, v1beta1.PodMetrics{
			ObjectMeta: metav1.ObjectMeta{
				Name:      pk.pod,
				Namespace: pk.namespace,
			},
			Containers: cms,
		})
	}

	return pmList
}

func buildNodeMetricsList(cpuResp, memResp *prometheusResponse) *v1beta1.NodeMetricsList {
	type nodeUsage struct {
		cpu    *resource.Quantity
		memory *resource.Quantity
	}

	nodes := map[string]*nodeUsage{}

	for _, r := range cpuResp.Data.Result {
		node := r.Metric["node"]
		if node == "" {
			continue
		}

		val, err := parseValue(r.Value)
		if err != nil {
			continue
		}
		milliCores := int64(math.Round(val * 1000))
		q := resource.NewMilliQuantity(milliCores, resource.DecimalSI)

		if _, ok := nodes[node]; !ok {
			nodes[node] = &nodeUsage{}
		}
		nodes[node].cpu = q
	}

	for _, r := range memResp.Data.Result {
		node := r.Metric["node"]
		if node == "" {
			continue
		}

		val, err := parseValue(r.Value)
		if err != nil {
			continue
		}
		bytes := int64(math.Round(val))
		q := resource.NewQuantity(bytes, resource.BinarySI)

		if _, ok := nodes[node]; !ok {
			nodes[node] = &nodeUsage{}
		}
		nodes[node].memory = q
	}

	nmList := &v1beta1.NodeMetricsList{}
	for name, usage := range nodes {
		nm := v1beta1.NodeMetrics{
			ObjectMeta: metav1.ObjectMeta{
				Name: name,
			},
			Usage: make(corev1.ResourceList),
		}
		if usage.cpu != nil {
			nm.Usage[corev1.ResourceCPU] = *usage.cpu
		}
		if usage.memory != nil {
			nm.Usage[corev1.ResourceMemory] = *usage.memory
		}
		nmList.Items = append(nmList.Items, nm)
	}

	return nmList
}
