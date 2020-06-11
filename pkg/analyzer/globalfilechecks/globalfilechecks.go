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

package globalfilechecks

import (
	"fmt"
	"path"

	"github.com/BurntSushi/toml"
	"github.com/bmatcuk/doublestar"

	"github.com/cruise-automation/fwanalyzer/pkg/analyzer"
	"github.com/cruise-automation/fwanalyzer/pkg/fsparser"
)

type filePermsConfigType struct {
	Suid                            bool
	SuidAllowedList                 map[string]bool
	WorldWrite                      bool
	SELinuxLabel                    bool
	Uids                            map[int]bool
	Gids                            map[int]bool
	BadFiles                        map[string]bool
	BadFilesInformationalOnly       bool
	FlagCapabilityInformationalOnly bool
}

type filePermsType struct {
	config *filePermsConfigType
	a      analyzer.AnalyzerType
}

func New(config string, a analyzer.AnalyzerType) *filePermsType {
	type filePermsConfig struct {
		Suid                            bool
		SuidWhiteList                   []string // keep for backward compatibility
		SuidAllowedList                 []string
		WorldWrite                      bool
		SELinuxLabel                    bool
		Uids                            []int
		Gids                            []int
		BadFiles                        []string
		BadFilesInformationalOnly       bool
		FlagCapabilityInformationalOnly bool
	}
	type fpc struct {
		GlobalFileChecks filePermsConfig
	}
	var conf fpc
	_, err := toml.Decode(config, &conf)
	if err != nil {
		panic("can't read config data: " + err.Error())
	}

	configuration := filePermsConfigType{
		Suid:                            conf.GlobalFileChecks.Suid,
		WorldWrite:                      conf.GlobalFileChecks.WorldWrite,
		SELinuxLabel:                    conf.GlobalFileChecks.SELinuxLabel,
		BadFilesInformationalOnly:       conf.GlobalFileChecks.BadFilesInformationalOnly,
		FlagCapabilityInformationalOnly: conf.GlobalFileChecks.FlagCapabilityInformationalOnly,
	}
	configuration.SuidAllowedList = make(map[string]bool)
	for _, alfn := range conf.GlobalFileChecks.SuidAllowedList {
		configuration.SuidAllowedList[path.Clean(alfn)] = true
	}
	// keep for backward compatibility
	for _, wlfn := range conf.GlobalFileChecks.SuidWhiteList {
		configuration.SuidAllowedList[path.Clean(wlfn)] = true
	}
	configuration.Uids = make(map[int]bool)
	for _, uid := range conf.GlobalFileChecks.Uids {
		configuration.Uids[uid] = true
	}
	configuration.Gids = make(map[int]bool)
	for _, gid := range conf.GlobalFileChecks.Gids {
		configuration.Gids[gid] = true
	}

	configuration.BadFiles = make(map[string]bool)
	for _, bf := range conf.GlobalFileChecks.BadFiles {
		configuration.BadFiles[path.Clean(bf)] = true
	}

	cfg := filePermsType{&configuration, a}

	return &cfg
}

func (state *filePermsType) Start() {}
func (state *filePermsType) Finalize() string {
	return ""
}

func (state *filePermsType) Name() string {
	return "GlobalFileChecks"
}

func (state *filePermsType) CheckFile(fi *fsparser.FileInfo, fpath string) error {
	if state.config.Suid {
		if fi.IsSUid() || fi.IsSGid() {
			if _, ok := state.config.SuidAllowedList[path.Join(fpath, fi.Name)]; !ok {
				state.a.AddOffender(path.Join(fpath, fi.Name), "File is SUID, not allowed")
			}
		}
	}
	if state.config.WorldWrite {
		if fi.IsWorldWrite() && !fi.IsLink() && !fi.IsDir() {
			state.a.AddOffender(path.Join(fpath, fi.Name), "File is WorldWriteable, not allowed")
		}
	}
	if state.config.SELinuxLabel {
		if fi.SELinuxLabel == fsparser.SELinuxNoLabel {
			state.a.AddOffender(path.Join(fpath, fi.Name), "File does not have SELinux label")
		}
	}

	if len(state.config.Uids) > 0 {
		if _, ok := state.config.Uids[fi.Uid]; !ok {
			state.a.AddOffender(path.Join(fpath, fi.Name), fmt.Sprintf("File Uid not allowed, Uid = %d", fi.Uid))
		}
	}

	if len(state.config.Gids) > 0 {
		if _, ok := state.config.Gids[fi.Gid]; !ok {
			state.a.AddOffender(path.Join(fpath, fi.Name), fmt.Sprintf("File Gid not allowed, Gid = %d", fi.Gid))
		}
	}

	if state.config.FlagCapabilityInformationalOnly {
		if len(fi.Capabilities) > 0 {
			state.a.AddInformational(path.Join(fpath, fi.Name), fmt.Sprintf("Capabilities found: %s", fi.Capabilities))
		}
	}

	for item := range state.config.BadFiles {
		fullpath := fi.Name
		// match the fullpath if it starts with "/"
		if item[0] == '/' {
			fullpath = path.Join(fpath, fi.Name)
		}
		m, err := doublestar.Match(item, fullpath)
		if err != nil {
			return err
		}
		if m {
			msg := "File not allowed"
			if item != fullpath {
				msg = fmt.Sprintf("File not allowed for pattern: %s", item)
			}

			if state.config.BadFilesInformationalOnly {
				state.a.AddInformational(path.Join(fpath, fi.Name), msg)
			} else {
				state.a.AddOffender(path.Join(fpath, fi.Name), msg)
			}
		}
	}

	return nil
}
