BASEDIR=$(dirname "$0")/..

kubectl delete -f https://raw.githubusercontent.com/istio/istio/release-1.10/samples/addons/prometheus.yaml -n
istio-system
kubectl delete -f https://raw.githubusercontent.com/istio/istio/release-1.10/samples/addons/grafana.yaml -n istio-system
kubectl delete -f https://raw.githubusercontent.com/istio/istio/release-1.10/samples/addons/kiali.yaml -n istio-system

kubectl delete -f $BASEDIR/demo/gateway/demo-ingress.yaml -n istio-system
kubectl delete -f $BASEDIR/demo/gateway/istio-ingressgateway.yaml -n istio-system

bash $BASEDIR/demo/metaprotocol-dubbo/uninstall.sh
bash $BASEDIR/demo/thrift/uninstall.sh
bash ${BASEDIR}/demo/kafka/uninstall.sh

kubectl delete kiali kiali -n istio-system
kubectl delete ns istio-system
