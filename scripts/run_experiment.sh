#!/usr/bin/env sh
INPUT=$1
MASTER_NODE=$2
while IFS= read -r line
do
  VAR1=$(echo $line | cut -d' ' -f1)
  VAR2=$(echo $line | cut -d' ' -f2)
  VAR3=$(echo $line | cut -d' ' -f3)
  VAR3=$(expr $VAR3 - 1)
  echo $VAR1 $VAR2 $VAR3
  ./start_experiment.sh ../deploy/kubernetes $MASTER_NODE $VAR2 $VAR3 $3 ../logs $VAR1
done < "$INPUT"