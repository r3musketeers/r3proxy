#!/usr/bin/env sh
LOGS_PATH=$1
EXPERIMENT_ID=$2
KUBE_FILES_DIR=$3

echo "Collecting metrics files..."
kubectl cp -n r3 $(kubectl get pods -n r3 -o=jsonpath='{.items[0].metadata.name}'):/tmp/throughput.log $LOGS_PATH/throughput$EXPERIMENT_ID.log
kubectl logs -n r3 $(kubectl get pods -n r3 -o=jsonpath='{.items[1].metadata.name}') > $LOGS_PATH/latency$EXPERIMENT_ID.log
echo "OK!"

echo "Deleting deployments..."
export SERVER_POD_IP=$(kubectl get pods -n r3 -o=jsonpath='{.items[0].status.podIP}')
envsubst < $KUBE_FILES_DIR/r3kvload-deployment.yml | kubectl delete -f -

kubectl delete -f $KUBE_FILES_DIR/r3httpkv-deployment.yml
echo "OK!"
