#!/bin/bash
set -e

export CATTLE_HOME=${CATTLE_HOME:-/var/lib/cattle}
MAIN=/usr/bin/agent
export PATH=${CATTLE_HOME}/bin:$PATH

cd $(dirname $0)

CONF=(${CATTLE_HOME}/pyagent/agent.conf
        /etc/cattle/agent/agent.conf
        ${CATTLE_HOME}/etc/cattle/agent/agent.conf
        /var/lib/rancher/etc/agent.conf)

for conf_file in "${CONF[@]}"; do
    if [ -e $conf_file ]
    then
        source $conf_file
    fi
done

echo Executing $MAIN
exec $MAIN
