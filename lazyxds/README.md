# Lazyxds

Lazyxds enables Istio only push needed xds to sidecars to reduce resource consumption and speed up xds configuration propagation.

## Problems to solve

![SotW xDS](docs/images/sotw-xds.png)

## Architecture

![SotW xDS](docs/images/arch.png)

## Install

### Pre-requirements:

* A running Kubernetes cluster, and istio(version >= 1.10.0) installed
* Kubectl installed, and the `~/.kube/conf` points to the cluster in the first step

### Install Lazyxds egress and controller

```
kubectl apply -f https://raw.githubusercontent.com/aeraki-framework/aeraki/master/lazyxds/install/lazyxds-egress.yaml
kubectl apply -f https://raw.githubusercontent.com/aeraki-framework/aeraki/master/lazyxds/install/lazyxds-controller.yaml
```

These steps install the lazyxds egress and controller into istio-system namespace.

## How to enable Lazy xDS

There are 2 ways to enable lazy xDS: per service, or per namespace. You just need add annotation `lazy-xds: "true"` to the target service or namespace.

### Enable per Service

```
apiVersion: v1
kind: Service
metadata:
  name: my-service
  annotations:
    lazy-xds: "true"
spec:
```

or use kubectl: 

`kubectl annotate service my-service lazy-xds=true --overwrite`

### Enable per Namespace

```
apiVersion: v1
kind: Namespace
metadata:
  name: my-namespace
  annotations:
    lazy-xds: "true"
spec:
```

or use kubectl: 

`kubectl annotate namespace my-namespace lazy-xds=true --overwrite`

## Bookinfo Demo

1. Install istio(version >= 1.10.0), and enable access log for debug purpose.

    ```
    istioctl install -y --set meshConfig.accessLogFile=/dev/stdout
    ```

2. Install lazyxds by following the instructions in [Install Lazyxds egress and controller](https://github.com/aeraki-framework/aeraki/blob/master/lazyxds/README.md#install-lazyxds-egress-and-controller).

3. Install bookinfo application:

    ```
    kubectl label namespace default istio-injection=enabled
    kubectl apply -f https://raw.githubusercontent.com/istio/istio/release-1.10/samples/bookinfo/platform/kube/bookinfo.yaml
    kubectl apply -f https://raw.githubusercontent.com/istio/istio/release-1.10/samples/bookinfo/networking/bookinfo-gateway.yaml
    ```
   
    Determine the ingress IP, and we use 80 as ingress port by default.
    ```
    export INGRESS_HOST=$(kubectl -n istio-system get service istio-ingressgateway -o jsonpath='{.status.loadBalancer.ingress[0].ip}')
    ```
   
    Save product page pod name to env for later use.
    ```
    export PRODUCT_PAGE_POD=$(kubectl get pod -l app=productpage -o jsonpath="{.items[0].metadata.name}")
    ```
   
    Check the eds of product page pod, we can see product page gets all eds of bookinfo, though it does not need all of them:
    ```
    istioctl pc endpoints $PRODUCT_PAGE_POD | grep '9080'
    172.22.0.10:9080                 HEALTHY     OK                outbound|9080||reviews.default.svc.cluster.local
    172.22.0.11:9080                 HEALTHY     OK                outbound|9080||reviews.default.svc.cluster.local
    172.22.0.12:9080                 HEALTHY     OK                outbound|9080||reviews.default.svc.cluster.local
    172.22.0.13:9080                 HEALTHY     OK                outbound|9080||productpage.default.svc.cluster.local
    172.22.0.8:9080                  HEALTHY     OK                outbound|9080||details.default.svc.cluster.local
    172.22.0.9:9080                  HEALTHY     OK                outbound|9080||ratings.default.svc.cluster.local
    ```

4. Enable lazy xds for the productpage service:

    ```
    kubectl annotate service productpage lazy-xds=true --overwrite
    ```
   
    Check the eds of product page:
    ```
    istioctl pc endpoints $PRODUCT_PAGE_POD | grep '9080'
    // no eds show
    ```
    After enable lazy loading, product page pod will not get any endpoints of bookinfo.

5. Access bookinfo the first time:

    ```
    curl -I "http://${INGRESS_HOST}/productpage"
    ```
   
   check the access log of product page pod:
   
   ```
   kubectl logs -c istio-proxy -f $PRODUCT_PAGE_POD
   ```
   
   ![access to egress](docs/images/productpage-accesslog-1.png)
   
   We can see the first access form product page to details and reviews go to `istio-egressgateway-lazyxds`
   
   Check the eds of product page again:
   
   ```
   172.22.0.10:9080                 HEALTHY     OK                outbound|9080||reviews.default.svc.cluster.local
   172.22.0.11:9080                 HEALTHY     OK                outbound|9080||reviews.default.svc.cluster.local
   172.22.0.12:9080                 HEALTHY     OK                outbound|9080||reviews.default.svc.cluster.local
   172.22.0.8:9080                  HEALTHY     OK                outbound|9080||details.default.svc.cluster.local
   ```
   
   We can see there are only reviews and details endpoints, which are the endpoints product page just need.

6. Access bookinfo again:

   ```
   curl -I "http://${INGRESS_HOST}/productpage"
   ```
    
   Check the access log of product page pod:
   
   ```
   kubectl logs -c istio-proxy -f $PRODUCT_PAGE_POD
   ```

   ![access to egress](docs/images/productpage-accesslog-2.png)
   
   We can see the traffic always go to the target services directly, will not proxy to `istio-egressgateway-lazyxds` anymore.
 
## Uninstall

```
kubectl delete -f https://raw.githubusercontent.com/aeraki-framework/aeraki/master/lazyxds/install/lazyxds-controller.yaml
kubectl delete -f https://raw.githubusercontent.com/aeraki-framework/aeraki/master/lazyxds/install/lazyxds-egress.yaml
```

## Performance

We setup two bookinfo applications in one istio mesh with lazyxds installed, the product page in `lazy-on` namespace enable lazy xds, and the another is not.
Then we use [istio load testing](https://github.com/istio/tools/tree/master/perf/load) to construct large size services increasingly, 
each load test namespace contains 19 services, each service contains 5 pods.

![performance-test-arch](docs/images/performance-test-arch.png)
   
Memory compare:
   
![performance-test-mem](docs/images/performance-test-mem.png)

EDS and CDS compare:

![performance-test-xds](docs/images/performance-test-xds.png)
