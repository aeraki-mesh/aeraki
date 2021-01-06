BASEDIR=$(dirname "$0")

kubectl create ns thrift
kubectl label namespace thrift istio-injection=enabled --overwrite=true
kubectl apply -f $BASEDIR/thrift-sample.yaml -n thrift
kubectl apply -f $BASEDIR/destinationrule.yaml -n thrift
kubectl apply -f $BASEDIR/virtualservice-traffic-splitting.yaml -n thrift