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

package ubifsparser

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strconv"
	"strings"

	"github.com/cruise-automation/fwanalyzer/pkg/fsparser"
)

type UbifsParser struct {
	fileinfoReg *regexp.Regexp
	fileLinkReg *regexp.Regexp
	imagepath   string
}

const (
	ubifsReaderCmd = "ubireader_list_files"
)

func New(imagepath string) *UbifsParser {
	parser := &UbifsParser{
		// 120777  1 0 0       0 Mar 13 08:53 tmp -> /var/tmp
		fileinfoReg: regexp.MustCompile(
			`^\s*(\d+)\s+(\d+)\s+(\d+)\s+(\d+)\s+(\d+)\s+(\S+\s+\d+)\s+(\d+:\d+)\s+(.+)$`),
		fileLinkReg: regexp.MustCompile(
			`(\S+)\s->\s(\S+)`),
		imagepath: imagepath,
	}

	return parser
}

func (e *UbifsParser) ImageName() string {
	return e.imagepath
}

func (e *UbifsParser) parseFileLine(line string) (fsparser.FileInfo, error) {
	res := e.fileinfoReg.FindAllStringSubmatch(line, -1)
	var fi fsparser.FileInfo
	if res == nil {
		return fi, fmt.Errorf("can't parse: %s", line)
	}
	size, _ := strconv.Atoi(res[0][5])
	fi.Size = int64(size)
	fi.Mode, _ = strconv.ParseUint(res[0][1], 8, 32)
	fi.Uid, _ = strconv.Atoi(res[0][3])
	fi.Gid, _ = strconv.Atoi(res[0][4])
	fi.Name = res[0][8]

	fi.SELinuxLabel = fsparser.SELinuxNoLabel

	// fill in linktarget
	if fi.IsLink() && strings.Contains(fi.Name, "->") {
		rlnk := e.fileLinkReg.FindAllStringSubmatch(fi.Name, -1)
		if rlnk == nil {
			return fsparser.FileInfo{}, fmt.Errorf("can't parse LinkTarget from %s", fi.Name)
		}
		fi.Name = rlnk[0][1]
		fi.LinkTarget = rlnk[0][2]
	}

	return fi, nil
}

func (e *UbifsParser) getDirList(dirpath string) ([]fsparser.FileInfo, error) {
	out, err := exec.Command(ubifsReaderCmd, "-P", dirpath, e.imagepath).CombinedOutput()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return nil, err
	}
	var dir []fsparser.FileInfo
	lines := strings.Split(string(out), "\n")
	for _, fline := range lines {
		if fline == "" {
			continue
		}
		fi, err := e.parseFileLine(fline)
		if err != nil {
			return nil, err
		}
		dir = append(dir, fi)
	}
	return dir, nil
}

func (e *UbifsParser) GetDirInfo(dirpath string) ([]fsparser.FileInfo, error) {
	dir, err := e.getDirList(dirpath)
	return dir, err
}

func (e *UbifsParser) GetFileInfo(dirpath string) (fsparser.FileInfo, error) {
	// return fake entry for root (/)
	if dirpath == "/" {
		return fsparser.FileInfo{Name: "/", Mode: fsparser.S_IFDIR}, nil
	}

	listpath := path.Dir(dirpath)
	listfile := path.Base(dirpath)
	var fi fsparser.FileInfo
	dir, err := e.getDirList(listpath)
	if err != nil {
		return fi, err
	}

	for _, info := range dir {
		if info.Name == listfile {
			return info, nil
		}
	}
	return fi, fmt.Errorf("file not found: %s", dirpath)
}

func (e *UbifsParser) CopyFile(filepath string, dstdir string) bool {
	err := exec.Command(ubifsReaderCmd, "--copy", filepath, "--copy-dest", dstdir, e.imagepath).Run()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return false
	}
	return true
}

func (f *UbifsParser) Supported() bool {
	_, err := exec.LookPath(ubifsReaderCmd)
	return err == nil
}
