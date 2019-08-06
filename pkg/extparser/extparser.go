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

package extparser

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/cruise-automation/fwanalyzer/pkg/fsparser"
)

type Ext2Parser struct {
	fileinfoReg *regexp.Regexp
	selinux     bool
	imagepath   string
}

const (
	e2ToolsCp = "e2cp"
	e2ToolsLs = "e2ls"
)

func New(imagepath string, selinux bool) *Ext2Parser {
	parser := &Ext2Parser{
		// 365  120777     0     0        7 12-Jul-2018 10:15 true
		fileinfoReg: regexp.MustCompile(
			`^\s*(\d+)\s+(\d+)\s+(\d+)\s+(\d+)\s+(\d+)\s+(\d+-\w+-\d+)\s+(\d+:\d+)\s+(\S+)`),
		imagepath: imagepath,
		selinux:   false,
	}
	if selinux && seLinuxSupported() {
		parser.enableSeLinux()
	}
	return parser
}

func (e *Ext2Parser) ImageName() string {
	return e.imagepath
}

func (e *Ext2Parser) enableSeLinux() {
	// with selinux support (-Z)
	// 2600  100750     0  2000     1041   1-Jan-2009 03:00 init.environ.rc   u:object_r:rootfs:s0
	e.fileinfoReg = regexp.MustCompile(
		`^\s*(\d+)\s+(\d+)\s+(\d+)\s+(\d+)\s+(\d+)\s+(\d+-\w+-\d+)\s+(\d+:\d+)\s+(\S+)\s+(\S+)`)
	e.selinux = true
}

func (e *Ext2Parser) parseFileLine(line string) fsparser.FileInfo {
	res := e.fileinfoReg.FindAllStringSubmatch(line, -1)
	var fi fsparser.FileInfo
	size, _ := strconv.Atoi(res[0][5])
	fi.Size = int64(size)
	fi.Mode, _ = strconv.ParseUint(res[0][2], 8, 32)
	fi.Uid, _ = strconv.Atoi(res[0][3])
	fi.Gid, _ = strconv.Atoi(res[0][4])
	fi.Name = res[0][8]

	if e.selinux {
		fi.SELinuxLabel = res[0][9]
	} else {
		fi.SELinuxLabel = fsparser.SELinuxNoLabel
	}

	return fi
}

// ignoreDot=true: will filter out "." and ".." files from the directory listing
func (e *Ext2Parser) getDirList(dirpath string, ignoreDot bool) ([]fsparser.FileInfo, error) {
	arg := fmt.Sprintf("%s:%s", e.imagepath, dirpath)
	params := "-la"
	if e.selinux {
		params += "Z"
	}
	out, err := exec.Command(e2ToolsLs, params, arg).CombinedOutput()
	if err != nil {
		// do NOT print file not found error
		if !strings.EqualFold(string(out), "File not found by ext2_lookup") {
			fmt.Fprintln(os.Stderr, err)
		}
		return nil, err
	}
	var dir []fsparser.FileInfo
	lines := strings.Split(string(out), "\n")
	for _, fline := range lines {
		if len(fline) > 1 && fline[0] != '>' {
			fi := e.parseFileLine(fline)
			// filter: . and ..
			if !ignoreDot || (fi.Name != "." && fi.Name != "..") {
				dir = append(dir, fi)
			}
		}
	}
	return dir, nil
}

func (e *Ext2Parser) GetDirInfo(dirpath string) ([]fsparser.FileInfo, error) {
	dir, err := e.getDirList(dirpath, true)
	return dir, err
}

func (e *Ext2Parser) GetFileInfo(dirpath string) (fsparser.FileInfo, error) {
	var fi fsparser.FileInfo
	dir, err := e.getDirList(dirpath, false)
	if len(dir) == 1 {
		return dir[0], err
	}
	// GetFileInfo was called on a directory only return entry for "."
	for _, info := range dir {
		if info.Name == "." {
			info.Name = filepath.Base(dirpath)
			return info, nil
		}
	}
	return fi, fmt.Errorf("file not found: %s", dirpath)
}

func (e *Ext2Parser) CopyFile(filepath string, dstdir string) bool {
	src := fmt.Sprintf("%s:%s", e.imagepath, filepath)
	_, err := exec.Command(e2ToolsCp, src, dstdir).Output()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return false
	}
	return true
}

func (f *Ext2Parser) Supported() bool {
	_, err := exec.LookPath(e2ToolsLs)
	if err != nil {
		return false
	}
	_, err = exec.LookPath(e2ToolsCp)
	return err == nil
}

func seLinuxSupported() bool {
	out, _ := exec.Command(e2ToolsLs).CombinedOutput()
	// look for Z (selinux support) in "Usage: e2ls [-acDfilrtZ][-d dir] file"
	if strings.Contains(string(out), "Z") {
		return true
	}
	fmt.Fprintln(os.Stderr, "extparser: selinux not supported by your version of e2ls")
	return false
}
