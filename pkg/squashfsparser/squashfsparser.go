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

package squashfsparser

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"os/user"
	"path"
	"regexp"
	"strconv"
	"strings"

	"github.com/cruise-automation/fwanalyzer/pkg/capability"
	"github.com/cruise-automation/fwanalyzer/pkg/fsparser"
)

const (
	unsquashfsCmd = "unsquashfs"
	cpCmd         = "cp"
)

// SquashFSParser parses SquashFS filesystem images.
type SquashFSParser struct {
	fileLineRegex *regexp.Regexp
	imagepath     string
	files         map[string][]fsparser.FileInfo
	securityInfo  bool
}

func uidForUsername(username string) (int, error) {
	// First check to see if it's an int. If not, look it up by name.
	uid, err := strconv.Atoi(username)
	if err == nil {
		return uid, nil
	}
	u, err := user.Lookup(username)
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(u.Uid)
}

func gidForGroup(group string) (int, error) {
	// First check to see if it's an int. If not, look it up by name.
	gid, err := strconv.Atoi(group)
	if err == nil {
		return gid, nil
	}
	g, err := user.LookupGroup(group)
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(g.Gid)
}

// From table[] in https://github.com/plougher/squashfs-tools/blob/master/squashfs-tools/unsquashfs.c
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

func parseMode(mode string) (uint64, error) {
	var m uint64
	if len(mode) != 10 {
		return 0, fmt.Errorf("parseMode: invalid mode string %s", mode)
	}
	for _, f := range modeFlags {
		if mode[f.pos] == f.chr {
			m |= f.val
		}
	}
	return m, nil
}

func getExtractFile(dirpath string) (string, error) {
	extractFile, err := ioutil.TempFile("", "squashfsparser")
	if err != nil {
		return "", err
	}
	_, err = extractFile.Write([]byte(dirpath))
	if err != nil {
		return "", err
	}
	err = extractFile.Close()
	if err != nil {
		return "", err
	}
	return extractFile.Name(), nil
}

func (s *SquashFSParser) enableSecurityInfo() {
	// drwxr-xr-x administrator/administrator 66 2019-04-08 18:49 squashfs-root	- -
	s.fileLineRegex = regexp.MustCompile(`^([A-Za-z-]+)\s+([\-\.\w]+|\d+)/([\-\.\w]+|\d+)\s+(\d+)\s+(\d+-\d+-\d+)\s+(\d+:\d+)\s+([\S ]+)\t(\S+)\s+(\S)`)
	s.securityInfo = true
}

// New returns a new SquashFSParser instance for the given image file.
func New(imagepath string, securityInfo bool) *SquashFSParser {
	parser := &SquashFSParser{
		// drwxr-xr-x administrator/administrator 66 2019-04-08 18:49 squashfs-root
		fileLineRegex: regexp.MustCompile(`^([A-Za-z-]+)\s+([\-\.\w]+|\d+)/([\-\.\w]+|\d+)\s+(\d+)\s+(\d+-\d+-\d+)\s+(\d+:\d+)\s+(.*)$`),
		imagepath:     imagepath,
		securityInfo:  false,
	}

	if securityInfo && securityInfoSupported() {
		parser.enableSecurityInfo()
	}

	return parser
}

func normalizePath(filepath string) (dir string, name string) {
	// Ensure directory and file names are consistent, with no relative parts
	// or trailing slash on directory names.
	dir, name = path.Split(path.Clean(filepath))
	dir = path.Clean(dir)
	return
}

func (s *SquashFSParser) parseFileLine(line string) (string, fsparser.FileInfo, error) {
	// TODO(jlarimer): add support for reading xattrs. unsquashfs can read
	// and write xattrs, but it doesn't display them when just listing files.
	var fi fsparser.FileInfo
	dirpath := ""
	res := s.fileLineRegex.FindStringSubmatch(line)
	if res == nil {
		return dirpath, fi, fmt.Errorf("Can't match line %s\n", line)
	}
	var err error
	fi.Mode, err = parseMode(res[1])
	if err != nil {
		return dirpath, fi, err
	}
	// unsquashfs converts the uid/gid to a username/group on this system, so
	// we need to convert it back to the numeric values.
	fi.Uid, err = uidForUsername(res[2])
	if err != nil {
		return dirpath, fi, err
	}
	fi.Gid, err = gidForGroup(res[3])
	if err != nil {
		return dirpath, fi, err
	}
	fi.Size, err = strconv.ParseInt(res[4], 10, 64)
	if err != nil {
		return dirpath, fi, err
	}
	// links show up with a name like "./dir2/file3 -> file1"
	if fi.Mode&fsparser.S_IFLNK == fsparser.S_IFLNK {
		parts := strings.Split(res[7], " -> ")
		dirpath, fi.Name = normalizePath(parts[0])
		fi.LinkTarget = parts[1]
	} else {
		dirpath, fi.Name = normalizePath(res[7])
	}

	if s.securityInfo {
		if res[8] != "-" {
			fi.Capabilities, _ = capability.New(res[8])
		}
		fi.SELinuxLabel = res[9]
	}

	return dirpath, fi, nil
}

