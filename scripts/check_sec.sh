#!/bin/bash

FILEPATH=$1
ORIG_FILENAME=$2
ORIG_UID=$3
ORIG_GID=$4
ORIG_MODE=$5
ORIG_LABEL=$6
CONFIG=$8

RESULT=$(checksec --output=json --file="$1")

export RESULT
export FILEPATH
export CONFIG
export ORIG_FILENAME

# Config format is JSON
# array for values allows multiple acceptable values
# {"cfg":
#  {
#    "pie": ["yes"], 
#    "relo": ["full", "partial"] 
#  },
#  "skip": ["/usr/bin/bla"]
# }
#
# usable cfg fields, omitted fields are not checked:
# {
#	"canary": "no",
#	"fortify_source": "no",
#	"nx": "yes",
#	"pie": "no",
#	"relro": "partial",
#	"rpath": "no",
#	"runpath": "no",
#	"symbols": "no"
# }


python -c 'import json
import sys
import os

cfg = os.getenv("CONFIG")
res = os.getenv("RESULT")
fp = os.getenv("FILEPATH")
orig_name = os.getenv("ORIG_FILENAME")

expected = {}

try:
  expected = json.loads(cfg.rstrip())
except Exception:
  print("bad config: {}".format(cfg.rstrip()))
  sys.exit(1)

try:
  result = json.loads(res.rstrip())

  if "skip" in expected: 
    if orig_name in expected["skip"]:
      sys.exit(0)

  if not fp in result:
    fp = "file"

  for k in expected["cfg"]:
    if k in result[fp]:
      passed = False
      for expected_value in expected["cfg"][k]:
        if expected_value == result[fp][k]:
          passed = True
          break
      if not passed:
        print(json.dumps(result[fp]).rstrip())
        sys.exit(0)

except Exception as e:
  if not "Not an ELF file:" in res:
     print(e)

sys.exit(0)
'
