#!/usr/bin/env python


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


import json
import sys
import os

error = False

def SetError(log):
    global error
    print log
    error = True

def test(cfgfile, e2toolspath=""):
    os.system(e2toolspath+" fwanalyzer -in test/test.img -cfg " + cfgfile + " >test/test_out.json 2>&1")

    with open("test/test_out.json") as read_file:
        try:
            data = json.load(read_file)
        except:
            data = {}

    if data.get("image_name") != "test/test.img":
        SetError("image_name")

    if "data" not in data:
        SetError("data")

    if data.get("data", {}).get("Version") != "1.2.3":
        SetError("Data Version")

    if "offenders" not in data:
        SetError("offenders")
    else:
        if "/dir2/file21" not in data["offenders"]:
            SetError("dir2/file21")

        if "/dir2/file22" not in data["offenders"]:
            SetError("dir2/file22")

        if data["offenders"]["/world"][0].find("WorldWriteable") == -1:
            SetError("WorldWriteable")

        if data["offenders"]["/dir2/file21"][0].find("File is SUID") == -1:
            SetError("SUID")

        if data["offenders"]["/dir2/file22"][0].find("DirContent") == -1:
            SetError("DirContent")

        if "nofile" not in data["offenders"]:
            SetError("DirContent")

        if data["offenders"]["/ver"][0].find("Digest") == -1:
            SetError("ver digest")

        if "File State Check failed: group found 1002 should be 0 : this needs to be this way" in data["offenders"]["/dir2/file22"]:
            SetError("FileStatCheck shouldn't default to uid/guid 0")

        if not "File not allowed for pattern: *1" in data["offenders"]["/file1"]:
            SetError("file1 not allowed")

        if not "File State Check failed: size: 0 AllowEmpyt=false : this needs to be this way" in data["offenders"]["/file1"]:
            SetError("file1 exists but size 0")

        if "elf_x8664 is not stripped" not in data["offenders"]["/bin/elf_x8664"]:
            SetError("script failed")

    if "informational" not in data:
        SetError("informational")
    else:
        if "/file1" not in data["informational"]:
            SetError("/file1")
        else:
            if "changed" not in data["informational"]["/file1"][0]:
                SetError("file1 not changed")
        if "/date1" not in data["informational"]:
            SetError("/date1")
        else:
            if "changed" not in data["informational"]["/date1"][0]:
                SetError("date1 not changed")

if __name__ == "__main__":
    test("test/test_cfg.toml")
    if error:
       os.system("cat test/test_out.json")
       sys.exit(error)

    # disable if your e2ls version does not support selinux (-Z) option
    test("test/test_cfg_selinux.toml")

    if error:
       os.system("cat test/test_out.json")
       sys.exit(error)
