#!/bin/bash

FILEPATH=$1
ORIG_FILENAME=$2
ORIG_UID=$3
ORIG_GID=$4
ORIG_MODE=$5
ORIG_SELINUXLABEL=$6

# this is an artificial test
if [ "$7" = "--" ]; then
    echo -n $9 $8
fi
