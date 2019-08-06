#!/bin/sh

FILEPATH=$1
ORIG_FILENAME=$2
ORIG_UID=$3
ORIG_GID=$4
ORIG_MODE=$5
ORIG_SELINUXLABEL=$6

ZIP=$(echo ${ORIG_FILENAME}|grep -e "\.zip")

if [ -n "$ZIP" ]; then

  DIR=$(dirname ${FILEPATH})
  mkdir ${DIR}/otacertdata
  pushd
  cd ${DIR}/otacertdata
  unzip ${FILEPATH} >/dev/null 2>&1
  popd
  find ${DIR}/otacertdata -name "*" -exec scripts/check_cert.sh {} {} \;
  rm -rf ${DIR}/otacertdata

fi
