BASEDIR=$(dirname "$0")

kubectl delete -f $BASEDIR/dubbo-sample.yaml -n dubbo
kubectl delete -f $BASEDIR/serviceentry.yaml -n dubbo
kubectl delete -f $BASEDIR/destinationrule.yaml -n dubbo
kubectl delete -f $BASEDIR/virtualservice-traffic-splitting.yaml -n dubbo
kubectl delete ns dubbo