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

package cpioparser

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strconv"
	"strings"

	"github.com/cruise-automation/fwanalyzer/pkg/fsparser"
)

const (
	cpioCmd         = "cpio"
	cpCmd           = "cp"
	MIN_LINE_LENGTH = 25
)

type CpioParser struct {
	fileInfoReg *regexp.Regexp
	devInfoReg  *regexp.Regexp
	fileLinkReg *regexp.Regexp
	imagepath   string
	files       map[string][]fsparser.FileInfo
	fixDirs     bool
}

func New(imagepath string, fixDirs bool) *CpioParser {
	parser := &CpioParser{
		//lrwxrwxrwx   1 0        0              19 Apr 24  2019 lib/libnss_dns.so.2 -> libnss_dns-2.18.so
		//-rwxrwxrwx   1 0        0              19 Apr 24 13:37 lib/lib.c
		fileInfoReg: regexp.MustCompile(
			`^([\w-]+)\s+(\d+)\s+(\d+)\s+(\d+)\s+(\d+)\s+(\w+\s+\d+\s+[\d:]+)\s+(.*)$`),
		// crw-r--r--   1 0        0          4,  64 Apr 24  2019 dev/ttyS0
		devInfoReg: regexp.MustCompile(
			`^([\w-]+)\s+(\d+)\s+(\d+)\s+(\d+)\s+(\d+),\s+(\d+)\s+(\w+\s+\d+\s+[\d:]+)\s+(.*)$`),
		fileLinkReg: regexp.MustCompile(`(\S+)\s->\s(\S+)`),
		imagepath:   imagepath,
		fixDirs:     fixDirs,
	}

	return parser
}

func (p *CpioParser) ImageName() string {
	return p.imagepath
}

var modeFlags = []struct {
	pos int
	chr byte
	val uint64
}{
	{0, '-', fsparser.S_IFREG},
	{0, 's', fsparser.S_IFSOCK},
	{0, 'l', fsparser.S_IFLNK},
	{0, 'b', fsparser.S_IFBLK},
	{0, 'd', fsparser.S_IFDIR},
	{0, 'c', fsparser.S_IFCHR},
	{0, 'p', fsparser.S_IFIFO},
	{1, 'r', fsparser.S_IRUSR},
	{2, 'w', fsparser.S_IWUSR},
	{3, 'x', fsparser.S_IXUSR},
	{3, 's', fsparser.S_IXUSR | fsparser.S_ISUID},
	{3, 'S', fsparser.S_ISUID},
	{4, 'r', fsparser.S_IRGRP},
	{5, 'w', fsparser.S_IWGRP},
	{6, 'x', fsparser.S_IXGRP},
	{6, 's', fsparser.S_IXGRP | fsparser.S_ISGID},
	{6, 'S', fsparser.S_ISGID},
	{7, 'r', fsparser.S_IROTH},
	{8, 'w', fsparser.S_IWOTH},
	{9, 'x', fsparser.S_IXOTH},
	{9, 't', fsparser.S_IXOTH | fsparser.S_ISVTX},
	{9, 'T', fsparser.S_ISVTX},
}

const (
	FILE_MODE_STR_LEN = 10 // such as "-rw-r--r--"
)

func parseMode(mode string) (uint64, error) {
	var m uint64
	if len(mode) != FILE_MODE_STR_LEN {
		return 0, fmt.Errorf("parseMode: invalid mode string %s", mode)
	}
	for _, f := range modeFlags {
		if mode[f.pos] == f.chr {
			m |= f.val
		}
	}
	return m, nil
}

// Ensure directory and file names are consistent, with no relative parts
// or trailing slash on directory names.
func normalizePath(filepath string) (dir string, name string) {
	dir, name = path.Split(path.Clean(filepath))
	dir = path.Clean(dir)
	return
}

const (
	NAME_IDX_NORMAL_FILE = 7
	NAME_IDX_DEVICE_FILE = 8
)

