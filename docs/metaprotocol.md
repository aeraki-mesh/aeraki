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

With the help of Aeraki Mesh, you can manage proprietary protocols in Istio Service Mesh. Just follow the 
below steps:

## Data Plane

Quickly build an Envoy Filter for a proprietary protocol on top of MetaProtocol:

* Implement the [codec interface](https://github.com/aeraki-mesh/meta-protocol-proxy/blob/ac788327239bd794e745ce18b382da858ddf3355/src/meta_protocol_proxy/codec/codec.h#L118) 
. You can refer to [Dubbo codec](https://github.com/aeraki-mesh/meta-protocol-proxy/tree/master/src/application_protocols/dubbo) and 
                    [Thrift codec](https://github.com/aeraki-mesh/meta-protocol-proxy/tree/master/src/application_protocols/thrift) as writing your own implementation.

* Define the protocol with Aeraki ApplicationProtocol CRD.
Below is an example of the Dubbo protocol:
```yaml
apiVersion: metaprotocol.aeraki.io/v1alpha1
kind: ApplicationProtocol
metadata:
  name: dubbo
  namespace: istio-system
spec:
  protocol: dubbo
  codec: aeraki.meta_protocol.codec.dubbo
```

## Control Plane
You don't need to implement the control plane. Aeraki watches services and traffic rules, generate the
 configurations for the sidecar proxies, and send the configurations to the data plane via EnvoyFilter and MetaRoute
  RDS. 

## Protocol selection

Similar to Istio, protocols are identified by service port prefix. Please name service ports with this pattern: 
tcp-metaprotocol-{layer-7-protocol}-xxx. For example, a Dubbo service port could be named tcp-metaprotocol-dubbo. 

## Traffic management

You can change the route via MetaRouter CRD. For example:

* Route the Dubbo requests calling method sayHello to v2:
```yaml
apiVersion: metaprotocol.aeraki.io/v1alpha1
kind: MetaRouter
metadata:
  name: test-metaprotocol-route
spec:
  hosts:
    - org.apache.dubbo.samples.basic.api.demoservice
  routes:
    - name: v2
    - match:
        attributes:
          method:
            exact: sayHello
      route:
        - destination:
            host: org.apache.dubbo.samples.basic.api.demoservice
            subset: v2
```

* Send 20% of the requests to v1 and 80% to v2:
```yaml
apiVersion: metaprotocol.aeraki.io/v1alpha1
kind: MetaRouter
metadata:
  name: test-metaprotocol-route
spec:
  hosts:
    - org.apache.dubbo.samples.basic.api.demoservice
  routes:
    - name: traffic-spilt
      route:
        - destination:
            host: org.apache.dubbo.samples.basic.api.demoservice
            subset: v1
          weight: 20
        - destination:
            host: org.apache.dubbo.samples.basic.api.demoservice
            subset: v2
          weight: 80
```

