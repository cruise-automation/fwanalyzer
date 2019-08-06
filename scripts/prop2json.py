#!/usr/bin/python3

#
# read Android property file and convert it to JSON
#

import json
import sys

props = {}

with open(sys.argv[1], 'r') as fp:
    while True:
        line = fp.readline()
        if not line:
            break
        if line.startswith('#'):
            continue
        
        line = line.rstrip("\n")
        parts = line.split("=", 1)
        if len(parts) == 2:
            props[parts[0]] = parts[1]
print(json.dumps(props))
