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

package dataextract

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/BurntSushi/toml"

	"github.com/cruise-automation/fwanalyzer/pkg/analyzer"
	"github.com/cruise-automation/fwanalyzer/pkg/fsparser"
	"github.com/cruise-automation/fwanalyzer/pkg/util"
)

type dataType struct {
	File          string
	Script        string
	ScriptOptions []string // options for script execution
	RegEx         string
	Json          string
	Desc          string
	Name          string // the name can be set directly otherwise the key will be used
}

type dataExtractType struct {
	config map[string][]dataType
	a      analyzer.AnalyzerType
}

func New(config string, a analyzer.AnalyzerType) *dataExtractType {
	type dataExtractListType struct {
		DataExtract map[string]dataType
	}
	cfg := dataExtractType{a: a, config: make(map[string][]dataType)}

	var dec dataExtractListType
	_, err := toml.Decode(config, &dec)
	if err != nil {
		panic("can't read config data: " + err.Error())
	}

	// convert name based map to filename based map with an array of dataType
	for name, item := range dec.DataExtract {
		var items []dataType
		if _, ok := cfg.config[item.File]; ok {
			items = cfg.config[item.File]
		}
		if item.Name == "" {
			// if the key ends with __[0-9] remove the suffix and use it as name
			if name[len(name)-1] >= '0' && name[len(name)-1] <= '9' && strings.HasSuffix(name[:len(name)-1], "__") {
				item.Name = name[:len(name)-3]
			} else {
				item.Name = name
			}
		}
		items = append(items, item)
		item.File = path.Clean(item.File)
		cfg.config[item.File] = items
	}

	return &cfg
}

func (state *dataExtractType) Start() {}
func (state *dataExtractType) Finalize() string {
	return ""
}

func (state *dataExtractType) Name() string {
	return "DataExtract"
}

func (state *dataExtractType) CheckFile(fi *fsparser.FileInfo, filepath string) error {
	if !fi.IsFile() {
		return nil
	}

	fn := path.Join(filepath, fi.Name)
	if _, ok := state.config[fn]; !ok {
		return nil
	}

	items := state.config[fn]

	// we record if the specific Name was already added with a non error value
	nameFilled := make(map[string]bool)

	for _, item := range items {
		// Name already set?
		if _, ok := nameFilled[item.Name]; ok {
			continue
		}

		if fi.IsLink() {
			state.a.AddData(item.Name, fmt.Sprintf("DataExtract ERROR: file is Link (extract data from actual file): %s : %s",
				item.Name, item.Desc))
			continue
		}

		if item.RegEx != "" {
			reg, err := regexp.Compile(item.RegEx)
			if err != nil {
				state.a.AddData(item.Name, fmt.Sprintf("DataExtract ERROR: regex compile error: %s : %s %s",
					item.RegEx, item.Name, item.Desc))
				continue
			}

			tmpfn, err := state.a.FileGet(fn)
			if err != nil {
				state.a.AddData(item.Name, fmt.Sprintf("DataExtract ERROR: file read error, file get: %s : %s : %s",
					err, item.Name, item.Desc))
				continue
			}
			fdata, err := ioutil.ReadFile(tmpfn)
			if err != nil {
				state.a.AddData(item.Name, fmt.Sprintf("DataExtract ERROR: file read error, file read: %s : %s : %s",
					err, item.Name, item.Desc))
				continue
			}
			_ = state.a.RemoveFile(tmpfn)
			res := reg.FindAllStringSubmatch(string(fdata), -1)
			if len(res) < 1 {
				state.a.AddData(item.Name, fmt.Sprintf("DataExtract ERROR: regex match error, regex: %s : %s : %s",
					item.RegEx, item.Name, item.Desc))
			} else {
				// only one match
				if len(res) == 1 && len(res[0]) == 2 {
					state.a.AddData(item.Name, res[0][1])
					nameFilled[item.Name] = true
				} else if len(res) > 1 {
					// multiple matches
					data := []string{}
					for _, i := range res {
						if len(i) == 2 {
							data = append(data, i[1])
						}
					}
					// convert to JSON arrary
					jdata, _ := json.Marshal(data)
					state.a.AddData(item.Name, string(jdata))
					nameFilled[item.Name] = true
				} else {
					state.a.AddData(item.Name, fmt.Sprintf("DataExtract ERROR: regex match error : %s : %s",
						item.Name, item.Desc))
				}
			}
		}

		if item.Script != "" {
			out, err := runScriptOnFile(state.a, item.Script, item.ScriptOptions, fi, fn)
			if err != nil {
				state.a.AddData(item.Name, fmt.Sprintf("DataExtract ERROR: script error: %s : %s : %s",
					err, item.Name, item.Desc))
			} else {
				state.a.AddData(item.Name, out)
				nameFilled[item.Name] = true
			}
		}

		if item.Json != "" {
			tmpfn, err := state.a.FileGet(fn)
			if err != nil {
				state.a.AddData(item.Name, fmt.Sprintf("DataExtract ERROR: file read error, file get: %s : %s : %s",
					err, item.Name, item.Desc))
				continue
			}
			fdata, err := ioutil.ReadFile(tmpfn)
			if err != nil {
				state.a.AddData(item.Name, fmt.Sprintf("DataExtract ERROR: file read error, file read: %s : %s : %s",
					err, item.Name, item.Desc))
				continue
			}
			_ = state.a.RemoveFile(tmpfn)

			out, err := util.XtractJsonField(fdata, strings.Split(item.Json, "."))
			if err != nil {
				state.a.AddData(item.Name, fmt.Sprintf("DataExtract ERROR: JSON decode error: %s : %s : %s",
					err, item.Name, item.Desc))
				continue
			}
			state.a.AddData(item.Name, out)
			nameFilled[item.Name] = true
		}
	}
	return nil
}

// runScriptOnFile runs the provided script with the following parameters:
// <filename> <filename in filesystem> <uid> <gid> <mode> <selinux label - can be empty> -- scriptOptions[0] scriptOptions[1]
func runScriptOnFile(a analyzer.AnalyzerType, script string, scriptOptions []string, fi *fsparser.FileInfo, fpath string) (string, error) {
	fname, err := a.FileGet(fpath)
	if err != nil {
		return "", err
	}
	options := []string{fname, filepath.Base(fpath), fmt.Sprintf("%d", fi.Uid), fmt.Sprintf("%d", fi.Gid),
		fmt.Sprintf("%o", fi.Mode), fi.SELinuxLabel}
	if len(scriptOptions) > 0 {
		options = append(options, "--")
		options = append(options, scriptOptions...)
	}
	out, err := exec.Command(script, options...).Output()
	_ = a.RemoveFile(fname)

	return string(out), err
}
