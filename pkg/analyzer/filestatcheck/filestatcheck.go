/*
Copyright 2019-present, Cruise LLC

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package filestatcheck

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/BurntSushi/toml"

	"github.com/cruise-automation/fwanalyzer/pkg/analyzer"
	"github.com/cruise-automation/fwanalyzer/pkg/capability"
	"github.com/cruise-automation/fwanalyzer/pkg/fsparser"
)

type fileexistType struct {
	AllowEmpty        bool
	Mode              string
	Uid               int
	Gid               int
	SELinuxLabel      string
	Capabilities      []string
	Desc              string
	InformationalOnly bool
}

type fileExistListType struct {
	FileStatCheck map[string]fileexistType
}

type fileExistType struct {
	files fileExistListType
	a     analyzer.AnalyzerType
}

func New(config string, a analyzer.AnalyzerType) *fileExistType {
	cfg := fileExistType{a: a}

	md, err := toml.Decode(config, &cfg.files)
	if err != nil {
		panic("can't read config data: " + err.Error())
	}

	for fn, item := range cfg.files.FileStatCheck {
		if !md.IsDefined("FileStatCheck", fn, "Uid") {
			item.Uid = -1
			cfg.files.FileStatCheck[fn] = item
		}

		if !md.IsDefined("FileStatCheck", fn, "Gid") {
			item.Gid = -1
			cfg.files.FileStatCheck[fn] = item
		}
	}

	return &cfg
}

func (state *fileExistType) Start() {}

func (state *fileExistType) CheckFile(fi *fsparser.FileInfo, filepath string) error {
	return nil
}

func (state *fileExistType) Name() string {
	return "FileStatCheck"
}

func (state *fileExistType) Finalize() string {
	for fn, item := range state.files.FileStatCheck {
		fi, err := state.a.GetFileInfo(fn)
		if err != nil {
			state.a.AddOffender(fn, fmt.Sprintf("file does not exist"))
		} else {
			checkMode := false
			var mode uint64
			if item.Mode != "" {
				checkMode = true
				mode, _ = strconv.ParseUint(item.Mode, 8, 0)
			}
			if !item.AllowEmpty && fi.Size == 0 {
				if item.InformationalOnly {
					state.a.AddInformational(fn, fmt.Sprintf("File State Check failed: size: %d AllowEmpyt=false : %s", fi.Size, item.Desc))
				} else {
					state.a.AddOffender(fn, fmt.Sprintf("File State Check failed: size: %d AllowEmpyt=false : %s", fi.Size, item.Desc))
				}
			}
			if checkMode && fi.Mode != mode {
				if item.InformationalOnly {
					state.a.AddInformational(fn, fmt.Sprintf("File State Check failed: mode found %o should be %s : %s", fi.Mode, item.Mode, item.Desc))
				} else {
					state.a.AddOffender(fn, fmt.Sprintf("File State Check failed: mode found %o should be %s : %s", fi.Mode, item.Mode, item.Desc))
				}
			}
			if item.Gid >= 0 && fi.Gid != item.Gid {
				if item.InformationalOnly {
					state.a.AddInformational(fn, fmt.Sprintf("File State Check failed: group found %d should be %d : %s", fi.Gid, item.Gid, item.Desc))
				} else {
					state.a.AddOffender(fn, fmt.Sprintf("File State Check failed: group found %d should be %d : %s", fi.Gid, item.Gid, item.Desc))
				}
			}
			if item.Uid >= 0 && fi.Uid != item.Uid {
				if item.InformationalOnly {
					state.a.AddInformational(fn, fmt.Sprintf("File State Check failed: owner found %d should be %d : %s", fi.Uid, item.Uid, item.Desc))
				} else {
					state.a.AddOffender(fn, fmt.Sprintf("File State Check failed: owner found %d should be %d : %s", fi.Uid, item.Uid, item.Desc))
				}
			}
			if item.SELinuxLabel != "" && !strings.EqualFold(item.SELinuxLabel, fi.SELinuxLabel) {
				if item.InformationalOnly {
					state.a.AddInformational(fn, fmt.Sprintf("File State Check failed: selinux label found = %s should be = %s : %s", fi.SELinuxLabel, item.SELinuxLabel, item.Desc))
				} else {
					state.a.AddOffender(fn, fmt.Sprintf("File State Check failed: selinux label found = %s should be = %s : %s", fi.SELinuxLabel, item.SELinuxLabel, item.Desc))
				}
			}

			if len(item.Capabilities) > 0 {
				if !capability.CapsEqual(item.Capabilities, fi.Capabilities) {
					if item.InformationalOnly {
						state.a.AddInformational(fn, fmt.Sprintf("Capabilities found: %s expected: %s", fi.Capabilities, item.Capabilities))
					} else {
						state.a.AddOffender(fn, fmt.Sprintf("Capabilities found: %s expected: %s", fi.Capabilities, item.Capabilities))
					}
				}
			}
		}
	}
	return ""
}
