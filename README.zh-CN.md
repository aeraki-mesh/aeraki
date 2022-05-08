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

# Aeraki （[English](./README.md)）

[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://github.com/aeraki-mesh/aeraki/blob/master/LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/aeraki-mesh/aeraki)](https://goreportcard.com/report/github.com/aeraki-mesh/aeraki)
[![CI Tests](https://github.com/aeraki-mesh/aeraki/workflows/ci/badge.svg?branch=master)](https://github.com/aeraki-mesh/aeraki/actions?query=branch%3Amaster+event%3Apush+workflow%3A%22ci%22)
[![E2E Tests](https://github.com/aeraki-mesh/aeraki/workflows/e2e-metaprotocol/badge.svg?branch=master)](https://github.com/aeraki-mesh/aeraki/actions?query=branch%3Amaster+event%3Apush+workflow%3A%22e2e-metaprotocol%22)
[![E2E Tests](https://github.com/aeraki-mesh/aeraki/workflows/e2e-dubbo/badge.svg?branch=master)](https://github.com/aeraki-mesh/aeraki/actions?query=branch%3Amaster+event%3Apush+workflow%3A%22e2e-dubbo%22)
[![E2E Tests](https://github.com/aeraki-mesh/aeraki/workflows/e2e-thrift/badge.svg?branch=master)](https://github.com/aeraki-mesh/aeraki/actions?query=branch%3Amaster+event%3Apush+workflow%3A%22e2e-thrift%22)
[![E2E Tests](https://github.com/aeraki-mesh/aeraki/workflows/e2e-kafka-zookeeper/badge.svg?branch=master)](https://github.com/aeraki-mesh/aeraki/actions?query=branch%3Amaster+event%3Apush+workflow%3A%22e2e-kafka-zookeeper%22)
[![E2E Tests](https://github.com/aeraki-mesh/aeraki/workflows/e2e-redis/badge.svg?branch=master)](https://github.com/aeraki-mesh/aeraki/actions?query=branch%3Amaster+event%3Apush+workflow%3A%22e2e-redis%22)

# 在服务网格中管理任何七层流量!

**Aeraki** [Air-rah-ki] 是希腊语 ”微风“ 的意思。 虽然服务网格已经成为微服务的重要基础设施，但许多（可能不是全部）服务网格的实现主要集中在 HTTP 协议，并将其他协议视为普通的 TCP 流量。Aeraki Mesh 的创建是为了提供一种非侵入性的、高度可扩展的方式来管理服务网格中的任何7层流量。

请注意：Aeraki 只处理服务网中的非 HTTP 的七层流量，而将 HTTP 流量留给其他现有的服务网格项目。(现有的项目已经足够优秀，而不必重新造轮子。)  Aeraki 目前可以与 Istio 集成，并且在未来可能会支持其他服务网格项目。

## 待解决的问题

我们在服务网格中面临的一些挑战：
*  Istio 和其他流行的服务网状结构实现对 HTTP 和 gRPC 协议之外的7层协议的支持非常有限。
* Envoy RDS (Route Discovery Service) 是专为 HTTP 设计的。而其他的协议，如 Dubbo 和 Thrift 等，只能使用监听器在线路由来进行流量管理，而当路由改变时，会破坏现有连接。
* 在服务网格中引入一个专有协议需要花费很多精力。需要编写一个 Envoy 过滤器来处理网络层的流量，以及一个网络层来管理这些 Envoy 代理。

这些障碍使得用户难以，甚至不可能去管理微服务中其他广泛使用的7层协议的流量。例如，在一个微服务应用中，我们可能使用以下协议。

* RPC: HTTP, gRPC, Thrift, Dubbo, Proprietary RPC Protocol …
* 消息队列: Kafka, RabbitMQ …
* 缓存: Redis, Memcached …
* 数据库: MySQL, PostgreSQL, MongoDB …

![ 微服务中常使用的七层协议 ](docs/protocols.png)

如果你已经在迁移到其他服务网格中投入了大量的精力，当然，你希望得到它的最大好处--管理你的微服务中所有协议的流量。

## Aeraki 的解决方案

为了解决这些问题，Aeraki Mesh 提供了一种非侵入性的、可扩展的方式来管理任何服务网中的7层流量。
![ Aeraki ](docs/aeraki-architecture.png)

正如该图所示，Aeraki Mesh 由以下几部分组成。

* Aeraki: [Aeraki](https://github.com/aeraki-mesh/aeraki) 为运维提供了高层次的、用户友好的流量管理规则，将规则转化为 envoy 过滤器配置，并利用 Istio 的`EnvoyFilter`API 将配置推送给 sidecar 代理。 Aeraki 还在网络层中充当了 MetaProtocol 的代理 RDS 服务器。不同于专注于 HTTP 的 Envoy RDS 相反，Aeraki RDS 旨在为所有七层协议提供通用的动态路由能力。
* MetaProtocol Proxy: [MetaProtocol Proxy](https://github.com/aeraki-mesh/meta-protocol-proxy) 为七层协议提供了常见的功能，如负载均衡、熔断、路由、速率限制、故障注入和认证。七层协议可以建立在 MetaProtocol 之上。要在服务网格中加入一个新的协议，唯一需要做的就是实现 [编解码器接口](https://github.com/aeraki-mesh/meta-protocol-proxy/blob/ac788327239bd794e745ce18b382da858ddf3355/src/meta_protocol_proxy/codec/codec.h#L118) 和几行配置。如果有特殊的要求，而内置的功能又不能满足，MetaProtocol Proxy 还存在着一个应用级的过滤器链机制，允许用户编写自己的七层过滤器，将自定义的逻辑加入 MetaProtocol Proxy。

[Dubbo](https://github.com/aeraki-mesh/meta-protocol-proxy/tree/master/src/application_protocols/dubbo) 和 [Thrift](https://github.com/aeraki-mesh/meta-protocol-proxy/tree/master/src/application_protocols/thrift) 已经在 MetaProtocol 的基础上实现了协议支持。我们也将会实现更多的协议。如果用户正在使用一个闭源的专有协议，也可以在你的服务网格中管理它，而只需为它编写一个 MetaProtocol 编解码器。

大多数请求/响应式的无状态协议都可以建立在 MetaProtocol Proxy 之上。但是，有些协议的路由策略过于 "特殊"，无法在 MetaProtocol 中规范化。例如，Redis 代理使用 slot number 将客户端查询映射到特定的Redis服务器节点，slot number 是由请求中的密钥计算出来的。 只要在 Envoy 代理方有一个可用的 Envoy 过滤器，Aeraki 仍然可以管理这些协议。目前，对于这一类的协议，Aeraki 支持 [Redis](https://github.com/aeraki-mesh/aeraki/blob/master/docs/zh/redis.md) 和 Kafka。
## 参考文档
* [应用协议的实现](docs/metaprotocol.md)
* [Dubbo (中文) ](https://github.com/aeraki-mesh/dubbo2istio#readme)
* [Redis (中文) ](docs/zh/redis.md)

## 支持的协议:
Aeraki 可以在一个服务网格中管理以下协议：
* Dubbo (Envoy 原生过滤器）
* Thrift (Envoy 原生过滤器)
* Kafka (Envoy 原生过滤器)
* Redis (Envoy 原生过滤器)
* MetaProtocol-Dubbo
* MetaProtocol-Thfirt
* MetaProtocol-QQ music(QQ 音乐)
* MetaProtcool-Yangshiping(央视频)
* MetaProtocol-Private protocol: Have a private protocol? No problem, any layer-7 protocols built on top of the [MetaProtocol](https://github.com/aeraki-mesh/meta-protocol-proxy) can be managed by Aeraki

支持的特性:
  * 流量管理
    * [x] 请求级负载均衡/地域感知负载均衡（支持一致哈希算法/会话保持）
    * [x] 熔断
    * [x] 灵活的路由匹配条件（任何从7层数据包中提取的属性，都可作为匹配条件）
    * [x] 通过 Aeraki MetaRDS 实现动态路由更新
    * [x] 基于版本的路由
    * [x] 流量分流
    * [x] 本地流量限制
    * [x] 全局流量限制
    * [x] 消息修改
    * [ ] 流量镜像
    * [ ] 请求转换
  * 可观测性
    * [x] 请求级指标 (请求级延迟、计数、错误等)
    * [ ] 分布式追踪
  * 安全性
    * [x] 基于接口/方法的对等授权
    * [ ] 授权请求

> 请注意: 建立在 MetaProtocol 之上的协议支持 Aeraki Mesh 的上述所有功能，Envoy 本地过滤器只支持部分上述功能，这取决于本地过滤器。

## 样例

https://www.aeraki.net/docs/v1.0/quickstart/

## 安装

https://www.aeraki.net/docs/v1.0/install/

## 构建

### 依赖
* Golang Version >= 1.16, and related golang tools installed like `goimports`, `gofmt`, etc.
* Docker and Docker-Compose installed

### 构建 Aeraki 二进制文件

```bash
# build aeraki binary on linux
make build

# build aeraki binary on darwin
make build-mac
```

### 构建 Aeraki Docker 镜像

```bash
# build aeraki docker image with the default latest tag
make docker-build

# build aeraki docker image with xxx tag
make docker-build tag=xxx

# build aeraki e2e docker image
make docker-build-e2e
```

## 讲座

* Istio meetup China(中文): [全栈服务网格 - Aeraki 助你在 Istio 服务网格中管理任何七层流量](https://www.youtube.com/watch?v=Bq5T3OR3iTM) 
* IstioCon 2021: [How to Manage Any Layer-7 Traffic in an Istio Service Mesh?](https://www.youtube.com/watch?v=sBS4utF68d8)

## 谁在使用 Aeraki?

真诚地感谢大家选择、贡献和使用 Aeraki。我们创建这个 issue 是为了收集使用案例，以便我们能够推动 Aeraki 社区向正确的方向发展，以更好地服务于真实的使用场景。我们鼓励您在这个问题上提交评论，包括您的使用方式：https://github.com/aeraki-mesh/aeraki/issues/105

## 联系我们
* Mail: 如果你有兴趣为这个项目作出贡献，请联系 zhaohuabing@gmail.com
* Wechat Group: 请联系微信ID：zhao_huabing，来加入 Aeraki 微信群聊
* Slack: 加入 [Aeraki slack 频道](https://istio.slack.com/archives/C02UB8YJ600)

## Landscapes

<p align="center">
<img src="https://landscape.cncf.io/images/left-logo.svg" width="150"/>&nbsp;&nbsp;<img src="https://landscape.cncf.io/images/right-logo.svg" width="200"/>
<br/><br/>
Aeraki Mesh enriches the <a href="https://landscape.cncf.io/?selected=aeraki-mesh">CNCF CLOUD NATIVE Landscape.</a>
</p>
