#!/bin/sh

FILEPATH=$1
ORIG_FILENAME=$2
ORIG_UID=$3
ORIG_GID=$4
ORIG_MODE=$5
ORIG_SELINUXLABEL=$6


ISPEM=$(file ${FILEPATH} |grep PEM)

if [ -n "$ISPEM" ]; then

  openssl x509 -noout -text -in ${FILEPATH} | grep Issuer:|sed 's/^ *//'
  openssl x509 -noout -text -in ${FILEPATH} | grep Subject:|sed 's/^ *//'

fi
