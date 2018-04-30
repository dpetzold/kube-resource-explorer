Resource lister
================

List the resource allocation to pods in a cluster. Inspired by:

https://github.com/kubernetes/kubernetes/issues/17512

## Usage

## Command Line Options                                                                                                
* `-namespace` - Limit the query to the specified namespace (defaults to all)
* `-field` - Field to sort by. Possible values are: "Name, Namespace, CpuReq, CpuLimit, FractionCpuReq, FractionCpuLimit, MemReq, MemLimit, FractionMemoryReq, FractionMemoryLimit"
* `-reverse` - Reserve the sort order

## Build

```
$ make build
```

## Example output

```
$ ./resource-lister -namespace kube-system -reverse -field MemReq
Namespace    Name                                               CpuReq  CpuReq%  CpuLimit  CpuLimit%  MemReq    MemReq%  MemLimit  MemLimit%
---------    ----                                               ------  -------  --------  ---------  ------    -------  --------  ---------
kube-system  event-exporter-v0.1.7-5c4d9556cf-kf4tf             0       0%       0         0%         0         0%       0         0%
kube-system  kube-proxy-gke-project-default-pool-175a4a05-mshh  100m    10%      0         0%         0         0%       0         0%
kube-system  kube-proxy-gke-project-default-pool-175a4a05-bv59  100m    10%      0         0%         0         0%       0         0%
kube-system  kube-proxy-gke-project-default-pool-175a4a05-ntfw  100m    10%      0         0%         0         0%       0         0%
kube-system  kube-dns-autoscaler-244676396-xzgs4                20m     2%       0         0%         10Mi      0%       0         0%
kube-system  l7-default-backend-1044750973-kqh98                10m     1%       10m       1%         20Mi      0%       20Mi      0%
kube-system  kubernetes-dashboard-768854d6dc-jh292              100m    10%      100m      10%        100Mi     3%       300Mi     11%
kube-system  kube-dns-323615064-8nxfl                           260m    27%      0         0%         110Mi     4%       170Mi     6%
kube-system  fluentd-gcp-v2.0.9-4qkwk                           100m    10%      0         0%         200Mi     7%       300Mi     11%
kube-system  fluentd-gcp-v2.0.9-jmtpw                           100m    10%      0         0%         200Mi     7%       300Mi     11%
kube-system  fluentd-gcp-v2.0.9-tw9vk                           100m    10%      0         0%         200Mi     7%       300Mi     11%
kube-system  heapster-v1.4.3-74b5bd94bb-fz8hd                   138m    14%      138m      14%        301856Ki  11%      301856Ki  11%
```
