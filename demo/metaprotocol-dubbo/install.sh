BASEDIR=$(dirname "$0")

kubectl create ns metaprotocol
kubectl label namespace metaprotocol istio-injection=enabled --overwrite=true
kubectl apply -f $BASEDIR/../../k8s/aeraki-bootstrap-config.yaml -n metaprotocol
kubectl apply -f $BASEDIR/metaprotocol-sample.yaml -n metaprotocol
kubectl apply -f $BASEDIR/serviceentry.yaml -n metaprotocol
kubectl apply -f $BASEDIR/destinationrule.yaml -n metaprotocol