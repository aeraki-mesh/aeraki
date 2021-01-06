BASEDIR=$(dirname "$0")/..

SCRIPTS_DIR=$BASEDIR/test/e2e/scripts
COMMON_DIR=$BASEDIR/test/e2e/common
export ISTIO_VERSION=1.8.0
export BUILD_TAG=latest

bash ${SCRIPTS_DIR}/aeraki.sh
bash ${SCRIPTS_DIR}/istio.sh -y -f ${COMMON_DIR}/istio-config.yaml

kubectl apply -f https://raw.githubusercontent.com/istio/istio/release-1.8/samples/addons/prometheus.yaml
kubectl apply -f https://raw.githubusercontent.com/istio/istio/release-1.8/samples/addons/grafana.yaml
kubectl apply -f https://raw.githubusercontent.com/istio/istio/release-1.8/samples/addons/kiali.yaml

kubectl apply -f $BASEDIR/demo/gateway/demo-ingress.yaml -n istio-system
kubectl apply -f $BASEDIR/demo/gateway/istio-ingressgateway.yaml -n istio-system

bash $BASEDIR/demo/dubbo/install.sh
bash $BASEDIR/demo/thrift/install.sh
bash ${BASEDIR}/demo/kafka/install.sh