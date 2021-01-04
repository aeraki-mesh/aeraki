BASEDIR=$(dirname "$0")

kubectl create ns kafka
kubectl label namespace kafka istio-injection=enabled --overwrite=true
helm repo add bitnami https://charts.bitnami.com/bitnami
helm repo add zhaohuabing http://zhaohuabing.com/helm-repo/
helm repo update
helm install my-release --set persistence.enabled=false --set zookeeper.persistence.enabled=false bitnami/kafka -n kafka
kubectl apply -f $BASEDIR/kafka-sample.yaml -n kafka

