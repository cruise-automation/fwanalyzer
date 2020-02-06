#!/usr/bin/env python3

# Copyright 2019-present, Cruise LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.


import json
import tempfile
import os
import os.path
import sys
import argparse
import subprocess
import hashlib


class CheckOTA:
    def __init__(self, fwanalyzer="fwanalyzer"):
        self._tmpdir = tempfile.mktemp()
        self._unpackdir = os.path.join(self._tmpdir, "unpacked")
        self._fwanalyzer = fwanalyzer

    def getTmpDir(self):
        return self._tmpdir

    def setUnpacked(self, unpacked):
        self._tmpdir = os.path.realpath(unpacked + "/..")
        self._unpackdir = os.path.realpath(unpacked)

    def runFwAnalyzeFs(self, img, cfg, cfginc, out):
        cfginclude = ""
        if cfginc:
            cfginclude = " -cfgpath " + cfginc
        cmd = self._fwanalyzer + " -in " + img + cfginclude + " -cfg " + cfg + " -out " + out
        subprocess.check_call(cmd, shell=True)

    def unpack(self, otafile, otaunpacker, mkboot):
        # customize based on firmware
        #
        # create tmp + unpackdir
        cmd = "mkdir -p " + self._unpackdir
        subprocess.check_call(cmd, shell=True)
        cmd = "unzip " + otafile
        subprocess.check_call(cmd, shell=True, cwd=self._unpackdir)
        # unpack payload
        cmd = otaunpacker + " payload.bin"
        subprocess.check_call(cmd, shell=True, cwd=self._unpackdir)
        # unpack boot.img
        cmd = mkboot + " boot.img boot_img"
        subprocess.check_call(cmd, shell=True, cwd=self._unpackdir)

    def delTmpDir(self):
        cmd = "rm -rf " + self._tmpdir
        subprocess.check_call(cmd, shell=True)

    # check result json
    def checkResult(self, result):
        with open(result) as read_file:
            data = json.load(read_file)

        if "offenders" in data:
            status = False
        else:
            status = True

        return (status, json.dumps(data, sort_keys=True, indent=2))


def getCfg(name):
    return name + ".toml"


def getOut(name):
    return name + "_out.json"


def getImg(name):
    if name == "boot":
        return "unpacked/"
    return "unpacked/" + name + ".img"


def hashfile(fpath):
    m = hashlib.sha256()
    with open(fpath, 'rb') as f:
        while True:
            data = f.read(65535)
            if not data:
                break
            m.update(data)
    return m.hexdigest()


def makeReport(ota, data):
    report = {}
    report["firmware"] = ota
    status = True
    for key in data:
        s, r = out[key]
        if not s:
            status = s
        report[key] = json.loads(r)

    report["firmware_digest"] = hashfile(ota)
    report["status"] = status
    return json.dumps(report, sort_keys=True, indent=2)


if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument('--ota', action='store', required=True, help="path to ota file")
    parser.add_argument('--cfg-path', action='store', required=True, help="path to directory containing config files")
    parser.add_argument('--cfg-include-path', action='store', help="path to config include files")
    parser.add_argument('--report', action='store', help="report file")
    parser.add_argument('--keep-unpacked', action='store_true', help="keep unpacked data")
    parser.add_argument('--targets', nargs='+', action='store', help="image targets e.g.: system vendor boot")
    parser.add_argument('--fwanalyzer-bin', action='store', default="fwanalyzer", help="path to fwanalyzer binary")
    args = parser.parse_args()

    # target file system images, a fwanalyzer config file is required for each of those
    targets = ["system", "vendor", "dsp", "boot"]

    # use target list from cmdline
    if args.targets:
        targets = args.targets

    out = {}

    for tgt in targets:
        if not os.path.isfile(os.path.join(args.cfg_path, getCfg(tgt))):
            print("OTA Check skipped, config file does not exist")
            sys.exit(0)

    ota = os.path.realpath(args.ota)
    cfg = os.path.realpath(args.cfg_path)
    otaunpacker = "extract_android_ota_payload.py"
    bootunpacker = "mkboot"

    check = CheckOTA(args.fwanalyzer_bin)
    if not ota.endswith("unpacked"):
        check.unpack(ota, otaunpacker, bootunpacker)
    else:
        check.setUnpacked(ota)
        args.keep_unpacked = True
        print("already unpacked")

    all_checks_ok = True
    for tgt in targets:
        check.runFwAnalyzeFs(os.path.join(check.getTmpDir(), getImg(tgt)),
                             os.path.join(cfg, getCfg(tgt)), args.cfg_include_path, getOut(tgt))
        ok, data = check.checkResult(getOut(tgt))
        out[tgt] = ok, data
        if not ok:
            all_checks_ok = False

    if args.keep_unpacked:
        print("unpacked: {0}\n".format(check.getTmpDir()))
    else:
        check.delTmpDir()

    report = makeReport(args.ota, out)
    if args.report != None:
        fp = open(args.report, "w+")
        fp.write(report)
        fp.close()
        print("report written to: " + args.report)

    if not all_checks_ok:
        print(report)
        print("OTA Check Failed")
        sys.exit(1)
    else:
        print("OTA Check Success")
        sys.exit(0)
