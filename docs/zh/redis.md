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

# Redis 流量管理

Aeraki 的 Redis 流量管理规则可以让您十分容易的控制应用访问 Redis 服务的流量。

此项功能基于 Envoy 的 [RedisProxy 能力](https://www.envoyproxy.io/docs/envoy/latest/intro/arch_overview/other_protocols/redis#arch-overview-redis) ，网格内的 Redis 流量将经由 Envoy 代理，这让控制网格内的流量变得异常简单，而且不需要对服务做任何的更改。

类比于 istio 原生的 VirtualService 和 DestinationRule ，我们在 Aeraki 上扩展了 RedisService 与 RedisDestination 。

通过使用 **RedisService** 与 **RedisDestination** ，您可以轻松的抽象应用所需访问的 Redis 集群/实例。

比如您在测试集群使用一个小型的单实例的 Redis ，而在生产环境使用一个复杂而稳定的多副本实例的集群模式的Redis，通过 Aeraki ，您将无需修改您应用的代码/配置，从而尽可能的保证了开发，预发布，线上环境相同（[Dev/prod parity](https://12factor.net/dev-prod-parity)）。

## RedisService

RedisService 让您配置如何在网格内将Redis请求路由到实际的 Redis 服务。每个RedisService 包含一组基于Key前缀识别的路由规则，Aeraki 会将每个给定的请求匹配到指定的目的地址上（**RedisDestination**）。

一个经典的使用场景是，您可以轻松的在应用的运行过程中，仅对热点 key 的部分进行在线的容量调整，无需修改程序。抑或是，通过 key 前缀路由，在不同的类型的类 Redis 数据库（比如 pika/codis/twemproxy...）之间做 a/b 测试，以验证真实业务场景下对异构数据库之间的性能差异。

### RedisService 示例

下面的 RedisService 根据请求 key 的前缀将流量路由到不同的 Redis 上。

```yaml
apiVersion: redis.aeraki.io/v1alpha1
kind: RedisService
metadata:
  name: app-cache
spec:
  host:
    - app-cache.redis.svc.cluster.local
  redis:
    - match:
        key:
          prefix: user
      route:
        host: app-cache-user.redis.svc.cluster.local
    - route:
        host: app-cache.redis.svc.cluster.local
```

#### Host 字段

类似于 VirtualService 的 Host 字段，这是客户端向 Redis 服务发送请求时使用的一个或多个地址。

*请注意：host 字段目前只能是 Kubernetes 集群内服务 DNS 全称，即格式为 \${svc}.\${ns}.svc.cluster.local 。*

#### Redis 字段

Redis 字段包含了 RedisService 的所有路由规则。每个路由规则必须包含请求要路由到的目的地。在每个 RedisService 中，您只能最多包含一个无匹配条件的路由规则作为服务的默认目的地，即默认路由。

在上述示例中，第一个路由规则包含一个规则，描述了 所有 key 前缀为 user 的流量，将会被路由到 app-cache-user.redis.svc.cluster.local 服务。

其中，**route 字段** 表明了路由规则的目的地，而其中的 **host 字段** 指明了具体的服务主机名。

*请注意：host 字段目前只能是 Kubernetes 集群内服务 DNS 全称，即格式为 \${svc}.\${ns}.svc.cluster.local 。*

除了默认路由外，其余路由将按照定义顺序依次匹配，（待定，将来可能修改为按照长度排序）。

除此之外，您还可以在路由上执行其他的操作，比如*镜像流量*。

### 故障注入

RedisService 允许您为 Redis 服务注入错误。（关于故障注入，您可以参考[istio 的故障注入](https://istio.io/latest/docs/concepts/traffic-management/#fault-injection)）

当前，通过 RedisService 我们支持配置两种类型的故障，分别是：

- 延迟(DELAY)：延迟是时间故障。它们模拟增加的网络延迟或一个超过负载的上游 Redis 服务。
- 错误(ERROR)：错误是请求错误。它们模拟异常情况下的 Redis 服务。

例如，以下配置会使百分之一的 GET 命令直接返回错误。

```yaml
apiVersion: redis.aeraki.io/v1alpha1
kind: RedisService
metadata:
  name: app-cache
spec:
  host:
    - app-cache.redis.svc.cluster.local
  redis:
    - route:
        host: app-cache.redis.svc.cluster.local
  faults:
    - type: ERROR
      percentage: 
        value: 1
      commands:
        - GET
```

## RedisDestination

RedisDestination 是 Aeraki Redis 流量管理的关键部分。您可以用 RedisService 来指定如何将流量路由到给定目标地址上，然后使用 RedisDestination 来配置该目标的流量。

通过配置 RedisDestination 您可以告知 Aeraki 您的 Redis 集群类别，从而让 Aeraki 协助您的应用程序屏蔽掉 [Cluster 模式](https://redis.io/topics/cluster-spec) 与普通模式的区别。

当您的应用使用 Cluster 模式的 Redis 集群时，您的服务可以使用任何语言实现的非集群模式客户端链接到 Redis 集群，就像单实例 Redis 一样。Sidecar 将自动同步您 Redis 集群的拓扑信息。

使用 RedisDestination 另一个好处是：您可以让 Aeraki 帮助您自动的完成 Redis 的认证，从而避免通过代码或应用配置泄漏出资源的密钥。此时，您的服务就像是访问一个无鉴权的 Redis。

此外，您还可以配置访问 Redis 集群的链接池信息。

### RedisDestination 示例

在下面这个示例中，我们配置了一个自动认证的 **Cluster 模式**的 RedisDestination 。

```yaml
---
apiVersion: redis.aeraki.io/v1alpha1
kind: RedisDestination
metadata:
  name: app-cache
spec:
  host: app-cache.redis.svc.cluster.local
  trafficPolicy:
    connectionPool:
      redis:
        mode: CLUSTER
        auth:
          secret:
            name: app-cache-token
```

#### Mode

RedisDestination 支持两种模式，分别是 *CLUSTER* 和 *PROXY*。

当配置为*CLUSTER*时，Aeraki 将认为 Redis 此时使用了 Redis Cluster 模式，将定期主动同步集群的拓扑信息，并自动将流量按拓扑规则进行路由。

当配置为*PROXY*时，Aeraki 仅做简单的代理。这适用于单实例 Redis 或类似 [Tewmproxy](https://github.com/twitter/twemproxy) 等代理模式下工作的 Redis 集群。

#### Auth

通过配置 Auth 字段，应用在访问 Redis 时，Aeraki 会自动为其完成鉴权。

RedisDestination 支持两种密钥获取的途径：secret 与 plain。

secret 代表所需要的用户名、密码信息存在于 [secret](https://kubernetes.io/docs/concepts/configuration/secret/) 配置中。

默认情况下，RedisDestination 使用所引用 secret 中定义的 `username` 的值作为 auth 使用的用户名，`password` 或 `token` 作为 auth 时使用的密码。

通过配置 `passwordField` 和 `usernameField` 您可以指定做为auth时使用的密码和用户名的 secret 配置的字段。

例如：

```yaml
---
apiVersion: v1
kind: Secret
metadata:
  name: app-cache-token
type: Opaque
data:
  my_password: dGVzdHJlZGlzCg==
---
apiVersion: redis.aeraki.io/v1alpha1
kind: RedisDestination
metadata:
  name: app-cache
spec:
  host: app-cache.redis.svc.cluster.local
  trafficPolicy:
    connectionPool:
      redis:
        mode: CLUSTER
        auth:
          secret:
            name: app-cache-token
            passwordField: my_password
```

*请注意：当前仅支持引用同命名空间下的 secret。*

## 集群外 Redis 资源的使用

当您使用集群外的 Redis 资源时，我们建议您使用 kubernetes 的[无选择器服务](https://kubernetes.io/docs/concepts/services-networking/service/#services-without-selectors)以配合 Aeraki 使用。

例如:

```yaml
apiVersion: v1
kind: Service
metadata:
  name: cache
spec:
  ports:
    - protocol: TCP
      port: 6379
      targetPort: 6379
---
apiVersion: v1
kind: Endpoints
metadata:
  name: cache
subsets:
  - addresses:
      - ip: 10.1.1.2 # 集群外 Redis 实例的地址，比如云厂商提供的 Redis 服务。
    ports:
      - port: 6379
```

## mTLS 支持

受限于当前 istio/envoy 存在的 bug，暂时不支持 mTLS。
