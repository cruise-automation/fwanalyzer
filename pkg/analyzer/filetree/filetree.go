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

package filetree

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path"
	"strings"

	"github.com/BurntSushi/toml"

	"github.com/cruise-automation/fwanalyzer/pkg/analyzer"
	"github.com/cruise-automation/fwanalyzer/pkg/fsparser"
	"github.com/cruise-automation/fwanalyzer/pkg/util"
)

const (
	newFileTreeExt string = ".new"
)

type fileTreeConfig struct {
	OldTreeFilePath       string
	CheckPath             []string
	CheckPermsOwnerChange bool
	CheckFileSize         bool
	CheckFileDigest       bool
	SkipFileDigest        bool
}

type fileTreeType struct {
	config fileTreeConfig
	a      analyzer.AnalyzerType

	tree    map[string]fileInfoSaveType
	oldTree map[string]fileInfoSaveType
}

type fileInfoSaveType struct {
	fsparser.FileInfo
	Digest string `json:"digest"`
}
type imageInfoSaveType struct {
	ImageName   string             `json:"image_name"`
	ImageDigest string             `json:"image_digest"`
	Files       []fileInfoSaveType `json:"files"`
}

func New(config string, a analyzer.AnalyzerType, outputDirectory string) *fileTreeType {
	type ftcfg struct {
		FileTreeCheck fileTreeConfig
	}
	var conf ftcfg
	md, err := toml.Decode(config, &conf)
	if err != nil {
		panic("can't read config data: " + err.Error())
	}

	// if CheckPath is undefined set CheckPath to root
	if !md.IsDefined("FileTreeCheck", "CheckPath") {
		conf.FileTreeCheck.CheckPath = []string{"/"}
	}

	for i := range conf.FileTreeCheck.CheckPath {
		conf.FileTreeCheck.CheckPath[i] = util.CleanPathDir(conf.FileTreeCheck.CheckPath[i])
	}

	cfg := fileTreeType{config: conf.FileTreeCheck, a: a}

	// if an output directory is set concat the path of the old filetree
	if outputDirectory != "" && cfg.config.OldTreeFilePath != "" {
		cfg.config.OldTreeFilePath = path.Join(outputDirectory, cfg.config.OldTreeFilePath)
	}

	return &cfg
}

func inPath(checkPath string, cfgPath []string) bool {
	for _, p := range cfgPath {
		if strings.HasPrefix(checkPath, p) {
			return true
		}
	}
	return false
}

func (state *fileTreeType) Start() {
	state.tree = make(map[string]fileInfoSaveType)
}

func (state *fileTreeType) Name() string {
	return "FileTreeChecks"
}

func (tree *fileTreeType) readOldTree() error {
	data, err := ioutil.ReadFile(tree.config.OldTreeFilePath)
	if err != nil {
		return err
	}
	var oldTree imageInfoSaveType
	err = json.Unmarshal(data, &oldTree)
	if err != nil {
		return err
	}
	tree.oldTree = make(map[string]fileInfoSaveType)
	for _, fi := range oldTree.Files {
		tree.oldTree[fi.Name] = fi
	}
	return nil
}

func (tree *fileTreeType) saveTree() error {
	imageInfo := tree.a.ImageInfo()
	oldtree := imageInfoSaveType{
		ImageName:   imageInfo.ImageName,
		ImageDigest: imageInfo.ImageDigest,
	}

	for _, fi := range tree.tree {
		oldtree.Files = append(oldtree.Files, fi)
	}

	jdata, err := json.Marshal(oldtree)
	if err != nil {
		return err
	}
	// make json look pretty
	var prettyJson bytes.Buffer
	err = json.Indent(&prettyJson, jdata, "", "\t")
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(tree.config.OldTreeFilePath+newFileTreeExt, prettyJson.Bytes(), 0644)
	if err != nil {
		return err
	}
	return nil
}

