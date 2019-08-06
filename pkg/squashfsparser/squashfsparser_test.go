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

package squashfsparser

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/cruise-automation/fwanalyzer/pkg/fsparser"
)

// This could be considered environment-specific...
func TestUidForUsername(t *testing.T) {
	uid, err := uidForUsername("root")
	if err != nil {
		t.Errorf("uidForUsername(\"root\") returned error: %v", err)
		return
	}
	if uid != 0 {
		t.Errorf("uidForUsername(\"root\") returned %d, should be 0", uid)
	}

	_, err = uidForUsername("asdfASDFxxx999")
	if err == nil {
		t.Errorf("uidForUsername(\"asdfASDFxxx\") did not return error")
		return
	}
}

// This could be considered environment-specific...
func TestGidForGroup(t *testing.T) {
	uid, err := gidForGroup("root")
	if err != nil {
		t.Errorf("gidForGroup(\"root\") returned error: %v", err)
		return
	}
	if uid != 0 {
		t.Errorf("gidForGroup(\"root\") returned %d, should be 0", uid)
	}

	_, err = gidForGroup("asdfASDFxxx999")
	if err == nil {
		t.Errorf("gidForGroup(\"asdfASDFxxx999\") did not return error")
	}
}

func TestParseMode(t *testing.T) {
	tests := []struct {
		mode   string
		result uint64
		err    bool
	}{
		{
			mode: "drwxr-xr-x",
			result: fsparser.S_IFDIR | fsparser.S_IRWXU | fsparser.S_IRGRP |
				fsparser.S_IXGRP | fsparser.S_IROTH | fsparser.S_IXOTH,
			err: false,
		},
		{
			mode: "-rw-r--r--",
			result: fsparser.S_IFREG | fsparser.S_IRUSR | fsparser.S_IWUSR |
				fsparser.S_IRGRP | fsparser.S_IROTH,
			err: false,
		},
		{
			mode: "lrwxrwxrwx",
			result: fsparser.S_IFLNK | fsparser.S_IRWXU | fsparser.S_IRWXG |
				fsparser.S_IRWXO,
			err: false,
		},
		{
			mode: "drwxrwxrwt",
			result: fsparser.S_IFDIR | fsparser.S_IRWXU | fsparser.S_IRWXG |
				fsparser.S_IRWXO | fsparser.S_ISVTX,
			err: false,
		},
		{
			// too short
			mode:   "blahblah",
			result: 0,
			err:    true,
		},
		{
			// too long
			mode:   "blahblahblah",
			result: 0,
			err:    true,
		},
	}

	for _, test := range tests {
		result, err := parseMode(test.mode)
		if err != nil && !test.err {
			t.Errorf("parseMode(\"%s\") returned error but shouldn't have: %s", test.mode, err)
			continue
		}
		if result != test.result {
			t.Errorf("parseMode(\"%s\") should be %#o, is %#o", test.mode, test.result, result)
		}
	}
}

func TestParseFileLine(t *testing.T) {
	tests := []struct {
		line    string
		dirpath string
		fi      fsparser.FileInfo
		err     bool
	}{
		{
			line:    "-rw-r--r-- root/root         32 2019-04-10 14:41 /Filey McFileFace",
			dirpath: "/",
			fi: fsparser.FileInfo{
				Size: 32,
				Mode: 0100644,
				Uid:  0,
				Gid:  0,
				Name: "Filey McFileFace",
			},
			err: false,
		},
		{
			line:    "lrwxrwxrwx 1010/2020         5 2019-04-10 14:36 /dir2/file3 -> file1",
			dirpath: "/dir2",
			fi: fsparser.FileInfo{
				Size:       5,
				Mode:       0120777,
				Uid:        1010,
				Gid:        2020,
				Name:       "file3",
				LinkTarget: "file1",
			},
			err: false,
		},
		{
			line:    "blah blah blah!",
			dirpath: "",
			fi:      fsparser.FileInfo{},
			err:     true,
		},
	}

	for _, test := range tests {
		dirpath, fi, err := parseFileLine(test.line)
		if err != nil && !test.err {
			t.Errorf("parseFileLine(\"%s\") returned error but shouldn't have: %s", test.line, err)
			continue
		}
		if dirpath != test.dirpath {
			t.Errorf("parseFileLine(\"%s\") dirpath got \"%s\", wanted \"%s\"", test.line, dirpath, test.dirpath)
		}
		if diff := cmp.Diff(fi, test.fi); diff != "" {
			t.Errorf("parseFileLine(\"%s\") result mismatch (-got, +want):\n%s", test.line, diff)
		}
	}
}

func TestImageName(t *testing.T) {
	testImage := "../../test/squashfs.img"
	f := New(testImage)

	imageName := f.ImageName()
	if imageName != testImage {
		t.Errorf("ImageName() returned %s, wanted %s", imageName, testImage)
	}
}

