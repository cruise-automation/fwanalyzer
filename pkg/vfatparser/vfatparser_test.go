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
	"testing"
)

var f *VFatParser

func TestMain(t *testing.T) {
	testImage := "../../test/vfat.img"
	f = New(testImage)

	if f.ImageName() != testImage {
		t.Errorf("imageName returned bad name")
	}
}

func TestGetDirInfo(t *testing.T) {
	dir, err := f.GetDirInfo("/")
	if err != nil {
		t.Errorf("GetDirInfo was nil: %s", err)
	}
	for _, fi := range dir {
		if fi.Name == "dir1" {
			if !fi.IsDir() {
				t.Errorf("dir1 must be Dir")
			}
		}
	}

	dir, err = f.GetDirInfo("dir1")
	if err != nil {
		t.Errorf("GetDirInfo was nil: %s", err)
	}
	for _, fi := range dir {
		fmt.Printf("Name: %s Size: %d Mode: %o\n", fi.Name, fi.Size, fi.Mode)
		if fi.Name == "file1" {
			if !fi.IsFile() {
				t.Errorf("file1 need to be a file")
			}
			if fi.Uid != 0 || fi.Gid != 0 {
				t.Errorf("file1 need to be 0:0")
			}
			if fi.Size != 5 {
				t.Errorf("file1 size needs to be 5")
			}
			if !fi.IsWorldWrite() {
				t.Errorf("file1 needs to be world writable")
			}
		}
	}

	if !f.CopyFile("dir1/file1", ".") {
		t.Errorf("CopyFile returned false")
	}
	if _, err := os.Stat("file1"); os.IsNotExist(err) {
		t.Errorf("%s", err)
	} else {
		os.Remove("file1")
	}
}

func TestGetFileInfo(t *testing.T) {
	fi, err := f.GetFileInfo("/DIR1/FILE1")
	if err != nil {
		t.Error(err)
	}
	if !fi.IsFile() {
		t.Errorf("/DIR1/FILE1 should be file")
	}

	fi, err = f.GetFileInfo("/")
	if err != nil {
		t.Error(err)
	}
	if !fi.IsDir() {
		t.Errorf("/ should be dir")
	}
}
