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

package dirparser

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"syscall"

	"github.com/cruise-automation/fwanalyzer/pkg/capability"
	"github.com/cruise-automation/fwanalyzer/pkg/fsparser"
)

const (
	cpCli string = "cp"
)

type DirParser struct {
	imagepath string
}

func New(imagepath string) *DirParser {
	var global DirParser
	global.imagepath = imagepath
	return &global
}

func (dir *DirParser) GetDirInfo(dirpath string) ([]fsparser.FileInfo, error) {
	files := make([]fsparser.FileInfo, 0)
	filepath := path.Join(dir.imagepath, dirpath)
	fp, err := os.Open(filepath)
	if err != nil {
		return files, err
	}
	defer fp.Close()
	names, err := fp.Readdirnames(0)
	if err != nil {
		return files, err
	}
	for _, fname := range names {
		fi, err := dir.GetFileInfo(path.Join(dirpath, fname))
		fi.Name = fname
		if err != nil {
			return files, err
		}
		files = append(files, fi)
	}
	return files, nil
}

func (dir *DirParser) GetFileInfo(dirpath string) (fsparser.FileInfo, error) {
	var fi fsparser.FileInfo
	fpath := path.Join(dir.imagepath, dirpath)
	var fileStat syscall.Stat_t
	err := syscall.Lstat(fpath, &fileStat)
	if err != nil {
		return fi, err
	}
	fi.Name = filepath.Base(dirpath)
	fi.Mode = uint64(fileStat.Mode)
	fi.Uid = int(fileStat.Uid)
	fi.Gid = int(fileStat.Gid)
	fi.SELinuxLabel = fsparser.SELinuxNoLabel
	fi.Size = fileStat.Size

	capsBytes := make([]byte, capability.CapByteSizeMax)
	capsSize, _ := syscall.Getxattr(fpath, "security.capability", capsBytes)
	// ignore err since we only care about the returned size
	if capsSize > 0 {
		fi.Capabilities, err = capability.New(capsBytes)
		if err != nil {
			fmt.Println(err)
		}
	}

	if fi.IsLink() {
		fi.LinkTarget, err = os.Readlink(fpath)
		if err != nil {
			return fi, err
		}
	}
	return fi, nil
}

// copy (extract) file out of the FS into dest dir
func (dir *DirParser) CopyFile(filepath string, dstdir string) bool {
	_, err := dir.GetFileInfo(filepath)
	if err != nil {
		return false
	}
	err = exec.Command(cpCli, "-a", path.Join(dir.imagepath, filepath), dstdir).Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s -a %s %s: failed", cpCli, path.Join(dir.imagepath, filepath), dstdir)
		return false
	}
	return true
}

func (dir *DirParser) ImageName() string {
	return dir.imagepath
}

func (f *DirParser) Supported() bool {
	_, err := exec.LookPath(cpCli)
	return err == nil
}
