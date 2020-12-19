#!/usr/bin/env sh
KUBE_FILES_DIR=$1
MASTER_NODE=$2
export N_JOBS=$3

echo "Applying server deployment..."
kubectl apply -f $KUBE_FILES_DIR/r3httpkv-deployment.yml
echo "OK!"

echo "Waiting server to be running..."
while [[ "$(curl -s -o /dev/null -w ''%{http_code}'' http://pc$MASTER_NODE.emulab.net:30001/ping)" -ne "200" ]]; do
  sleep 10;
done
echo "OK!"

echo "Applying job deployment..."
export EXTRA_THREADS=$4
export SLEEP_DURATION=$5
export SERVER_POD_IP=$(kubectl get pods -n r3 -o=jsonpath='{.items[0].status.podIP}')

echo $SERVER_POD_IP

envsubst < $KUBE_FILES_DIR/r3kvload-deployment.yml
envsubst < $KUBE_FILES_DIR/r3kvload-deployment.yml | kubectl apply -f -
echo "OK!"

echo "Waiting job to complete..."
kubectl wait -n r3 --for=condition=complete --timeout=1h job.batch/r3kvload
echo "OK!"

LOGS_PATH=$6
EXPERIMENT_ID=$7

echo "Collecting metrics files..."
kubectl cp -n r3 $(kubectl get pods -n r3 -o=jsonpath='{.items[0].metadata.name}'):/tmp/throughput.log $LOGS_PATH/throughput$EXPERIMENT_ID.log
kubectl logs -n r3 $(kubectl get pods -n r3 -o=jsonpath='{.items[1].metadata.name}') > $LOGS_PATH/latency$EXPERIMENT_ID.log
echo "OK!"

echo "Deleting deployments..."
export SERVER_POD_IP=$(kubectl get pods -n r3 -o=jsonpath='{.items[0].status.podIP}')
envsubst < $KUBE_FILES_DIR/r3kvload-deployment.yml | kubectl delete -f -

kubectl delete -f $KUBE_FILES_DIR/r3httpkv-deployment.yml
echo "OK!"
