BASEDIR=$(dirname "$0")

kubectl create ns dubbo
kubectl label namespace dubbo istio-injection=enabled --overwrite=true
kubectl apply -f $BASEDIR/dubbo-sample.yaml -n dubbo
kubectl apply -f $BASEDIR/serviceentry.yaml -n dubbo
kubectl apply -f $BASEDIR/destinationrule.yaml -n dubbo
kubectl apply -f $BASEDIR/virtualservice-traffic-splitting.yaml -n dubbo