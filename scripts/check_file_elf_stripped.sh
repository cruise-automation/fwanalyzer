#!/bin/sh

FILEPATH=$1
ORIG_FILENAME=$2
ORIG_UID=$3
ORIG_GID=$4
ORIG_MODE=$5
ORIG_SELINUXLABEL=$6

INFO=$(file ${FILEPATH}|grep "not stripped")

if [ -n "$INFO" ]; then
    echo -n ${ORIG_FILENAME} "is not stripped"
fi
