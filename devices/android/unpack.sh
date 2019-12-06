#!/bin/sh

# -- unpack android OTA --

if [ -z "$1" ]; then
    echo "syntax: $0 <android_ota.zip>"
    exit 1
fi
OTAFILE=$1

# tmpdir should contained 'unpacked' as last path element
TMPDIR=$(pwd)
if [ "$(basename $TMPDIR)" != "unpacked" ]; then
    echo "run script in directory named 'unpacked'"
    exit 1
fi

# unpack
unzip $OTAFILE >../unpack.log 2>&1
extract_android_ota_payload.py payload.bin >>../unpack.log 2>&1
mkboot boot.img boot_img >>../unpack.log 2>&1

# output targets, targets are consumed by check.py
# key = name of fwanalyzer config file without extension
#   e.g. 'system' => will look for 'system.toml'
# value = path to filesystem image (or directory)

# analyze system.img using system.toml
echo -n '{ "system": "unpacked/system.img" }'
