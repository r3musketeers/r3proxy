#!/usr/bin/env sh
ANSIBLE_SCRIPTS_DIR=$1

MASTER_NODE=$2

shift 1

echo "Connecting to nodes for the first time..."
for i in $@
do
  ssh -p 22 renantf@pc$i.emulab.net
done
echo "OK!"

echo "Running ansible scripts..."
ansible-playbook -i $ANSIBLE_SCRIPTS_DIR/hosts $ANSIBLE_SCRIPTS_DIR/01-setup.yaml
ansible-playbook -i $ANSIBLE_SCRIPTS_DIR/hosts --extra-vars "daemon_json_file=$ANSIBLE_SCRIPTS_DIR/daemon.json" $ANSIBLE_SCRIPTS_DIR/02-dependencies.yaml
ansible-playbook -i $ANSIBLE_SCRIPTS_DIR/hosts $ANSIBLE_SCRIPTS_DIR/03-init-master.yaml

echo "Install cluster network with the command:"
echo "\tkubectl apply -f https://raw.githubusercontent.com/coreos/flannel/master/Documentation/kube-flannel.yml >> pod_network_setup.txt"
ssh -p 22 renantf@pc$MASTER_NODE.emulab.net

ansible-playbook -i $ANSIBLE_SCRIPTS_DIR/hosts $ANSIBLE_SCRIPTS_DIR/04-join-workers.yaml
echo "OK!"

echo "Copying kube config to local machine..."
scp -r renantf@pc$MASTER_NODE.emulab.net:/users/renantf/.kube/config ~/.kube/config
echo "OK!"
