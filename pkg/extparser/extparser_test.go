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
	"os"
	"testing"
)

var e *Ext2Parser

func TestMain(t *testing.T) {
	testImage := "../../test/test.img"

	e = New(testImage, false)

	if e.ImageName() != testImage {
		t.Errorf("ImageName returned bad name")
	}
}

func TestGetDirList(t *testing.T) {
	dir, err := e.getDirList("/", true)
	if err != nil {
		t.Errorf("getDirList failed")
	}
	for _, i := range dir {
		if i.Name == "." || i.Name == ".." {
			t.Errorf(". or .. should not appear in dir listing")
		}
	}

	dir, err = e.getDirList("/", false)
	if err != nil {
		t.Errorf("getDirList failed")
	}

	dot := false
	dotdot := false
	for _, i := range dir {
		if i.Name == "." {
			dot = true
		}
		if i.Name == ".." {
			dotdot = true
		}
	}
	if !dot || !dotdot {
		t.Errorf(". and .. should appear in dir listing")
	}
}

func TestGetDirInfo(t *testing.T) {
	dir, err := e.GetDirInfo("/")
	if err != nil {
		t.Errorf("GetDirInfo failed")
	}
	for _, i := range dir {
		if i.Name == "." || i.Name == ".." {
			t.Errorf(". or .. should not appear in dir listing")
		}
	}
	if len(dir) == 0 {
		t.Errorf("root needs to be >= 1 entries due to lost+found")
	}

	if !e.CopyFile("/date1", ".") {
		t.Errorf("copyfile returned false")
	}
	if _, err := os.Stat("date1"); os.IsNotExist(err) {
		t.Errorf("%s", err)
	} else {
		os.Remove("date1")
	}
}

func TestGetFileInfo(t *testing.T) {
	tests := []struct {
		filePath string
		isFile   bool
		isDir    bool
		filename string
	}{
		{"/date1", true, false, "date1"},
		{"/", false, true, "/"},
		{"/dir1", false, true, "dir1"},
	}
	for _, test := range tests {
		fi, err := e.GetFileInfo(test.filePath)
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
}
