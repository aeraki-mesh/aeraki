BASEDIR=$(dirname "$0")

kubectl delete -f $BASEDIR/thrift-sample.yaml -n meta-thrift
kubectl delete -f $BASEDIR/destinationrule.yaml -n meta-thrift
kubectl delete ns meta-thrift