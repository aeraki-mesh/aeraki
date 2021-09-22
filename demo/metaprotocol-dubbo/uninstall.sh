BASEDIR=$(dirname "$0")

kubectl delete -f $BASEDIR/metaprotocol-sample.yaml -n metaprotocol
kubectl delete -f $BASEDIR/serviceentry.yaml -n metaprotocol
kubectl delete -f $BASEDIR/destinationrule.yaml -n metaprotocol
kubectl delete ns metaprotocol