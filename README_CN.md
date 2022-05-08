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

# Aeraki

[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://github.com/aeraki-mesh/aeraki/blob/master/LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/aeraki-mesh/aeraki)](https://goreportcard.com/report/github.com/aeraki-mesh/aeraki)
[![CI Tests](https://github.com/aeraki-mesh/aeraki/workflows/ci/badge.svg?branch=master)](https://github.com/aeraki-mesh/aeraki/actions?query=branch%3Amaster+event%3Apush+workflow%3A%22ci%22)
[![E2E Tests](https://github.com/aeraki-mesh/aeraki/workflows/e2e-metaprotocol/badge.svg?branch=master)](https://github.com/aeraki-mesh/aeraki/actions?query=branch%3Amaster+event%3Apush+workflow%3A%22e2e-metaprotocol%22)
[![E2E Tests](https://github.com/aeraki-mesh/aeraki/workflows/e2e-dubbo/badge.svg?branch=master)](https://github.com/aeraki-mesh/aeraki/actions?query=branch%3Amaster+event%3Apush+workflow%3A%22e2e-dubbo%22)
[![E2E Tests](https://github.com/aeraki-mesh/aeraki/workflows/e2e-thrift/badge.svg?branch=master)](https://github.com/aeraki-mesh/aeraki/actions?query=branch%3Amaster+event%3Apush+workflow%3A%22e2e-thrift%22)
[![E2E Tests](https://github.com/aeraki-mesh/aeraki/workflows/e2e-kafka-zookeeper/badge.svg?branch=master)](https://github.com/aeraki-mesh/aeraki/actions?query=branch%3Amaster+event%3Apush+workflow%3A%22e2e-kafka-zookeeper%22)
[![E2E Tests](https://github.com/aeraki-mesh/aeraki/workflows/e2e-redis/badge.svg?branch=master)](https://github.com/aeraki-mesh/aeraki/actions?query=branch%3Amaster+event%3Apush+workflow%3A%22e2e-redis%22)

# 在服务网格中管理 **任何** 7层流量！

**Aeraki** [Air-rah-ki]取自希腊语 "微风"。虽然服务网格技术已成为微服务的重要基础设施，但许多（如果不是全部）服务网格实现主要关注HTTP协议，并将其他协议视为普通TCP流量。Aeraki Mesh的创建是为了提供一种非侵入、高扩展的方式，来管理服务网格中任何7层流量。

注意：Aeraki在服务网格中只处理非HTTP的7层流量，而将HTTP流量留给其他现有的服务网格项目。(因为他们已经做得够好了，我们不想重新发明轮子! ) Aeraki目前可以与Istio集成，未来可能会支持其他服务网格项目。

## 需要解决的问题

我们在服务网格中面临着一些挑战：
* Istio和其他流行的服务网格实现对HTTP和gRPC之外的七层协议支持非常有限。
* Envoy RDS(Route Discovery Service)只为HTTP设计。其他协议如Dubbo和Thrift，只能使用监听器内联路由的方式进行流量管理，当路由发生改变时，会破坏现有的连接。
* 在服务网格中引入一个专有协议是要花费很多精力的。你需要写一个Envoy过滤器来处理数据平面的流量，以及一个控制平面来管理这些Envoy代理。

这些障碍使得用户很难，甚至不可能管理微服务中其他广泛使用的7层协议流量。例如，在一个微服务应用中，我们可能使用以下协议：
* RPC: HTTP, gRPC, Thrift, Dubbo, 专有RPC协议 …
* 消息: Kafka, RabbitMQ …
* 缓存: Redis, Memcached …
* 数据库: MySQL, PostgreSQL, MongoDB …

![ Common Layer 7 Protocols Used in Microservices ](docs/protocols.png)

如果你已经为迁移到服务网格中投入了大量的精力，当然，你想从它身上得到最大的好处--管理你微服务中所有协议的流量。

## Aeraki的解决方案

为了解决这些问题，Aeraki Mesh提供了一种非侵入、可扩展的方式来管理服务网格中的任何7层流量。
![ Aeraki ](docs/aeraki-architecture.png)

如上图所示，Aeraki Mesh由以下组件组成：

* Aeraki: [Aeraki](https://github.com/aeraki-mesh/aeraki)为运维提供高层次、用户友好的流量管理规则，将规则转化为envoy过滤器配置，并利用Istio的`EnvoyFilter` API将配置推送到边车代理。Aeraki还充当数据平面中MetaProtocol代理的RDS服务器。与专注于HTTP的Envoy RDS相反，Aeraki RDS目的是为所有第七层协议提供通用的动态路由能力。
* MetaProtocol Proxy: [MetaProtocol Proxy](https://github.com/aeraki-mesh/meta-protocol-proxy)为七层协议提供通用功能，如负载均衡、熔断、路由、限流、故障注入和认证。七层协议可以建立在MetaProtocol之上。想在服务网格中加入一个新的协议，唯一需要做的就是实现[编接码器接口](https://github.com/aeraki-mesh/meta-protocol-proxy/blob/ac788327239bd794e745ce18b382da858ddf3355/src/meta_protocol_proxy/codec/codec.h#L118)和几行配置。如果你有特殊的要求，而内置的功能又不能满足，MetaProtocol Proxy还有一个应用级的过滤链机制，允许用户编写他们自己的七层过滤器，将自定义的逻辑加入MetaProtocol Proxy。

[Dubbo](https://github.com/aeraki-mesh/meta-protocol-proxy/tree/master/src/application_protocols/dubbo)和[Thrift](https://github.com/aeraki-mesh/meta-protocol-proxy/tree/master/src/application_protocols/thrift)已经基于MetaProtocol实现协议支持。更多的协议支持正在进行中。如果你正在使用一个闭源的专有协议，只需为它编写一个MetaProtocol编解码器，你也可以在你的服务网格中轻松管理它。

大多数请求/响应样式的无状态协议都可以建立在MetaProtocol Proxy之上。然而，有些协议的路由策略过于 "特殊"，无法在MetaProtocol中实现规范化。例如，Redis代理使用槽号将客户端查询请求映射到特定的Redis服务器节点，槽号是由请求中的密钥计算出来的。只要在Envoy代理端有一个可用的Envoy过滤器，Aeraki仍然可以管理这些协议。目前，对于这一类的协议，Aeraki已支持了[Redis](https://github.com/aeraki-mesh/aeraki/blob/master/docs/zh/redis.md)和Kafka。

## 参考资料
* [如何实现一个应用协议](docs/metaprotocol.md)
* [Dubbo (中文) ](https://github.com/aeraki-mesh/dubbo2istio#readme)
* [Redis (中文) ](docs/zh/redis.md)

## 已支持的协议列表：
Aeraki可以在服务网格中管理以下协议：
* Dubbo (Envoy原生过滤器）
* Thrift (Envoy原生过滤器)
* Kafka (Envoy原生过滤器)
* Redis (Envoy原生过滤器)
* MetaProtocol-Dubbo
* MetaProtocol-Thfirt
* MetaProtocol-QQ music(QQ 音乐)
* MetaProtcool-Yangshiping(央视频)
* MetaProtocol-Private protocol: 有一个私有协议？没问题，任何建立在[MetaProtocol](https://github.com/aeraki-mesh/meta-protocol-proxy)之上的七层协议都可以由Aeraki管理。

已支持的功能特性：
  * 流量管理
    * [x] 请求级别负载均衡/地域感知负载均衡（支持一致哈希/会话保持）
    * [x] 熔断
    * [x] 灵活的路由匹配条件（从7层数据包中提取的任何属性，都可作为匹配条件）
    * [x] 通过Aeraki MetaRDS实现动态路由更新
    * [x] 基于版本的路由
    * [x] 流量分流
    * [x] 本地限流
    * [x] 全局限流
    * [x] 消息修改
    * [ ] 流量镜像
    * [ ] 请求转换
  * 可观测性
    * [x] 请求级别的指标（请求延迟、计数、错误等）
    * [ ] 分布式追踪
  * 安全性
    * [x] 基于接口/方法的对等授权
    * [ ] 请求授权

注意：建立在MetaProtocol之上的协议支持Aeraki Mesh的上述所有功能，而Envoy原生过滤器只支持上述部分功能，这取决于原生过滤器的自身能力。

## Demo

https://www.aeraki.net/docs/v1.0/quickstart/

## 安装

https://www.aeraki.net/docs/v1.0/install/

## 构建

### 前置条件:
* 安装Golang，版本 >= 1.16, 并安装相关的golang工具，如`goimports`，`gofmt`等
* 安装Docker 和 Docker-Compose

### 构建 Aeraki 二进制文件

```bash
# build aeraki binary on linux
make build

# build aeraki binary on darwin
make build-mac
```

### 构建 Aeraki 容器镜像

```bash
# build aeraki docker image with the default latest tag
make docker-build

# build aeraki docker image with xxx tag
make docker-build tag=xxx

# build aeraki e2e docker image
make docker-build-e2e
```

## 演讲

* Istio meetup China(中文): [全栈服务网格 - Aeraki 助你在 Istio 服务网格中管理任何七层流量](https://www.youtube.com/watch?v=Bq5T3OR3iTM) 
* IstioCon 2021: [How to Manage Any Layer-7 Traffic in an Istio Service Mesh?](https://www.youtube.com/watch?v=sBS4utF68d8)

## 谁正在使用Aeraki？

真诚地感谢大家选择、贡献和使用Aeraki。我们创建这个issue是为了收集使用案例，以便我们能够推动Aeraki社区向正确的方向发展，并更好地服务于真实应用场景。我们鼓励你在这个issue上提交评论，包括你的使用案例：https://github.com/aeraki-mesh/aeraki/issues/105 。

## 联系我们
* 邮件：如果你有兴趣为这个项目做贡献，请联系zhaohuabing@gmail.com
* 微信群：请联系微信ID：zhao_huabing，加入Aeraki微信群
* Slack: 加入[Aeraki slack频道](https://istio.slack.com/archives/C02UB8YJ600)

## 入选云原生全景图

<p align="center">
<img src="https://landscape.cncf.io/images/left-logo.svg" width="150"/>&nbsp;&nbsp;<img src="https://landscape.cncf.io/images/right-logo.svg" width="200"/>
<br/><br/>
Aeraki Mesh 已入选 <a href="https://landscape.cncf.io/?selected=aeraki-mesh">CNCF基金会云原生全景图</a>
</p>
