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

package filecontent

import (
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/bmatcuk/doublestar"

	"github.com/cruise-automation/fwanalyzer/pkg/analyzer"
	"github.com/cruise-automation/fwanalyzer/pkg/fsparser"
	"github.com/cruise-automation/fwanalyzer/pkg/util"
)

type contentType struct {
	File              string   // filename
	InformationalOnly bool     // put result into Informational (not Offenders)
	RegEx             string   // regex to match against the file content
	RegExLineByLine   bool     // match regex line by line vs whole file
	Match             bool     // define if regex should match or not
	Digest            string   // used for SHA256 matching
	Script            string   // used for script execution
	ScriptOptions     []string // options for script execution
	Json              string   // used for json field matching
	Desc              string   // description
	name              string   // name of this check (need to be unique)
	checked           bool     // if this file was checked or not
}

type fileContentType struct {
	files map[string][]contentType
	a     analyzer.AnalyzerType
}

func validateItem(item contentType) bool {
	if item.RegEx != "" && (item.Digest == "" && item.Script == "" && item.Json == "") {
		return true
	}
	if item.Digest != "" && (item.RegEx == "" && item.Script == "" && item.Json == "") {
		return true
	}
	if item.Script != "" && (item.RegEx == "" && item.Digest == "" && item.Json == "") {
		return true
	}
	if item.Json != "" && (item.RegEx == "" && item.Digest == "" && item.Script == "") {
		return true
	}
	return false
}

func New(config string, a analyzer.AnalyzerType, MatchInvert bool) *fileContentType {
	type fileContentListType struct {
		FileContent map[string]contentType
	}
	cfg := fileContentType{a: a, files: make(map[string][]contentType)}

	var fcc fileContentListType
	_, err := toml.Decode(config, &fcc)
	if err != nil {
		panic("can't read config data: " + err.Error())
	}

	// convert text name based map to filename based map with an array of checks
	for name, item := range fcc.FileContent {
		if !validateItem(item) {
			a.AddOffender(name, "FileContent: check must include one of Digest, RegEx, Json, or Script")
			continue
		}
		var items []contentType
		if _, ok := cfg.files[item.File]; ok {
			items = cfg.files[item.File]
		}
		item.name = name
		if MatchInvert {
			item.Match = !item.Match
		}
		items = append(items, item)
		item.File = path.Clean(item.File)
		cfg.files[item.File] = items
	}

	return &cfg
}

func (state *fileContentType) Start() {}

func (state *fileContentType) Finalize() string {
	for fn, items := range state.files {
		for _, item := range items {
			if !item.checked {
				state.a.AddOffender(fn, fmt.Sprintf("FileContent: file %s not found", fn))
			}
		}
	}
	return ""
}

func (state *fileContentType) Name() string {
	return "FileContent"
}

func regexCompile(rx string) (*regexp.Regexp, error) {
	reg, err := regexp.CompilePOSIX(rx)
	if err != nil {
		reg, err = regexp.Compile(rx)
	}
	return reg, err
}

func (state *fileContentType) canCheckFile(fi *fsparser.FileInfo, fn string, item contentType) bool {
	if !fi.IsFile() {
		state.a.AddOffender(fn, fmt.Sprintf("FileContent: '%s' file is NOT a file : %s", item.name, item.Desc))
		return false
	}
	if fi.IsLink() {
		state.a.AddOffender(fn, fmt.Sprintf("FileContent: '%s' file is a link (check actual file) : %s", item.name, item.Desc))
		return false
	}
	return true
}

