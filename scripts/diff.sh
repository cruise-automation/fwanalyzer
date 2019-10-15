#!/bin/sh

origname=$1
oldfile=$2
curfile=$3

diff -u $oldfile $curfile

exit 0
