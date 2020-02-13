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
	"strings"
	"testing"
)

var d *DirParser

func TestMain(t *testing.T) {
	testImage := "../../test/"
	d = New(testImage)

	if d.ImageName() != testImage {
		t.Errorf("ImageName returned bad name")
	}
}

func TestGetDirInfo(t *testing.T) {
	dir, err := d.GetDirInfo("/")
	if err != nil {
		t.Errorf("GetDirInfo failed")
	}
	for _, i := range dir {
		if i.Name == "." || i.Name == ".." {
			t.Errorf(". or .. should not appear in dir listing")
		}
	}

	output_file := "/tmp/dirfs_test_file"
	if !d.CopyFile("test.img", output_file) {
		t.Errorf("copyfile returned false")
	}
	if _, err := os.Stat(output_file); os.IsNotExist(err) {
		t.Errorf("%s", err)
	} else {
		os.Remove(output_file)
	}
}

func TestGetFileInfo(t *testing.T) {
	tests := []struct {
		filePath string
		isFile   bool
		isDir    bool
		filename string
	}{
		{"/test.img", true, false, "test.img"},
		{"/", false, true, "/"},
		{"/testdir", false, true, "testdir"},
	}
	for _, test := range tests {
		fi, err := d.GetFileInfo(test.filePath)
		if err != nil {
			t.Errorf("GetFileInfo failed")
		}
		if fi.IsFile() != test.isFile {
			t.Errorf("GetFileInfo failed, isFile != %v", test.isFile)
		}
		if fi.IsDir() != test.isDir {
			t.Errorf("GetFileInfo failed, isDir != %v", test.isDir)
		}
		if fi.Name != test.filename {
			t.Errorf("filename does not match: %s", fi.Name)
		}
	}

	fi, err := d.GetFileInfo("/testlink")
	if err != nil {
		t.Errorf("GetFileInfo failed")
	}
	if !fi.IsLink() {
		t.Errorf("GetFileInfo failed, not a link")
	}
	if fi.Name != "testlink" {
		t.Errorf("GetFileInfo failed, incorrect link name: %s", fi.Name)
	}
	if fi.LinkTarget != "testdir" {
		t.Errorf("GetFileInfo failed, incorrect link target: %s", fi.LinkTarget)
	}
}

func TestCapability(t *testing.T) {
	fi, err := d.GetFileInfo("/test.cap.file")
	if err != nil {
		t.Error(err)
	}
	fmt.Println(fi.Capabilities)
	if len(fi.Capabilities) == 0 || !strings.EqualFold(fi.Capabilities[0], "cap_net_admin+p") {
		t.Error("capability test failed: likely need to run 'sudo setcap cap_net_admin+p test/test.cap.file'")
	}
}
