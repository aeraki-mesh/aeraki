alias k=kubectl

BASEDIR=$(dirname "$0")/..
SCRIPTS_DIR=$BASEDIR/test/e2e/scripts
COMMON_DIR=$BASEDIR/test/e2e/common
export ISTIO_VERSION=1.8.0
export BUILD_TAG=latest

bash ${SCRIPTS_DIR}/aeraki.sh
bash ${SCRIPTS_DIR}/istio.sh -y -f ${COMMON_DIR}/istio-config.yaml

k create ns dubbo
kubectl label namespace dubbo istio-injection=enabled --overwrite=true
k apply -f $BASEDIR/demo/dubbo/dubbo-sample.yaml -n dubbo
k apply -f $BASEDIR/demo/dubbo/serviceentry.yaml -n dubbo
k apply -f $BASEDIR/demo/dubbo/destinationrule.yaml -n dubbo
k apply -f $BASEDIR/demo/dubbo/virtualservice-traffic-splitting.yaml -n dubbo

k create ns thrift
kubectl label namespace thrift istio-injection=enabled --overwrite=true
k apply -f $BASEDIR/demo/thrift/thrift-sample.yaml -n thrift
k apply -f $BASEDIR/demo/thrift/destinationrule.yaml -n thrift
k apply -f $BASEDIR/demo/thrift/virtualservice-traffic-splitting.yaml -n thrift

kubectl apply -f https://raw.githubusercontent.com/istio/istio/release-1.8/samples/addons/prometheus.yaml
kubectl apply -f https://raw.githubusercontent.com/istio/istio/release-1.8/samples/addons/grafana.yaml
kubectl apply -f https://raw.githubusercontent.com/istio/istio/release-1.8/samples/addons/kiali.yaml

kubectl apply -f $BASEDIR/demo/gateway/demo-ingress.yaml -n istio-system
kubectl apply -f $BASEDIR/demo/gateway/istio-ingressgateway.yaml -n istio-system
