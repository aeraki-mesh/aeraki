<!--
# Copyright Aeraki Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
-->

# 快速安装

本文章主要讲解如何使用`helm` 来安装 `aeraki`,  使Istio 支持更多的第 7 层协议。

## 前提条件

1. [安装](https://helm.sh/docs/intro/install) 3.1.1 以上版本的 Helm 客户端。
2. [安装](https://istio.io/latest/docs/setup/install/) istio(1.10.0)以上。


## 安装步骤

1. clone `aeraki`代码: 

```bash
# 检查 helm版本, 我本地安装的是(v3.6.3)版本的helm
$ helm version
version.BuildInfo{Version:"v3.6.3", GitCommit:"d506314abfb5d21419df8c7e7e68012379db2354", GitTreeState:"dirty", GoVersion:"go1.16.5"}

# 下载代码
$ git clone https://github.com/aeraki-mesh/aeraki

$ cd aeraki/manifests/charts

```

2. istio-system为aeraki组件创建命名空间:

```bash
$ helm install aeraki ./aeraki -n istio-system

WARNING: Kubernetes configuration file is group-readable. This is insecure. Location: /root/.kube/config
WARNING: Kubernetes configuration file is world-readable. This is insecure. Location: /root/.kube/config
NAME: aeraki
LAST DEPLOYED: Thu Jul 29 16:25:22 2021
NAMESPACE: istio-system
STATUS: deployed
REVISION: 1
TEST SUITE: None
```

## 验证安装

查询所有istio-system命名空间下pod的状态，所有的STATUS都为Running:

```bash
$ kubectl get po -n istio-system

NAME                                   READY   STATUS    RESTARTS   AGE
aeraki-68f87454dc-lf6mh                1/1     Running   0          68s
istio-ingressgateway-b6b8445c6-xpsj7   1/1     Running   0          43m
istiod-bcddf5d98-mq9ln                 1/1     Running   0          43m
```

## 也可以看看

[使用 aeraki 来对dubbo协议进行测试](https://github.com/aeraki-mesh/dubbo2istio)

Dubbo2istio 将 Dubbo 服务注册表中的 Dubbo 服务自动同步到 Istio 服务网格中，目前已经支持 ZooKeeper，Nacos 和 Etcd。
