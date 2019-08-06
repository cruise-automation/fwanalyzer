#!/bin/sh

FILEPATH=$1
ORIG_FILENAME=$2
ORIG_UID=$3
ORIG_GID=$4
ORIG_MODE=$5
ORIG_SELINUXLABEL=$6

INFO=$(file ${FILEPATH}|grep "ELF 64-bit LSB  executable, ARM aarch64")

if [ -z "$INFO" ]; then
    echo -n ${ORIG_FILENAME} "not an ARM aarch64 elf file"
fi
