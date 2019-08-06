#!/bin/sh

FILEPATH=$1
ORIG_FILENAME=$2
ORIG_UID=$3
ORIG_GID=$4
ORIG_MODE=$5
ORIG_SELINUXLABEL=$6

INFO=$(file ${FILEPATH} | grep "private key")
PERMS=$(echo ${ORIG_MODE} | grep -E ".*r[-w][-x]$")

if [ -n "$INFO" ]; then
    echo -n ${ORIG_FILENAME} "is a private key"
    if [ -n "$PERMS" ]; then
        echo -n " that is world readable"
    fi
fi
