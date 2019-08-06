#!/bin/sh

FILEPATH=$1
ORIG_FILENAME=$2
ORIG_UID=$3
ORIG_GID=$4
ORIG_MODE=$5
ORIG_SELINUXLABEL=$6

APK=$(echo ${ORIG_FILENAME}|grep -e "\.apk")

if [ -n "$APK" ]; then

  DIR=$(dirname ${FILEPATH})
  mkdir ${DIR}/apkdata
  cd ${DIR}/apkdata; unzip ${FILEPATH} >/dev/null 2>&1; openssl cms -cmsout -noout -text -print -in META-INF/CERT.RSA -inform DER |grep subject:|sed 's/^ *//'
  rm -rf ${DIR}/apkdata

fi
