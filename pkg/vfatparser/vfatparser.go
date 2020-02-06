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

package vfatparser

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

type mDirReg struct {
	rx     *regexp.Regexp
	hasExt bool
}

type VFatParser struct {
	mDirRegex []mDirReg
	imagepath string
}

const (
	vFatLsCmd string = "mdir"
	vFatCpCmd string = "mcopy"
)

func New(imagepath string) *VFatParser {
	var regs []mDirReg
	// BZIMAGE        5853744 2018-11-12  10:04  bzImage
	regs = append(regs, mDirReg{regexp.MustCompile(`^([~\w]+)\s+(\d+)\s\d+-\d+-\d+\s+\d+:\d+\s+(\w+)$`), false})
	// BZIMAGE  SIG       287 2018-11-12  10:04  bzImage.sig
	regs = append(regs, mDirReg{regexp.MustCompile(`^([~\w]+)\s+(\w+)\s+(\d+)\s\d+-\d+-\d+\s+\d+:\d+\s+(.+).*`), true})
	// EFI          <DIR>     2018-11-12  10:04
	regs = append(regs, mDirReg{regexp.MustCompile(`^([~\.\w]+)\s+<DIR>.*`), false})
	// startup  nsh        12 2018-11-12  10:04
	regs = append(regs, mDirReg{regexp.MustCompile(`^([~\w]+)\s+(\w+)\s+(\d+).*`), true})
	// grubenv           1024 2018-11-12  10:04
	regs = append(regs, mDirReg{regexp.MustCompile(`^([~\w]+)\s+(\d+).*`), false})

	parser := &VFatParser{
		mDirRegex: regs,
		imagepath: imagepath,
	}
	// configure mtools to skip size checks on VFAT images
	os.Setenv("MTOOLS_SKIP_CHECK", "1")
	return parser
}

func (f *VFatParser) ImageName() string {
	return f.imagepath
}

func (f *VFatParser) parseFileLine(line string) (fsparser.FileInfo, error) {
	var fi fsparser.FileInfo
	for _, reg := range f.mDirRegex {
		res := reg.rx.FindAllStringSubmatch(line, -1)
		if res != nil {
			size := 0
			if len(res[0]) == 2 {
				fi.Mode = fsparser.S_IFDIR
			} else {
				fi.Mode = fsparser.S_IFREG
				if reg.hasExt {
					size, _ = strconv.Atoi(res[0][3])
				} else {
					size, _ = strconv.Atoi(res[0][2])
				}
			}
			fi.Mode |= fsparser.S_IRWXU | fsparser.S_IRWXG | fsparser.S_IRWXO
			fi.Size = int64(size)
			fi.Uid = 0
			fi.Gid = 0
			fi.SELinuxLabel = fsparser.SELinuxNoLabel
			fi.Name = res[0][1]
			if reg.hasExt {
				fi.Name = fmt.Sprintf("%s.%s", res[0][1], res[0][2])
			}
			// use long name
			if (!reg.hasExt && len(res[0]) > 3) || (reg.hasExt && len(res[0]) > 4) {
				fi.Name = res[0][len(res[0])-1]
			}
			return fi, nil
		}
	}
	return fi, fmt.Errorf("not a file/dir")
}

func (f *VFatParser) getDirList(dirpath string, ignoreDot bool) ([]fsparser.FileInfo, error) {
	var dir []fsparser.FileInfo
	out, err := exec.Command(vFatLsCmd, "-i", f.imagepath, dirpath).CombinedOutput()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return nil, err
	}
	lines := strings.Split(string(out), "\n")
	for _, fline := range lines {
		if len(fline) > 1 {
			fi, err := f.parseFileLine(fline)
			if err == nil {
				// filter: . and ..
				if !ignoreDot || (fi.Name != "." && fi.Name != "..") {
					dir = append(dir, fi)
				}
			}
		}
	}
	return dir, nil
}

func (f *VFatParser) GetDirInfo(dirpath string) ([]fsparser.FileInfo, error) {
	if dirpath == "" {
		dirpath = "/"
	}
	return f.getDirList(dirpath, true)
}

func (f *VFatParser) GetFileInfo(dirpath string) (fsparser.FileInfo, error) {
	// return fake entry for root (/)
	if dirpath == "/" {
		return fsparser.FileInfo{Name: "/", Mode: fsparser.S_IFDIR}, nil
	}

	var fifake fsparser.FileInfo
	dir, err := f.getDirList(dirpath, false)
	if err != nil {
		return fifake, err
	}

	// GetFileInfo was called on non directory
	if len(dir) == 1 {
		return dir[0], nil
	}

	for _, info := range dir {
		if info.Name == "." {
			info.Name = filepath.Base(dirpath)
			return info, nil
		}
	}

	return fifake, fmt.Errorf("file not found: %s", dirpath)
}

func (f *VFatParser) CopyFile(filepath string, dstdir string) bool {
	src := fmt.Sprintf("::%s", filepath)
	_, err := exec.Command(vFatCpCmd, "-bni", f.imagepath, src, dstdir).Output()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return false
	}
	return true
}

func (f *VFatParser) Supported() bool {
	_, err := exec.LookPath(vFatCpCmd)
	if err != nil {
		return false
	}
	_, err = exec.LookPath(vFatLsCmd)
	return err == nil
}
