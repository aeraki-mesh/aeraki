BASEDIR=$(dirname "$0")

helm delete my-release -n kafka
kubectl delete -f $BASEDIR/kafka-sample.yaml -n kafka

kubectl delete ns kafka