func (state *fileContentType) CheckFile(fi *fsparser.FileInfo, filepath string) error {
	fn := path.Join(filepath, fi.Name)
	if _, ok := state.files[fn]; !ok {
		return nil
	}

	items := state.files[fn]

	for n, item := range items {
		items[n].checked = true
		//fmt.Printf("name: %s file: %s (%s)\n", item.name, item.File, fn)
		if item.RegEx != "" {
			if !state.canCheckFile(fi, fn, item) {
				continue
			}
			reg, err := regexCompile(item.RegEx)
			if err != nil {
				state.a.AddOffender(fn, fmt.Sprintf("FileContent: regex compile error: %s : %s : %s", item.RegEx, item.name, item.Desc))
				continue
			}

			tmpfn, err := state.a.FileGet(fn)
			// this should never happen since this function is called for every existing file
			if err != nil {
				state.a.AddOffender(fn, fmt.Sprintf("FileContent: error reading file: %s", err))
				continue
			}
			fdata, _ := ioutil.ReadFile(tmpfn)
			err = state.a.RemoveFile(tmpfn)
			if err != nil {
				panic("RemoveFile failed")
			}
			if item.RegExLineByLine {
				for _, line := range strings.Split(strings.TrimSuffix(string(fdata), "\n"), "\n") {
					if reg.MatchString(line) == item.Match {
						if item.InformationalOnly {
							state.a.AddInformational(fn, fmt.Sprintf("RegEx check failed, for: %s : %s : line: %s", item.name, item.Desc, line))
						} else {
							state.a.AddOffender(fn, fmt.Sprintf("RegEx check failed, for: %s : %s : line: %s", item.name, item.Desc, line))
						}
					}
				}
			} else {
				if reg.Match(fdata) == item.Match {
					if item.InformationalOnly {
						state.a.AddInformational(fn, fmt.Sprintf("RegEx check failed, for: %s : %s", item.name, item.Desc))
					} else {
						state.a.AddOffender(fn, fmt.Sprintf("RegEx check failed, for: %s : %s", item.name, item.Desc))
					}
				}
			}
			continue
		}

		if item.Digest != "" {
			if !state.canCheckFile(fi, fn, item) {
				continue
			}
			digestRaw, err := state.a.FileGetSha256(fn)
			if err != nil {
				return err
			}
			digest := hex.EncodeToString(digestRaw)
			saved, _ := hex.DecodeString(item.Digest)
			savedStr := hex.EncodeToString(saved)
			if digest != savedStr {
				if item.InformationalOnly {
					state.a.AddInformational(fn, fmt.Sprintf("Digest (sha256) did not match found = %s should be = %s. %s : %s ", digest, savedStr, item.name, item.Desc))
				} else {
					state.a.AddOffender(fn, fmt.Sprintf("Digest (sha256) did not match found = %s should be = %s. %s : %s ", digest, savedStr, item.name, item.Desc))
				}
			}
			continue
		}

		if item.Script != "" {
			cbd := callbackDataType{state, item.Script, item.ScriptOptions, item.InformationalOnly}
			if fi.IsDir() {
				state.a.CheckAllFilesWithPath(checkFileScript, &cbd, fn)
			} else {
				if !state.canCheckFile(fi, fn, item) {
					continue
				}
				checkFileScript(fi, filepath, &cbd)
			}
		}

		if item.Json != "" {
			if !state.canCheckFile(fi, fn, item) {
				continue
			}
			tmpfn, err := state.a.FileGet(fn)
			if err != nil {
				state.a.AddOffender(fn, fmt.Sprintf("FileContent: error getting file: %s", err))
				continue
			}
			fdata, err := ioutil.ReadFile(tmpfn)
			if err != nil {
				state.a.AddOffender(fn, fmt.Sprintf("FileContent: error reading file: %s", err))
				continue
			}
			err = state.a.RemoveFile(tmpfn)
			if err != nil {
				panic("RemoveFile failed")
			}

			field := strings.SplitAfterN(item.Json, ":", 2)
			if len(field) != 2 {
				state.a.AddOffender(fn, fmt.Sprintf("FileContent: error Json config bad = %s, %s, %s", item.Json, item.name, item.Desc))
				continue
			}

			// remove ":" so we just have the value we want to check
			field[0] = strings.Replace(field[0], ":", "", 1)

			fieldData, err := util.XtractJsonField(fdata, strings.Split(field[0], "."))
			if err != nil {
				state.a.AddOffender(fn, fmt.Sprintf("FileContent: error Json bad field = %s, %s, %s", field[0], item.name, item.Desc))
				continue
			}
			if fieldData != field[1] {
				if item.InformationalOnly {
					state.a.AddInformational(fn, fmt.Sprintf("Json field %s = %s did not match = %s, %s, %s", field[0], fieldData, field[1], item.name, item.Desc))
				} else {
					state.a.AddOffender(fn, fmt.Sprintf("Json field %s = %s did not match = %s, %s, %s", field[0], fieldData, field[1], item.name, item.Desc))
				}
			}
		}
	}
	return nil
}

type callbackDataType struct {
	state             *fileContentType
	script            string
	scriptOptions     []string
	informationalOnly bool
}

/*
 * Extract file and run script passing the file name as the argument to the script.
 * Only regular files that are not empty are processed, script is for checking content.
 * The script output is used to indicate an issue, the output is saved in the offender record.
 *
 * The first element in scriptOptions (from the callback data) defines a path match string.
 * This allows to specify a pattern the filename has to match. Files with names that do not match will
 * not be analyzed by the script. This is to speed up execution time since files have to be extracted
 * to analyze them with the external script.
 *
 * The following elements in scriptOptions will be passed to the script as cmd line arguments.
 *
 * The script is run with the following parameters:
 * script.sh <filename> <filename in filesystem> <uid> <gid> <mode> <selinux label - can be empty> -- <ScriptOptions[1]> <ScriptOptions[2]>
 */
func checkFileScript(fi *fsparser.FileInfo, fullpath string, cbData analyzer.AllFilesCallbackData) {
	cbd := cbData.(*callbackDataType)

	fullname := path.Join(fullpath, fi.Name)

	// skip/ignore anything but normal files
	if !fi.IsFile() || fi.IsLink() {
		return
	}

	if len(cbd.scriptOptions) >= 1 {
		m, err := doublestar.Match(cbd.scriptOptions[0], fi.Name)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Match error: %s\n", err)
			return
		}
		// file name didn't match the specifications in scriptOptions[0]
		if !m {
			return
		}
	}

	fname, _ := cbd.state.a.FileGet(fullname)
	args := []string{fname,
		fullname,
		fmt.Sprintf("%d", fi.Uid),
		fmt.Sprintf("%d", fi.Gid),
		fmt.Sprintf("%o", fi.Mode),
		fi.SELinuxLabel,
	}
	if len(cbd.scriptOptions) >= 2 {
		args = append(args, "--")
		args = append(args, cbd.scriptOptions[1:]...)
	}

	out, err := exec.Command(cbd.script, args...).CombinedOutput()
	if err != nil {
		cbd.state.a.AddOffender(fullname, fmt.Sprintf("script(%s) error=%s", cbd.script, err))
	}

	err = cbd.state.a.RemoveFile(fname)
	if err != nil {
		panic("removeFile failed")
	}

	if len(out) > 0 {
		if cbd.informationalOnly {
			cbd.state.a.AddInformational(fullname, string(out))
		} else {
			cbd.state.a.AddOffender(fullname, string(out))
		}
	}
}