func (s *SquashFSParser) loadFileList() error {
	if s.files != nil {
		return nil
	}
	s.files = make(map[string][]fsparser.FileInfo)

	// we want to use -lln (numeric output) but that is only available in 4.4 and later
	args := []string{"-d", "", "-lls", s.imagepath}
	if s.securityInfo {
		// -llS is only available in our patched version
		args = append([]string{"-llS"}, args...)
	}

	out, err := exec.Command(unsquashfsCmd, args...).CombinedOutput()
	if err != nil {
		fmt.Fprintf(os.Stderr, "getDirList: %s", err)
		return err
	}
	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		path, fi, err := s.parseFileLine(line)
		if err == nil {
			dirfiles := s.files[path]
			dirfiles = append(dirfiles, fi)
			s.files[path] = dirfiles
		}
	}
	return nil
}

// GetDirInfo returns information on the specified directory.
func (s *SquashFSParser) GetDirInfo(dirpath string) ([]fsparser.FileInfo, error) {
	if err := s.loadFileList(); err != nil {
		return nil, err
	}

	return s.files[path.Clean(dirpath)], nil
}

// GetFileInfo returns information on the specified file.
func (s *SquashFSParser) GetFileInfo(filepath string) (fsparser.FileInfo, error) {
	if err := s.loadFileList(); err != nil {
		return fsparser.FileInfo{}, err
	}

	dirpath, name := normalizePath(filepath)
	// the root is stored as "."
	if dirpath == "/" && name == "" {
		dirpath = "."
		name = "."
	}
	dir := s.files[dirpath]
	for _, fi := range dir {
		if fi.Name == name {
			return fi, nil
		}
	}
	return fsparser.FileInfo{}, fmt.Errorf("Can't find file %s", filepath)
}

// CopyFile copies the specified file to the specified destination.
func (s *SquashFSParser) CopyFile(filepath string, dstdir string) bool {
	// The list of files/directories to extract needs to be in a file...
	extractFile, err := getExtractFile(filepath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create temporary file: %v\n", err)
		return false
	}
	defer os.Remove(extractFile)

	// The -d argument to unsquashfs specifies a directory to unsquash to, but
	// the directory can't exist. It also extracts the full path. To fit the
	// semantics of CopyFile, we need to extract to a new temporary directly and
	// then copy the file to the specified destination.
	tmpdir, err := ioutil.TempDir("", "squashfsparser")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create temporary directory: %v\n", err)
		return false
	}
	defer os.RemoveAll(tmpdir)
	tmpdir = path.Join(tmpdir, "files")

	out, err := exec.Command(unsquashfsCmd, "-d", tmpdir, "-e", extractFile, s.imagepath).CombinedOutput()
	if err != nil {
		fmt.Fprintf(os.Stderr, "unsquashfs failed: %v: %s\n", err, out)
		return false
	}

	err = exec.Command(cpCmd, "-a", path.Join(tmpdir, filepath), dstdir).Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s -a %s %s: failed", cpCmd, path.Join(tmpdir, filepath), dstdir)
		return false
	}
	return true
}

// ImageName returns the name of the filesystem image.
func (s *SquashFSParser) ImageName() string {
	return s.imagepath
}

func (f *SquashFSParser) Supported() bool {
	_, err := exec.LookPath(unsquashfsCmd)
	if err != nil {
		return false
	}
	_, err = exec.LookPath(cpCmd)
	return err == nil
}

func securityInfoSupported() bool {
	out, _ := exec.Command(unsquashfsCmd).CombinedOutput()
	// look for -ll[S] (securityInfo support) in output
	if strings.Contains(string(out), "-ll[S]") {
		return true
	}
	fmt.Fprintln(os.Stderr, "squashfsparser: security info (selinux + capabilities) not supported by your version of unsquashfs")
	return false
}
