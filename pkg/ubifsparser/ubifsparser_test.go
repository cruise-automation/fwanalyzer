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
	"os"
	"testing"
)

func TestCleanup(t *testing.T) {
	testImage := "../../test/ubifs.img"

	e := New(testImage)

	if e.ImageName() != testImage {
		t.Errorf("ImageName returned bad name")
	}

	fi, err := e.GetFileInfo("/")
	if err != nil {
		t.Error(err)
	}
	if !fi.IsDir() {
		t.Errorf("/ should be dir")
	}

	dir, err := e.GetDirInfo("/")
	if err != nil {
		t.Errorf("getDirList failed")
	}
	if len(dir) != 5 {
		t.Errorf("should be 5 files, but %d found", len(dir))
	}

	fi, err = e.GetFileInfo("/file1.txt")
	if err != nil {
		t.Errorf("GetFileInfo failed")
	}
	if !fi.IsFile() {
		t.Errorf("GetFileInfo failed, not a file")
	}
	if fi.IsDir() {
		t.Errorf("GetFileInfo failed, not a dir")
	}
	if fi.Name != "file1.txt" {
		t.Errorf("filename does not match: %s", fi.Name)
	}

	fi, err = e.GetFileInfo("/bin/elf_arm64")
	if err != nil {
		t.Errorf("GetFileInfo failed")
	}
	if !fi.IsFile() {
		t.Errorf("GetFileInfo failed, not a file")
	}
	if fi.IsDir() {
		t.Errorf("GetFileInfo failed, not a dir")
	}
	if fi.Size != 3740436 {
		t.Errorf("file size does not match: %s", fi.Name)
	}

	fi, err = e.GetFileInfo("/dateX")
	if err != nil {
		t.Errorf("GetFileInfo failed")
	}
	if fi.IsFile() {
		t.Errorf("GetFileInfo failed, not a file")
	}
	if fi.IsDir() {
		t.Errorf("GetFileInfo failed, not a dir")
	}
	if !fi.IsLink() {
		t.Errorf("GetFileInfo failed, is link")
	}
	if fi.LinkTarget != "date1.txt" {
		t.Errorf("link does not match: %s", fi.LinkTarget)
	}

	fi, err = e.GetFileInfo("/dir1")
	if err != nil {
		t.Errorf("GetFileInfo failed")
	}
	if fi.IsFile() {
		t.Errorf("GetFileInfo failed, not a file")
	}
	if !fi.IsDir() {
		t.Errorf("GetFileInfo failed, not a dir")
	}
	if fi.Name != "dir1" {
		t.Errorf("filename does not match: %s", fi.Name)
	}

	if !e.CopyFile("/bin/elf_arm32", "xxx-test-xxx") {
		t.Errorf("copyfile returned false")
	}
	if _, err := os.Stat("xxx-test-xxx"); os.IsNotExist(err) {
		t.Errorf("%s", err)
	} else {
		os.Remove("xxx-test-xxx")
	}
}
