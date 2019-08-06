#!/bin/sh

FILEPATH=$1
ORIG_FILENAME=$2
ORIG_UID=$3
ORIG_GID=$4
ORIG_MODE=$5
ORIG_SELINUXLABEL=$6

INFO=$(file ${FILEPATH}|grep "ELF 32-bit LSB  executable, ARM, EABI5")

if [ -z "$INFO" ]; then
    echo -n ${ORIG_FILENAME} "not an ARM32 elf file"
fi
