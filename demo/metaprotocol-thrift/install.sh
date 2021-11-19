BASEDIR=$(dirname "$0")

kubectl create ns meta-thrift
kubectl label namespace meta-thrift istio-injection=enabled --overwrite=true
kubectl apply -f $BASEDIR/../../k8s/aeraki-bootstrap-config.yaml -n meta-thrift
kubectl apply -f $BASEDIR/thrift-sample.yaml -n meta-thrift
kubectl apply -f $BASEDIR/destinationrule.yaml -n meta-thrift