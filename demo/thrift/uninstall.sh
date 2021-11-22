BASEDIR=$(dirname "$0")

kubectl delete -f $BASEDIR/thrift-sample.yaml -n thrift
kubectl delete -f $BASEDIR/destinationrule.yaml -n thrift
kubectl delete -f $BASEDIR/virtualservice-traffic-splitting.yaml -n thrift
kubectl delete ns thrift