#!/usr/bin/env python3

# Copyright 2019 GM Cruise LLC
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

import argparse
import hashlib
import json
import os
import sys
import subprocess
import tempfile

class CheckFirmware:
    def __init__(self, fwanalyzer="fwanalyzer"):
        self._tmpdir = ""
        self._unpackdir = ""
        self._fwanalyzer = fwanalyzer
        self._unpacked = False

    def get_tmp_dir(self):
        return self._tmpdir

    def run_fwanalyzer_fs(self, img, cfg, cfginc, out, options=""):
        cfginclude = ""
        if cfginc:
            cfginclude = " -cfgpath {0}".format(cfginc)
        cmd = "{0} -in {1} {2} -cfg {3} -out {4} {5}".format(self._fwanalyzer, img, cfginclude, cfg, out, options)
        return subprocess.check_call(cmd, shell=True)

    def unpack(self, fwfile, unpacker, cfgpath):
        TARGETS_FILE = "targets.json"
        try:
            if os.path.exists(os.path.join(fwfile, "unpacked")) and os.path.exists(os.path.join(fwfile, TARGETS_FILE)):
                self._tmpdir = fwfile
                self._unpackdir = os.path.join(self._tmpdir, "unpacked")
                print("{0}: is a directory containing an 'unpacked' path, skipping".format(fwfile))
                cmd = "cat {0}".format(os.path.join(fwfile, TARGETS_FILE))
                self._unpacked = True
            else:
                self._tmpdir = tempfile.mkdtemp()
                self._unpackdir = os.path.join(self._tmpdir, "unpacked")
                os.mkdir(self._unpackdir)
                cmd = "{0} {1} {2}".format(unpacker, fwfile, cfgpath)
            res = subprocess.check_output(cmd, shell=True, cwd=self._unpackdir)
            targets = json.loads(res.decode('utf-8'))
            with open(os.path.join(self._tmpdir, TARGETS_FILE), "w") as fp:
                fp.write(res.decode('utf-8'))
            return targets
        except Exception as e:
            print("Exception: {0}".format(e))
            print("can't load targets from output of '{0}' check your script".format(unpacker))
            return None

    def del_tmp_dir(self):
        if not self._unpacked:
            cmd = "rm -rf {0}".format(self._tmpdir)
            return subprocess.check_call(cmd, shell=True)

    def files_by_ext_stat(self, data):
        allext = {}
        for i in data["files"]:
            fn, ext = os.path.splitext(i["name"])
            if ext in allext:
                count, ext = allext[ext]
                allext[ext] = count + 1, ext
            else:
                allext[ext] = (1, ext)
        return (len(data["files"]), allext)

    def analyze_filetree(self, filetreefile):
        with open(filetreefile) as fp:
            data = json.load(fp)
        num_files, stats = self.files_by_ext_stat(data)
        out = {}
        percent = num_files / 100
        # only keep entries with count > 1% and files that have an extension
        for i in stats:
            (count, ext) = stats[i]
            if count > percent and ext != "":
                out[ext] = (count, ext)

        return {
            "total_files": num_files,
            "file_extension_stats_inclusion_if_more_than": percent,
            "file_extension_stats": sorted(out.values(), reverse=True)
        }

    # check result and run post analysis
    def check_result(self, result):
        with open(result) as read_file:
            data = json.load(read_file)

        if "offenders" in data:
            status = False
        else:
            status = True

        CURRENT_FILE_TREE = "current_file_tree_path"

        if CURRENT_FILE_TREE in data:
            if os.path.isfile(data[CURRENT_FILE_TREE]):
                data["file_tree_analysis"] = self.analyze_filetree(data[CURRENT_FILE_TREE])

        return (status, json.dumps(data, sort_keys=True, indent=2))


def hashfile(fpath):
    m = hashlib.sha256()
    with open(fpath, "rb") as f:
        while True:
            data = f.read(65535)
            if not data:
                break
            m.update(data)
    return m.hexdigest()


# make report from image reports
def make_report(fwfile, data):
    report = {}
    status = True
    for key in data:
        img_status, img_report  = out[key]
        if status != False:
            status = img_status
        report[key] = json.loads(img_report)
    report["firmware"] = fwfile
    if os.path.isfile(fwfile):
        report["firmware_digest"] = hashfile(fwfile)
    report["status"] = status
    return json.dumps(report, sort_keys=True, indent=2)


if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument("--fw", action="store", required=True, help="path to firmware file OR path to unpacked firmware")
    parser.add_argument("--unpacker", action="store", required=True, help="path to unpacking script")
    parser.add_argument("--cfg-path", action="store", required=True, help="path to directory containing config files")
    parser.add_argument("--cfg-include-path", action="store", help="path to config include files")
    parser.add_argument("--report", action="store", help="report file")
    parser.add_argument("--keep-unpacked", action="store_true", help="keep unpacked data")
    parser.add_argument("--fwanalyzer-bin", action="store", default="fwanalyzer", help="path to fwanalyzer binary")
    parser.add_argument("--fwanalyzer-options", action="store", default="", help="options passed to fwanalyzer")
    args = parser.parse_args()

    fw = os.path.realpath(args.fw)
    cfg = os.path.realpath(args.cfg_path)

    check = CheckFirmware(args.fwanalyzer_bin)
    targets = check.unpack(fw, os.path.realpath(args.unpacker), cfg)
    print("using tmp directory: {0}".format(check.get_tmp_dir()))
    if not targets:
        print("no targets defined")
        sys.exit(1)

    # target file system images, a fwanalyzer config file is required for each of those
    for tgt in targets:
        cfg_file_name = "{0}.toml".format(tgt)
        if not os.path.isfile(os.path.join(args.cfg_path, cfg_file_name)):
            print("skipped, config file '{0}' for '{1}' does not exist\n".format(
                os.path.join(args.cfg_path, cfg_file_name), targets[tgt]))
            sys.exit(0)
        else:
            print("using config file '{0}' for '{1}'".format(
                os.path.join(args.cfg_path, cfg_file_name), targets[tgt]))

    out = {}
    all_checks_ok = True
    for tgt in targets:
        cfg_file_name = "{0}.toml".format(tgt)
        out_file_name = "{0}_out.json".format(tgt)
        check.run_fwanalyzer_fs(os.path.join(check.get_tmp_dir(), targets[tgt]),
            os.path.join(cfg, cfg_file_name), args.cfg_include_path, out_file_name,
            options=args.fwanalyzer_options)
        ok, data = check.check_result(out_file_name)
        out[tgt] = ok, data
        if not ok:
            all_checks_ok = False

    if args.keep_unpacked:
        print("unpacked: {0}\n".format(check.get_tmp_dir()))
    else:
        check.del_tmp_dir()

    report = make_report(args.fw, out)
    if args.report != None:
        with open(args.report, "w+") as fp:
            fp.write(report)
        print("report written to '{0}'".format(args.report))
    else:
        print(report)

    if not all_checks_ok:
        print("Firmware Analysis: checks failed")
        sys.exit(1)
    else:
        print("Firmware Analysis: checks passed")
        sys.exit(0)