func (state *fileTreeType) CheckFile(fi *fsparser.FileInfo, filepath string) error {
	if state.config.OldTreeFilePath == "" {
		return nil
	}

	fn := path.Join(filepath, fi.Name)

	digest := "0"
	if fi.IsFile() && !state.config.SkipFileDigest {
		digestRaw, err := state.a.FileGetSha256(fn)
		if err != nil {
			return err
		}
		digest = hex.EncodeToString(digestRaw)
	}

	state.tree[fn] = fileInfoSaveType{
		fsparser.FileInfo{
			Name:         fn,
			Size:         fi.Size,
			Uid:          fi.Uid,
			Gid:          fi.Gid,
			Mode:         fi.Mode,
			SELinuxLabel: fi.SELinuxLabel,
		},
		digest,
	}

	return nil
}

func (state *fileTreeType) Finalize() string {
	if state.config.OldTreeFilePath == "" {
		return ""
	}

	var added []fileInfoSaveType
	var removed []fileInfoSaveType
	var changed []string

	_ = state.readOldTree()

	// find modified files
	for filepath, fi := range state.oldTree {
		// skip files if not in configured path
		if !inPath(filepath, state.config.CheckPath) {
			continue
		}
		_, ok := state.tree[filepath]
		if !ok {
			removed = append(removed, fi)
		} else {
			oFi := fi
			cFi := state.tree[filepath]
			if oFi.Mode != cFi.Mode ||
				oFi.Uid != cFi.Uid ||
				oFi.Gid != cFi.Gid ||
				oFi.SELinuxLabel != cFi.SELinuxLabel ||
				((oFi.Size != cFi.Size) && state.config.CheckFileSize) ||
				((oFi.Digest != cFi.Digest) && state.config.CheckFileDigest) {
				changed = append(changed, filepath)
			}
		}
	}

	// find new files
	for filepath, fi := range state.tree {
		// skip files if not in configured path
		if !inPath(filepath, state.config.CheckPath) {
			continue
		}
		_, ok := state.oldTree[filepath]
		if !ok {
			added = append(added, fi)
		}
	}

	treeUpdated := false
	if len(added) > 0 || len(removed) > 0 || (len(changed) > 0 && state.config.CheckPermsOwnerChange) {
		err := state.saveTree()
		if err != nil {
			panic("saveTree failed")
		}
		treeUpdated = true
	}

	for _, fi := range added {
		fileInfoStr := fiToString(fi, true) //a.config.GlobalConfig.FsTypeOptions == "selinux")
		state.a.AddInformational(fi.Name, fmt.Sprintf("CheckFileTree: new file: %s", fileInfoStr))
	}
	for _, fi := range removed {
		fileInfoStr := fiToString(fi, true) //a.config.GlobalConfig.FsTypeOptions == "selinux")
		state.a.AddInformational(fi.Name, fmt.Sprintf("CheckFileTree: file removed: %s", fileInfoStr))
	}
	if state.config.CheckPermsOwnerChange {
		for _, filepath := range changed {
			fileInfoStrOld := fiToString(state.oldTree[filepath], true) //state.config..FsTypeOptions == "selinux")
			fileInfoStrCur := fiToString(state.tree[filepath], true)    //a.config.GlobalConfig.FsTypeOptions == "selinux")
			state.a.AddInformational(state.tree[filepath].Name,
				fmt.Sprintf("CheckFileTree: file perms/owner/size/digest changed from: %s to: %s", fileInfoStrOld, fileInfoStrCur))
		}
	}

	if state.config.OldTreeFilePath != "" {
		type reportData struct {
			OldFileTreePath     string `json:"old_file_tree_path"`
			CurrentFileTreePath string `json:"current_file_tree_path,omitempty"`
		}
		newPath := ""
		if treeUpdated {
			newPath = state.config.OldTreeFilePath + newFileTreeExt
		}

		data := reportData{state.config.OldTreeFilePath, newPath}
		jdata, _ := json.Marshal(&data)
		return string(jdata)
	}

	return ""
}

// provide fileinfo as a human readable string
func fiToString(fi fileInfoSaveType, selinux bool) string {
	if selinux {
		return fmt.Sprintf("%o %d:%d %d %s SELinux label: %s", fi.Mode, fi.Uid, fi.Gid, fi.Size, fi.Digest, fi.SELinuxLabel)
	} else {
		return fmt.Sprintf("%o %d:%d %d %s", fi.Mode, fi.Uid, fi.Gid, fi.Size, fi.Digest)
	}
}