func (p *CpioParser) parseFileLine(line string) (string, fsparser.FileInfo, error) {
	reg := p.fileInfoReg
	nameIdx := NAME_IDX_NORMAL_FILE
	dirpath := ""
	if strings.HasPrefix(line, "b") || strings.HasPrefix(line, "c") {
		reg = p.devInfoReg
		nameIdx = NAME_IDX_DEVICE_FILE
	}
	res := reg.FindAllStringSubmatch(line, -1)
	var fi fsparser.FileInfo
	// only normal files have a size
	if nameIdx == NAME_IDX_NORMAL_FILE {
		size, _ := strconv.Atoi(res[0][5])
		fi.Size = int64(size)
	}
	fi.Mode, _ = parseMode(res[0][1])
	fi.Uid, _ = strconv.Atoi(res[0][3])
	fi.Gid, _ = strconv.Atoi(res[0][4])
	// cpio returns relative pathnames so add leading "/"
	fi.Name = "/" + res[0][nameIdx]

	// fill in linktarget
	if fi.IsLink() && strings.Contains(fi.Name, "->") {
		rlnk := p.fileLinkReg.FindAllStringSubmatch(fi.Name, -1)
		if rlnk == nil {
			return "", fsparser.FileInfo{}, fmt.Errorf("can't parse LinkTarget from %s", fi.Name)
		}
		fi.Name = rlnk[0][1]
		fi.LinkTarget = rlnk[0][2]
	}

	// handle root directory
	if fi.Name == "/." {
		dirpath = "."
		fi.Name = "."
	} else {
		dirpath, fi.Name = normalizePath(fi.Name)
	}

	return dirpath, fi, nil

}

// GetDirInfo returns information on the specified directory.
func (p *CpioParser) GetDirInfo(dirpath string) ([]fsparser.FileInfo, error) {
	if err := p.loadFileList(); err != nil {
		return nil, err
	}

	return p.files[path.Clean(dirpath)], nil
}

// GetFileInfo returns information on the specified file.
func (p *CpioParser) GetFileInfo(filepath string) (fsparser.FileInfo, error) {
	if err := p.loadFileList(); err != nil {
		return fsparser.FileInfo{}, err
	}

	dirpath, name := normalizePath(filepath)
	// the root is stored as "."
	if dirpath == "/" && name == "" {
		dirpath = "."
		name = "."
	}
	dir := p.files[dirpath]
	for _, fi := range dir {
		if fi.Name == name {
			return fi, nil
		}
	}
	return fsparser.FileInfo{}, fmt.Errorf("Can't find file %s", filepath)
}

func (p *CpioParser) loadFileList() error {
	if p.files != nil {
		return nil
	}

	out, err := exec.Command("sh", "-c", cpioCmd+" -tvn --quiet < "+p.imagepath).CombinedOutput()
	if err != nil {
		if err.Error() != errors.New("exit status 2").Error() {
			fmt.Fprintf(os.Stderr, "getDirList: >%s<", err)
			return err
		}
	}
	return p.loadFileListFromString(string(out))
}

func (p *CpioParser) loadFileListFromString(rawFileList string) error {
	p.files = make(map[string][]fsparser.FileInfo)

	lines := strings.Split(rawFileList, "\n")
	for _, line := range lines {
		if len(line) < MIN_LINE_LENGTH {
			continue
		}
		if strings.HasPrefix(line, "cpio") {
			continue
		}
		path, fi, err := p.parseFileLine(line)
		if err == nil {
			dirfiles := p.files[path]
			dirfiles = append(dirfiles, fi)
			p.files[path] = dirfiles

			if p.fixDirs {
				p.fixDir(path, fi.Name)
			}
		}
	}
	return nil
}

/*
 * With cpio it is possible that a file exists in a directory that does not have its own entry.
 *  e.g. "dev/tty6" exists in the cpio but there is no entry for "dev"
 * This function creates the missing directories in the internal structure.
 */
func (p *CpioParser) fixDir(dir string, name string) {
	if dir == "/" {
		return
	}
	basename := path.Base(dir)
	dirname := path.Dir(dir)

	// check that all dirname parts exist
	if strings.Contains(dirname, "/") {
		p.fixDir(dirname, basename)
	}

	dirExists := false
	for _, f := range p.files[dirname] {
		if f.Name == basename {
			dirExists = true
		}
	}
	if !dirExists {
		dirfiles := p.files[dirname]
		dirfiles = append(dirfiles, fsparser.FileInfo{Name: basename, Mode: 040755, Uid: 0, Gid: 0, Size: 0})
		p.files[dirname] = dirfiles
	}
}

// CopyFile copies the specified file to the specified destination.
func (p *CpioParser) CopyFile(filepath string, dstdir string) bool {
	out, err := exec.Command("sh", "-c", cpioCmd+" -i --to-stdout "+filepath[1:]+" < "+p.imagepath+" > "+dstdir).CombinedOutput()
	if err != nil {
		if err.Error() != errors.New("exit status 2").Error() {
			fmt.Fprintf(os.Stderr, "cpio failed: %v: %s\n", err, out)
			return false
		}
	}
	return true
}

func (p *CpioParser) Supported() bool {
	if _, err := exec.LookPath(cpioCmd); err != nil {
		return false
	}
	if _, err := exec.LookPath(cpCmd); err != nil {
		return false
	}
	return true
}
