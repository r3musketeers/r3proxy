#!/usr/bin/env sh
ROLE_LABEL=$1
shift 1

for i in $@                               
do
  kubectl label nodes node$i.r3exp.scalablesmr.emulab.net role=$ROLE_LABEL
done
