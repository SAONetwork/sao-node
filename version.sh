#!/bin/bash
nodeVersion=$(git describe --tags --dirty 2>/dev/null || git rev-parse --short HEAD 2>/dev/null)_cons-
consInfo=$(go list -u -m --json github.com/SaoNetwork/sao)

if [[ `echo $consInfo | jq '.Replace == null'` == "false" ]];then
  consDir=$(echo $consInfo | jq -r '.Replace.Path')
  consVersion=$(git -C $consDir describe --tags --dirty 2>/dev/null || git -C $consDir rev-parse --short HEAD 2>/dev/null)
  nodeVersion+=$consVersion
else
  nodeVersion+=$(echo $consInfo | jq -r '.Version')
fi
echo $nodeVersion
