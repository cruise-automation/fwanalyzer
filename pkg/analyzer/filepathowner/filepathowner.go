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

package filepathowner

import (
	"fmt"
	"path"

	"github.com/BurntSushi/toml"

	"github.com/cruise-automation/fwanalyzer/pkg/analyzer"
	"github.com/cruise-automation/fwanalyzer/pkg/fsparser"
)

type filePathOwner struct {
	Uid int
	Gid int
}

type filePathOwenrList struct {
	FilePathOwner map[string]filePathOwner
}

type fileownerpathType struct {
	files filePathOwenrList
	a     analyzer.AnalyzerType
}

func New(config string, a analyzer.AnalyzerType) *fileownerpathType {
	cfg := fileownerpathType{a: a}

	_, err := toml.Decode(config, &cfg.files)
	if err != nil {
		panic("can't read config data: " + err.Error())
	}

	return &cfg
}

func (state *fileownerpathType) Start() {}
func (state *fileownerpathType) CheckFile(fi *fsparser.FileInfo, filepath string) error {
	return nil
}

func (state *fileownerpathType) Name() string {
	return "FilePathOwner"
}

type cbDataCheckOwnerPath struct {
	a   analyzer.AnalyzerType
	fop filePathOwner
}

func (state *fileownerpathType) Finalize() string {
	for fn, item := range state.files.FilePathOwner {
		filelist := cbDataCheckOwnerPath{a: state.a, fop: item}
		df, err := state.a.GetFileInfo(fn)
		if err != nil {
			state.a.AddOffender(fn, fmt.Sprintf("FilePathOwner, directory not found: %s", fn))
			continue
		}
		// check the directory itself
		cbCheckOwnerPath(&df, fn, &filelist)
		// check anything within the directory
		state.a.CheckAllFilesWithPath(cbCheckOwnerPath, &filelist, fn)
	}

	return ""
}

// check that every file within a given directory is owned by the given UID and GID
func cbCheckOwnerPath(fi *fsparser.FileInfo, fullpath string, data analyzer.AllFilesCallbackData) {
	var filelist *cbDataCheckOwnerPath = data.(*cbDataCheckOwnerPath)

	ppath := fullpath
	if len(fi.Name) > 0 {
		ppath = path.Join(ppath, fi.Name)
	}

	if fi.Uid != filelist.fop.Uid {
		filelist.a.AddOffender(ppath, fmt.Sprintf("FilePathOwner Uid not allowed, Uid = %d should be = %d", fi.Uid, filelist.fop.Uid))
	}
	if fi.Gid != filelist.fop.Gid {
		filelist.a.AddOffender(ppath, fmt.Sprintf("FilePathOwner Gid not allowed, Gid = %d should be = %d", fi.Gid, filelist.fop.Gid))
	}
}
