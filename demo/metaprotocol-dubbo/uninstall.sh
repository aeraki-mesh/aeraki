BASEDIR=$(dirname "$0")

kubectl delete -f $BASEDIR/dubbo-sample.yaml -n meta-dubbo
kubectl delete -f $BASEDIR/serviceentry.yaml -n meta-dubbo
kubectl delete -f $BASEDIR/destinationrule.yaml -n meta-dubbo
kubectl delete ns meta-dubbo