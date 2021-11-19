BASEDIR=$(dirname "$0")

kubectl create ns meta-dubbo
kubectl label namespace meta-dubbo istio-injection=enabled --overwrite=true
kubectl apply -f $BASEDIR/../../k8s/aeraki-bootstrap-config.yaml -n meta-dubbo
kubectl apply -f $BASEDIR/dubbo-sample.yaml -n meta-dubbo
kubectl apply -f $BASEDIR/serviceentry.yaml -n meta-dubbo
kubectl apply -f $BASEDIR/destinationrule.yaml -n meta-dubbo