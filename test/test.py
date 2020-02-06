#!/usr/bin/env python


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

    if data.get("data", {}).get("extract_test") != "test extract":
        SetError("extract test")

    if "offenders" not in data:
        SetError("offenders")
    else:
        if not "/dir2/file21" in data["offenders"]:
            SetError("dir2/file21")

        if not "/dir2/file22" in data["offenders"]:
            SetError("dir2/file22")

        if not "File is WorldWriteable, not allowed" in data["offenders"]["/world"]:
            SetError("WorldWriteable")

        if not "File is SUID, not allowed" in data["offenders"]["/dir2/file21"]:
            SetError("SUID")

        if not "DirContent: File file22 not allowed in directory /dir2" in data["offenders"]["/dir2/file22"]:
            SetError("DirContent")

        if not "nofile" in data["offenders"]:
            SetError("DirContent")

        if not "test script" in data["offenders"]["/file2"]:
            SetError("file2")

        if not "Digest (sha256) did not match found = 44c77e41961f354f515e4081b12619fdb15829660acaa5d7438c66fc3d326df3 should be = 8b15095ed1af38d5e383af1c4eadc5ae73cab03964142eb54cb0477ccd6a8dd4. ver needs to be specific :  " in data["offenders"]["/ver"]:
            SetError("ver digest")

        if "File State Check failed: group found 1002 should be 0 : this needs to be this way" in data["offenders"]["/dir2/file22"]:
            SetError("FileStatCheck shouldn't default to uid/guid 0")

        if not "File not allowed for pattern: *1" in data["offenders"]["/file1"]:
            SetError("file1 not allowed")

        if not "File State Check failed: size: 0 AllowEmpyt=false : this needs to be this way" in data["offenders"]["/file1"]:
            SetError("file1 exists but size 0")

        if not "elf_x8664 is not stripped" in data["offenders"]["/bin/elf_x8664"]:
            SetError("script failed")

    if not "informational" in data:
        SetError("informational")
    else:
        if not "/file1" in data["informational"]:
            SetError("/file1")
        else:
            if not "changed" in data["informational"]["/file1"][0]:
                SetError("file1 not changed")
        if not "/date1" in data["informational"]:
            SetError("/date1")
        else:
            if not "changed" in data["informational"]["/date1"][0]:
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
