# kube-capacity

[![Go Report Card](https://goreportcard.com/badge/github.com/robscott/kube-capacity?v=1)](https://goreportcard.com/report/github.com/robscott/kube-capacity) [![CircleCI](https://circleci.com/gh/robscott/kube-capacity.svg?style=svg)](https://circleci.com/gh/robscott/kube-capacity)

This is a simple CLI that provides an overview of the resource requests, limits, and utilization in a Kubernetes cluster. It attempts to combine the best parts of the output from `kubectl top` and `kubectl describe` into an easy to use CLI focused on cluster resources.

## Installation
Go binaries are automatically built with each release by [GoReleaser](https://github.com/goreleaser/goreleaser). These can be accessed on the GitHub [releases page](https://github.com/robscott/kube-capacity/releases) for this project.

### Homebrew
This project can be installed with [Homebrew](https://brew.sh/):
```
brew tap robscott/tap
brew install robscott/tap/kube-capacity
```

### Krew
This project can be installed with [Krew](https://github.com/GoogleContainerTools/krew):
```
kubectl krew install resource-capacity
```

## Usage
By default, kube-capacity will output a list of nodes with the total CPU and Memory resource requests and limits for all the pods running on them. For clusters with more than one node, the first line will also include cluster wide totals. That output will look something like this:

```
kube-capacity

NODE              CPU REQUESTS    CPU LIMITS    MEMORY REQUESTS    MEMORY LIMITS
*                 560m (28%)      130m (7%)     572Mi (9%)         770Mi (13%)
example-node-1    220m (22%)      10m (1%)      192Mi (6%)         360Mi (12%)
example-node-2    340m (34%)      120m (12%)    380Mi (13%)        410Mi (14%)
```

### Including Pods
For more detailed output, kube-capacity can include pods in the output. When `-p` or `--pods` are passed to kube-capacity, it will include pod specific output that looks like this:

```
kube-capacity --pods

NODE              NAMESPACE     POD                   CPU REQUESTS    CPU LIMITS    MEMORY REQUESTS    MEMORY LIMITS
*                 *             *                     560m (28%)      780m (38%)    572Mi (9%)         770Mi (13%)

example-node-1    *             *                     220m (22%)      320m (32%)    192Mi (6%)         360Mi (12%)
example-node-1    kube-system   metrics-server-lwc6z  100m (10%)      200m (20%)    100Mi (3%)         200Mi (7%)
example-node-1    kube-system   coredns-7b5bcb98f8    120m (12%)      120m (12%)    92Mi (3%)          160Mi (5%)

example-node-2    *             *                     340m (34%)      460m (46%)    380Mi (13%)        410Mi (14%)
example-node-2    kube-system   kube-proxy-3ki7       200m (20%)      280m (28%)    210Mi (7%)         210Mi (7%)
example-node-2    tiller        tiller-deploy         140m (14%)      180m (18%)    170Mi (5%)         200Mi (7%)
```

### Including Utilization
To help understand how resource utilization compares to configured requests and limits, kube-capacity can include utilization metrics in the output. It's important to note that this output relies on [metrics-server](https://github.com/kubernetes-incubator/metrics-server) functioning correctly in your cluster. When `-u` or `--util` are passed to kube-capacity, it will include resource utilization information that looks like this:

```
kube-capacity --util

NODE              CPU REQUESTS    CPU LIMITS    CPU UTIL    MEMORY REQUESTS    MEMORY LIMITS   MEMORY UTIL
*                 560m (28%)      130m (7%)     40m (2%)    572Mi (9%)         770Mi (13%)     470Mi (8%)
example-node-1    220m (22%)      10m (1%)      10m (1%)    192Mi (6%)         360Mi (12%)     210Mi (7%)
example-node-2    340m (34%)      120m (12%)    30m (3%)    380Mi (13%)        410Mi (14%)     260Mi (9%)
```

### Displaying Available Resources
To more clearly see the total available resources on the node it is possible to pass the `--available` option
to kube-capacity, which will give output in the following format

```
kube-capacity --available

NODE              CPU REQUESTS    CPU LIMITS    MEMORY REQUESTS    MEMORY LIMITS
*                 560/2000m       130/2000m     572/5923Mi         770/5923Mi 
example-node-1    220/1000m       10/1000m      192/3200Mi         360/3200Mi 
example-node-2    340/1000m       120/1000m     380/2923Mi         410/2923Mi
```

### Including Pods and Utilization
For more detailed output, kube-capacity can include both pods and resource utilization in the output. When `--util` and `--pods` are passed to kube-capacity, it will result in a wide output that looks like this:

```
kube-capacity --pods --util

NODE              NAMESPACE     POD                   CPU REQUESTS    CPU LIMITS   CPU UTIL     MEMORY REQUESTS    MEMORY LIMITS   MEMORY UTIL
*                 *             *                     560m (28%)      780m (38%)   340m (17%)   572Mi (9%)         770Mi (13%)     470Mi (8%)

example-node-1    *             *                     220m (22%)      320m (32%)   160m (16%)   192Mi (6%)         360Mi (12%)     210Mi (7%)
example-node-1    kube-system   metrics-server-lwc6z  100m (10%)      200m (20%)   70m (7%)     100Mi (3%)         200Mi (7%)      120Mi (4%)
example-node-1    kube-system   coredns-7b5bcb98f8    120m (12%)      120m (12%)   90m (9%)     92Mi (3%)          160Mi (5%)      90Mi (3%)

example-node-2    *             *                     340m (34%)      460m (46%)   180m (18%)   380Mi (13%)        410Mi (14%)     260Mi (9%)
example-node-2    kube-system   kube-proxy-3ki7       200m (20%)      280m (28%)   110m (11%)   210Mi (7%)         210Mi (7%)      120Mi (4%)
example-node-2    tiller        tiller-deploy         140m (14%)      180m (18%)   70m (7%)     170Mi (6%)         200Mi (7%)      140Mi (5%)
```

It's worth noting that utilization numbers from pods will likely not add up to the total node utilization numbers. Unlike request and limit numbers where node and cluster level numbers represent a sum of pod values, node metrics come directly from metrics-server and will likely include other forms of resource utilization.

### Sorting
To highlight the nodes, pods, and containers with the highest metrics, you can sort by a variety of columns:

```
kube-capacity --util --sort cpu.util

NODE              CPU REQUESTS    CPU LIMITS    CPU UTIL    MEMORY REQUESTS    MEMORY LIMITS   MEMORY UTIL
*                 560m (28%)      130m (7%)     40m (2%)    572Mi (9%)         770Mi (13%)     470Mi (8%)
example-node-2    340m (34%)      120m (12%)    30m (3%)    380Mi (13%)        410Mi (14%)     260Mi (9%)
example-node-1    220m (22%)      10m (1%)      10m (1%)    192Mi (6%)         360Mi (12%)     210Mi (7%)
```

> **Note** Starting in v0.7.4 you can append `.percentage` to sort by percentage. For
example, `kube-capacity --util --sort cpu.util.percentage`.

### Displaying Pod Count
To display the pod count of each node and the whole cluster, you can pass **--pod-count** argument:
```shell
$ kube-capacity --pod-count

NODE           CPU REQUESTS   CPU LIMITS   MEMORY REQUESTS   MEMORY LIMITS   POD COUNT
*              950m (2%)      200m (0%)    284Mi (0%)        284Mi (0%)      10/220
minikube       850m (5%)      100m (0%)    231Mi (1%)        231Mi (1%)      8/110
minikube-m02   100m (0%)      100m (0%)    53Mi (0%)         53Mi (0%)       2/110
```

### Filtering By Labels
For more advanced usage, kube-capacity also supports filtering by pod, namespace, and/or node labels. The following examples show how to use these filters:

```
kube-capacity --pod-labels app=nginx
kube-capacity --namespace default
kube-capacity --namespace-labels team=api
kube-capacity --node-labels kubernetes.io/role=node
```

### Filtering By Node Taints
Kube-capacity supports advanced filtering by taints. Users can filter in and filter out taints within the same expression. The following examples show how to use node taint filters:

```
kube-capacity --node-taints special=true:NoSchedule 
kube-capacity --node-taints special:NoSchedule 
```
These will return only special nodes.
```
kube-capacity --node-taints special=true:NoSchedule-
kube-capacity --node-taints special:NoSchedule-
```
These will filter out special nodes and return only unspecial nodes.
```
kube-capacity --node-taints special=true:NoSchedule,old-hardware:NoSchedule-
```
This will return special nodes that are not tainted with `old-hardware:NoSchedule`. In other words, display the special nodes but don't display the ones that are running on old hardware.
```
kube-capacity --no-taint
```
This will filter out all nodes with taints. 

### JSON and YAML Output
By default, kube-capacity will provide output in a table format. To view this data in JSON or YAML format, the output flag can be used. Here are some sample commands:
```
kube-capacity --pods --output json
kube-capacity --pods --containers --util --output yaml
```

### CSV and TSV Output
If you would like the data in a comma or tab separated file to make importing the data into a spreadsheet easier the output flag has options for those as well. Here are some sample commands:
```
kube-capacity --pods --output csv
kube-capacity --pods --containers --util --output tsv
```
>Note: the `--available` flag is ignored with these two choices as the values can be derived within a spreadsheet

## Flags Supported
```
      --as string                 user to impersonate command with
      --as-group string           group to impersonate command with
  -c, --containers                includes containers in output
      --context string            context to use for Kubernetes config
  -h, --help                      help for kube-capacity
      --kubeconfig string         kubeconfig file to use for Kubernetes config
  -n, --namespace string          only include pods from this namespace
      --namespace-labels string   labels to filter namespaces with
      --no-taint                  exclude nodes with taints
      --node-labels string        labels to filter nodes with
  -o, --output string             output format for information
                                    (supports: [table json yaml csv tsv])
                                    (default "table")
  -a, --available                 includes quantity available instead of percentage used (ignored with csv or tsv output types)
  -t, --node-taints               taints to filter nodes with
  -l, --pod-labels string         labels to filter pods with
  -p, --pods                      includes pods in output
      --sort string               attribute to sort results by (supports:
                                    [cpu.util cpu.request cpu.limit mem.util mem.request mem.limit cpu.util.percentage
                                    cpu.request.percentage cpu.limit.percentage mem.util.percentage mem.request.percentage
                                    mem.limit.percentage name])
                                    (default "name")
  -u, --util                      includes resource utilization in output
      --pod-count                 includes pod counts for each of the nodes and the whole cluster
```

## Prerequisites
Any commands requesting cluster utilization are dependent on [metrics-server](https://github.com/kubernetes-incubator/metrics-server) running on your cluster. If it's not already installed, you can install it with the official [helm chart](https://github.com/helm/charts/tree/master/stable/metrics-server).

## Similar Projects
There are already some great projects out there that have similar goals.

- [kube-resource-report](https://github.com/hjacobs/kube-resource-report): generates HTML/CSS report for resource requests and limits across multiple clusters.
- [kubetop](https://github.com/LeastAuthority/kubetop): a CLI similar to top for Kubernetes, focused on resource utilization (not requests and limits).

## Contributors

Although this project was originally developed by [robscott](https://github.com/robscott), there have been some great contributions from others:

- [endzyme](https://github.com/endzyme)
- [justinbarrick](https://github.com/justinbarrick)
- [Padarn](https://github.com/Padarn)
- [nickatsegment](https://github.com/nickatsegment)
- [fnickels](https://github.com/fnickels)
- [isaacnboyd](https://github.com/isaacnboyd)

## License
Apache License 2.0
