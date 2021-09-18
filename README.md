# Aeraki

[![CI Tests](https://github.com/aeraki-framework/aeraki/workflows/ci/badge.svg?branch=master)](https://github.com/aeraki-framework/aeraki/actions?query=branch%3Amaster+event%3Apush+workflow%3A%22ci%22)
[![E2E Tests](https://github.com/aeraki-framework/aeraki/workflows/e2e-metaprotocol/badge.svg?branch=master)](https://github.com/aeraki-framework/aeraki/actions?query=branch%3Amaster+event%3Apush+workflow%3A%22e2e-metaprotocol%22)
[![E2E Tests](https://github.com/aeraki-framework/aeraki/workflows/e2e-dubbo/badge.svg?branch=master)](https://github.com/aeraki-framework/aeraki/actions?query=branch%3Amaster+event%3Apush+workflow%3A%22e2e-dubbo%22)
[![E2E Tests](https://github.com/aeraki-framework/aeraki/workflows/e2e-thrift/badge.svg?branch=master)](https://github.com/aeraki-framework/aeraki/actions?query=branch%3Amaster+event%3Apush+workflow%3A%22e2e-thrift%22)
[![E2E Tests](https://github.com/aeraki-framework/aeraki/workflows/e2e-kafka-zookeeper/badge.svg?branch=master)](https://github.com/aeraki-framework/aeraki/actions?query=branch%3Amaster+event%3Apush+workflow%3A%22e2e-kafka-zookeeper%22)
[![E2E Tests](https://github.com/aeraki-framework/aeraki/workflows/e2e-redis/badge.svg?branch=master)](https://github.com/aeraki-framework/aeraki/actions?query=branch%3Amaster+event%3Apush+workflow%3A%22e2e-redis%22)

# Manage **any** layer 7 traffic in Istio service mesh!
![ Aeraki ](docs/aeraki&istio.png)
---
Aeraki [Air-rah-ki] is the Greek word for 'breeze'. While Istio connects microservices in a service mesh, Aeraki provides a framework to allow Istio to support more layer 7 protocols other than just HTTP and gRPC. We hope that this breeze can help Istio sail a little further.

In a nutshell, you can think of Aeraki as the [Operator pattern](https://kubernetes.io/docs/concepts/extend-kubernetes/operator/) in Istio to automate the envoy configuration for layer 7 protocols.

IstioCon2021 Talk: [How to Manage Any Layer-7 Traffic in an Istio Service Mesh?](https://www.youtube.com/watch?v=sBS4utF68d8)

## Architecture
![ Aeraki ](docs/aeraki-architecture.png)

## Problems to solve

We are facing some challenges in service mesh:
* Istio and other service mesh implementations have very limited support for layer 7 protocols other than HTTP and gRPC.
* Envoy RDS(Route Discovery Service) is solely designed for HTTP and gRPC. Other protocols such as Dubbo and Thrift
 can only use listener in-line routes for traffic management, which breaks existing connections when routes change.
* It takes a lot of effort to introduce a proprietary protocol into a service mesh. You'll need to write an Envoy
 filter to handle the traffic in the data plane, and a control plane to manage those Envoys.

## Aeraki's approach

To address these problems, Aeraki Framework providing a non-intrusive, extendable way to manage any layer 7 traffic in a service mesh.

Aeraki Framework consists of the following components:
* Aeraki: [Aeraki](https://github.com/aeraki-framework/aeraki) provides high-level, user-friendly traffic management rules 
to operations, translates the rules to envoy filter configurations, and leverages Istio's EnvoyFilter API to push the 
configurations to the sidecar proxies. Aeraki also serves as the RDS server for metaprotocol proxies in the data plane. 
Contrary to Envoy RDS, which focuses on HTTP, Aeraki RDS is aimed to provide general dynamic route capabilities for all 
layer-7 protocols. 
* MetaProtocol: [MetaProtocol](https://github.com/aeraki-framework/meta-protocol-proxy) provides common capabilities for
 Layer-7 protocols, such as load balancing, circuit breaker, load balancing, routing, rate limiting, fault injection, and 
 auth. Layer-7 protocols can be built on top of MetaProtocol. To add a new protocol into the service mesh, the only thing 
 you need to do is implementing the [codec interface](https://github.com/aeraki-framework/meta-protocol-proxy/blob/ac788327239bd794e745ce18b382da858ddf3355/src/meta_protocol_proxy/codec/codec.h#L118). 
 If you have special requirements which can't be meet by the built-in capabilities, MetaProtocol Proxy also has a filter chain mechanism, 
 allowing users to write their own layer-7 filters to add custom logic into MetaProtocol Proxy. 
 
 [Dubbo](https://github.com/aeraki-framework/meta-protocol-proxy/tree/master/src/application_protocols/dubbo) and 
 [Thrift](https://github.com/aeraki-framework/meta-protocol-proxy/tree/master/src/application_protocols/thrift) have
  already been implemented based on MetaProtocol. More protocols are on the way. 
  
  Need to manage proprietary protocols in Istio service mesh? All you need to do is just implementing the
    [codec interface](https://github.com/aeraki-framework/meta-protocol-proxy/blob/ac788327239bd794e745ce18b382da858ddf3355/src/meta_protocol_proxy/codec/codec.h#L118). Aeraki Framework will take care all the others for you. It's so easy!

## Reference
* [Create plugins for proprietary protocols](docs/metaprotocol.md)
* [Dubbo (中文) ](https://github.com/aeraki-framework/dubbo2istio#readme)
* [Redis (中文) ](docs/zh/redis.md)
* [LazyXDS（xDS 按需加载）](lazyxds/README.md)

## Supported protocols:
* Dubbo
  * Service Discovery
    * [x] ServiceEntry Integration ([Example](https://github.com/aeraki-framework/aeraki/blob/master/demo/dubbo/serviceentry.yaml))
    * [x] [ZooKeeper Integration](https://github.com/aeraki-framework/dubbo2istio)
    * [x] [Nacos Integration](https://github.com/aeraki-framework/dubbo2istio)
  * Traffic Management
    * [x] Request Level Load Balancing
    * [x] Version Based Routing
    * [x] Traffic Splitting
    * [x] Method Based Routing
    * [x] Header Based Routing
    * [x] Crcuit Breaker
    * [x] Locality Load Balancing
    * [x] RDS(Route Discovery Service)
  * Observability
    * [x] Dubbo Request Metrics
  * Security 
    * [x] Peer Authorization on Interface/Method
    * [ ] Rquest Authorization
* Thrift
  * Traffic Management
    * [x] Request Level Load Balancing
    * [x] Version Based Routing
    * [x] Traffic Splitting
    * [ ] Header Based Routing
    * [ ] Rate Limit
  * Observability
    * [x] Thrift Request Metrics
* Kafka
  * [x] Metrics
* ZooKeeper
  * [x] Metrics
* Redis
  * [x] Redis Cluster
  * [x] Sharding
  * [x] Prefix Routing
  * [x] Auth
  * [x] Traffic Mirroring
* [ ] MySql
* [ ] MongoDB
* [ ] Postgres
* [ ] RocketMQ
* [ ] TARS
* ...

## Demo

[Live Demo: kiali Dashboard](http://aeraki.zhaohuabing.com:20001/)

[Live Demo: Service Metrics: Grafana](http://aeraki.zhaohuabing.com:3000/d/pgz7wp-Gz/aeraki-demo?orgId=1&refresh=10s&kiosk)

[Live Demo: Service Metrics: Prometheus](http://aeraki.zhaohuabing.com:9090/new/graph?g0.expr=envoy_dubbo_inbound_20880___response_success&g0.tab=0&g0.stacked=1&g0.range_input=1h&g1.expr=envoy_dubbo_outbound_20880__org_apache_dubbo_samples_basic_api_demoservice_request&g1.tab=0&g1.stacked=1&g1.range_input=1h&g2.expr=envoy_thrift_inbound_9090___response&g2.tab=0&g2.stacked=1&g2.range_input=1h&g3.expr=envoy_thrift_outbound_9090__thrift_sample_server_thrift_svc_cluster_local_response_success&g3.tab=0&g3.stacked=1&g3.range_input=1h&g4.expr=envoy_thrift_outbound_9090__thrift_sample_server_thrift_svc_cluster_local_request&g4.tab=0&g4.stacked=1&g4.range_input=1h)

Screenshot: Service Metrics:
![Screenshot: Service Metrics](docs/metrics.png)

Recored Demo: Dubbo and Thrift Traffic Management
[![Thrift and Dubbo traffic management demo](http://i3.ytimg.com/vi/vrjp-Yg3Leg/maxresdefault.jpg)](https://www.youtube.com/watch?v=vrjp-Yg3Leg)

## Install

### Pre-requirements:
* A running Kubernetes cluster, which can be either a cluster in the cloud, or a local cluster created with kind/minikube
* Kubectl installed, and the `~/.kube/conf` points to the cluster in the first step
* Helm installed, which will be used to install some components in the demo

### Download Aeraki from the Github
```bash
git clone https://github.com/aeraki-framework/aeraki.git
```

### Install Aeraki and demo applications
```bash
aeraki/demo/install-demo.sh
```

### Open the following URLs in your browser to play with Aeraki and view service metrics
* Kaili `http://{istio-ingressgateway_external_ip}:20001`
* Grafana `http://{istio-ingressgateway_external_ip}:3000`
* Prometheus `http://{istio-ingressgateway_external_ip}:9090`

You can import Aeraika demo dashboard from file `demo/aeraki-demo.json` into the Grafana.

## Contact
* Mail: If you're interested in contributing to this project, please reach out to zhaohuabing@gmail.com
* Wechat Group: Please contact Wechat ID: zhao_huabing to join the Aeraki Wechat group
* Slack: Join [Aeraki slack channel](http://aeraki.slack.com/)
