/*
Copyright 2019 GM Cruise LLC

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

package filecmp

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"syscall"

	"github.com/BurntSushi/toml"

	"github.com/cruise-automation/fwanalyzer/pkg/analyzer"
	"github.com/cruise-automation/fwanalyzer/pkg/fsparser"
)

type cmpType struct {
	File              string // filename
	OldFilePath       string
	Script            string
	ScriptOptions     string
	InformationalOnly bool   // put result into Informational (not Offenders)
	name              string // name of this check (need to be unique)

}

type fileCmpType struct {
	files map[string][]cmpType
	a     analyzer.AnalyzerType
}

func New(config string, a analyzer.AnalyzerType, fileDirectory string) *fileCmpType {
	type fileCmpListType struct {
		FileCmp map[string]cmpType
	}
	cfg := fileCmpType{a: a, files: make(map[string][]cmpType)}

	var fcc fileCmpListType
	_, err := toml.Decode(config, &fcc)
	if err != nil {
		panic("can't read config data: " + err.Error())
	}

	// convert text name based map to filename based map with an array of checks
	for name, item := range fcc.FileCmp {
		// make sure required options are set
		if item.OldFilePath == "" || item.Script == "" {
			continue
		}
		var items []cmpType
		if _, ok := cfg.files[item.File]; ok {
			items = cfg.files[item.File]
		}

		if fileDirectory != "" {
			item.OldFilePath = path.Join(fileDirectory, item.OldFilePath)
		}

		item.name = name
		item.File = path.Clean(item.File)
		items = append(items, item)
		cfg.files[item.File] = items
	}

	return &cfg
}

func (state *fileCmpType) Start() {}

func (state *fileCmpType) Finalize() string {
	return ""
}

func (state *fileCmpType) Name() string {
	return "FileCmp"
}

func fileExists(filePath string) error {
	var fileState syscall.Stat_t
	return syscall.Lstat(filePath, &fileState)
}

func copyFile(out string, in string) error {
	src, err := os.Open(in)
	if err != nil {
		return err
	}
	defer src.Close()
	dst, err := os.Create(out)
	if err != nil {
		return err
	}
	defer dst.Close()
	_, err = io.Copy(dst, src)
	return err
}

func makeTmpFromOld(filePath string) (string, error) {
	tmpfile, err := ioutil.TempFile("", "")
	if err != nil {
		return "", err
	}
	defer tmpfile.Close()
	src, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer src.Close()
	_, err = io.Copy(tmpfile, src)
	return tmpfile.Name(), err
}

func (state *fileCmpType) CheckFile(fi *fsparser.FileInfo, filepath string) error {
	if !fi.IsFile() {
		return nil
	}

	fn := path.Join(filepath, fi.Name)
	if _, ok := state.files[fn]; !ok {
		return nil
	}

	for _, item := range state.files[fn] {
		tmpfn, err := state.a.FileGet(fn)
		if err != nil {
			state.a.AddOffender(fn, fmt.Sprintf("FileCmp: error getting file: %s", err))
			continue
		}

		// we don't have a saved file so save it now and skip this check
		if fileExists(item.OldFilePath) != nil {
			err := copyFile(item.OldFilePath+".new", tmpfn)
			if err != nil {
				state.a.AddOffender(fn, fmt.Sprintf("FileCmp: error saving file: %s", err))
				continue
			}
			state.a.AddInformational(fn, "FileCmp: saved file for next run")
			continue
		}

		oldTmp, err := makeTmpFromOld(item.OldFilePath)
		if err != nil {
			state.a.AddOffender(fn, fmt.Sprintf("FileCmp: error getting old file: %s", err))
			continue
		}
		args := []string{fi.Name, oldTmp, tmpfn}
		if len(item.ScriptOptions) > 0 {
			args = append(args, "--")
			args = append(args, item.ScriptOptions)
		}

		out, err := exec.Command(item.Script, args...).CombinedOutput()
		if err != nil {
			state.a.AddOffender(path.Join(filepath, fi.Name), fmt.Sprintf("script(%s) error=%s", item.Script, err))
		}

		err = state.a.RemoveFile(tmpfn)
		if err != nil {
			panic("removeFile failed")
		}
		err = state.a.RemoveFile(oldTmp)
		if err != nil {
			panic("removeFile failed")
		}

		if len(out) > 0 {
			if item.InformationalOnly {
				state.a.AddInformational(path.Join(filepath, fi.Name), string(out))
			} else {
				state.a.AddOffender(path.Join(filepath, fi.Name), string(out))
			}
		}
	}

	return nil
}