func TestDirInfoRoot(t *testing.T) {
	testImage := "../../test/squashfs.img"
	f := New(testImage)

	/*
		$ unsquashfs -d "" -ll test/squashfs.img
		Parallel unsquashfs: Using 8 processors
		5 inodes (4 blocks) to write

		drwxr-xr-x jlarimer/jlarimer        63 2019-04-11 08:06
		-rw-r--r-- root/jlarimer             0 2019-04-10 14:41 /Filey McFileFace
		drwxr-x--- 1007/1008                 3 2019-04-10 14:36 /dir1
		drwxr-xr-x jlarimer/jlarimer        69 2019-04-10 14:40 /dir2
		---------- jlarimer/jlarimer         7 2019-04-10 14:36 /dir2/file1
		-rwsr-xr-x jlarimer/jlarimer         5 2019-04-10 14:36 /dir2/file2
		lrwxrwxrwx jlarimer/jlarimer         5 2019-04-10 14:36 /dir2/file3 -> file1
		drwx------ 1005/1005                28 2019-04-10 14:40 /dir2/subdir2
		-rw-r--r-- jlarimer/jlarimer        20 2019-04-10 14:40 /dir2/subdir2/file4
	*/

	tests := map[string]map[string]fsparser.FileInfo{
		"/": {
			"Filey McFileFace": fsparser.FileInfo{
				Name: "Filey McFileFace",
				Mode: 0100644,
				Uid:  0,
				Gid:  1001,
				Size: 0,
			},
			"dir1": fsparser.FileInfo{
				Name: "dir1",
				Mode: 0040750,
				Uid:  1007,
				Gid:  1008,
				Size: 3,
			},
			"dir2": fsparser.FileInfo{
				Name: "dir2",
				Mode: 0040755,
				Uid:  1001,
				Gid:  1001,
				Size: 69,
			},
		},
		"/dir2": {
			"file1": fsparser.FileInfo{
				Name: "file1",
				Mode: 0100000,
				Uid:  1001,
				Gid:  1001,
				Size: 7,
			},
			"file2": fsparser.FileInfo{
				Name: "file2",
				Mode: 0104755,
				Uid:  1001,
				Gid:  1001,
				Size: 5,
			},
			"file3": fsparser.FileInfo{
				Name:       "file3",
				Mode:       0120777,
				Uid:        1001,
				Gid:        1001,
				Size:       5,
				LinkTarget: "file1",
			},
			"subdir2": fsparser.FileInfo{
				Name: "subdir2",
				Mode: 0040700,
				Uid:  1005,
				Gid:  1005,
				Size: 28,
			},
		},
		"/dir2/subdir2": {
			"file4": fsparser.FileInfo{
				Name: "file4",
				Mode: 0100644,
				Uid:  1001,
				Gid:  1001,
				Size: 20,
			},
		},
	}

	for _, testdir := range []string{"/", "/dir2", "/dir2/subdir2"} {
		dirtests := tests[testdir]
		dir, err := f.GetDirInfo(testdir)
		if err != nil {
			t.Errorf("GetDirInfo() returned error: %v", err)
			return
		}
		for _, fi := range dir {
			//fmt.Printf("Directory: %s, Name: %s, Size: %d, Mode: %o\n", testdir, fi.Name, fi.Size, fi.Mode)
			tfi, ok := dirtests[fi.Name]
			if !ok {
				t.Errorf("File \"%s\" not found in test map", fi.Name)
				continue
			}
			if diff := cmp.Diff(fi, tfi); diff != "" {
				t.Errorf("GetDirInfo() result mismatch for \"%s\" (-got, +want):\n%s", fi.Name, diff)
			}
			delete(dirtests, fi.Name)
		}
		for name := range dirtests {
			t.Errorf("File \"%s\" exists in test map but not in test filesystem", name)
		}
	}
}

func TestGetFileInfo(t *testing.T) {
	testImage := "../../test/squashfs.img"
	f := New(testImage)

	fi, err := f.GetFileInfo("/")
	if err != nil {
		t.Error(err)
	}
	if !fi.IsDir() {
		t.Errorf("/ should be dir")
	}

	fi, err = f.GetFileInfo("/dir2/file3")
	if err != nil {
		t.Errorf("GetDirInfo() returned error: %v", err)
		return
	}

	tfi := fsparser.FileInfo{
		Name:       "file3",
		Mode:       0120777,
		Uid:        1001,
		Gid:        1001,
		Size:       5,
		LinkTarget: "file1",
	}

	if diff := cmp.Diff(fi, tfi); diff != "" {
		t.Errorf("GetFileInfo() result mismatch (-got, +want):\n%s", diff)
	}
}

func TestCopyFile(t *testing.T) {
	testImage := "../../test/squashfs.img"
	f := New(testImage)

	if !f.CopyFile("/dir2/subdir2/file4", ".") {
		t.Errorf("CopyFile() returned false")
		return
	}
	defer os.Remove("file4")

	data, err := ioutil.ReadFile("file4")
	if err != nil {
		t.Errorf("can't read file4: %v", err)
		return
	}

	expected := "feed me a stray cat\n"
	if string(data) != expected {
		t.Errorf("file4 expected \"%s\" but got \"%s\"", expected, data)
	}
}
