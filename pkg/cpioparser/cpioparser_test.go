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

package cpioparser

import (
	"os"
	"testing"
)

type testData struct {
	Line       string
	Mode       uint64
	Dir        string
	Name       string
	IsFile     bool
	LinkTarget string
}

func TestParseLine(t *testing.T) {
	testImage := "../../test/test.cpio"
	p := New(testImage, true)

	testdata := []testData{
		{`-rw-r--r--   1 0        0              21 Apr 11  2008 etc/motd`, 0100644, "/etc/", "motd", true, ""},
		{`-rw-r--r--   1 0        0              21 Apr 11 13:37 etc/mxtd`, 0100644, "/etc/", "mxtd", true, ""},
		{`crw-r--r--   1 0        0          4,  64 Apr 24  2019 dev/ttyS0`, 020644, "/dev", "ttyS0", false, ""},
		{`lrwxrwxrwx   1 0        0              19 Apr 24  2019 lib/libcrypto.so.1.0.0 -> libcrypto-1.0.0.so`, 0120777, "/lib", "libcrypto.so.1.0.0", false, "libcrypto-1.0.0.so"},
		{`drwxr-xr-x   2 0        0               0 Aug  8 18:53 .`, 040755, ".", ".", false, ""},
	}

	for _, test := range testdata {
		dir, res, err := p.parseFileLine(test.Line)
		if err != nil {
			t.Error(err)
		}
		if res.Mode != test.Mode {
			t.Errorf("bad file mode: %o", res.Mode)
		}
		if dir != test.Dir && res.Name != test.Name {
			t.Errorf("name error: %s %s", dir, res.Name)
		}
		if res.IsFile() != test.IsFile {
			t.Error("isFile bad")
		}
		if test.LinkTarget != res.LinkTarget {
			t.Errorf("bad link target: %s", res.LinkTarget)
		}
	}
}

func TestFixDir(t *testing.T) {
	testImage := "../../test/test.cpio"
	p := New(testImage, true)

	testdata := `
crw-r--r--   1 0        0          3,   1 Jan 13 17:57 dev/ttyp1
crw-r--r--   1 0        0          3,   1 Jan 13 17:57 dev/x/ttyp1`

	err := p.loadFileListFromString(testdata)
	if err != nil {
		t.Error(err)
	}

	ok := false
	for _, fn := range p.files["/"] {
		if fn.Name == "dev" {
			ok = true
		}
	}
	if !ok {
		t.Errorf("dir '/dev' not found")
	}

	ok = false
	for _, fn := range p.files["/dev"] {
		if fn.Name == "x" {
			ok = true
		}
	}
	if !ok {
		t.Errorf("dir '/dev/x' not found")
	}
}

func TestFull(t *testing.T) {
	testImage := "../../test/test.cpio"
	p := New(testImage, false)

	fi, err := p.GetFileInfo("/")
	if err != nil {
		t.Error(err)
	}
	if !fi.IsDir() {
		t.Errorf("/ should be dir")
	}

	dir, err := p.GetDirInfo("/")
	if err != nil {
		t.Error(err)
	}
	if len(dir) < 1 {
		t.Errorf("/ should not be empty")
	}

	fi, err = p.GetFileInfo("/etc/fstab")
	if err != nil {
		t.Error(err)
	}
	if !fi.IsFile() {
		t.Error("should be a file")
	}
	if fi.Name != "fstab" {
		t.Errorf("name bad: %s", fi.Name)
	}
	if fi.IsDir() {
		t.Error("should be a file")
	}
	if fi.Size != 385 {
		t.Error("bad size")
	}
	if fi.Uid != 1000 || fi.Gid != 1000 {
		t.Error("bad owner/group")
	}

	fi, err = p.GetFileInfo("/dev/tty6")
	if err != nil {
		t.Error(err)
	}
	if fi.IsFile() {
		t.Error("should not be a file")
	}
	if fi.Name != "tty6" {
		t.Errorf("name bad: %s", fi.Name)
	}
	if fi.IsDir() {
		t.Error("should not be a dir")
	}
	if fi.Size != 0 {
		t.Error("bad size")
	}
	if fi.Uid != 0 || fi.Gid != 0 {
		t.Error("bad owner/group")
	}

	testfilename := "testfile123"
	if !p.CopyFile("/etc/fstab", testfilename) {
		t.Error("failed to copy fstab")
	}

	stat, err := os.Stat(testfilename)
	if err != nil {
		t.Error(err)
	}
	if stat.Size() != 385 {
		t.Error("bad file size after copy out")
	}
	os.Remove(testfilename)
}
