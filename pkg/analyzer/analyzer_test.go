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

package analyzer

import (
	"os"
	"testing"
)

func TestBasic(t *testing.T) {
	cfg := `
[GlobalConfig]
FsType = "dirfs"
DigestImage = false
`

	// check tmp file test
	analyzer := NewFromConfig("../../test/testdir", cfg)
	_ = analyzer.CleanUp()
	if _, err := os.Stat(analyzer.tmpdir); !os.IsNotExist(err) {
		t.Errorf("tmpdir was not removed")
	}

	// file test
	analyzer = NewFromConfig("../../test/testdir", cfg)
	fi, err := analyzer.GetFileInfo("/file1.txt")
	if err != nil {
		t.Errorf("GetFileInfo failed")
	}
	if !fi.IsFile() {
		t.Errorf("GetFileInfo failed, should be regular file")
	}
	if fi.IsDir() {
		t.Errorf("GetFileInfo failed, not a dir")
	}
	if fi.Name != "file1.txt" {
		t.Errorf("filename does not match")
	}

	// directory test
	fi, err = analyzer.GetFileInfo("/dir1")
	if err != nil {
		t.Errorf("GetFileInfo failed")
	}
	if fi.IsFile() {
		t.Errorf("GetFileInfo failed, not a file")
	}
	if !fi.IsDir() {
		t.Errorf("GetFileInfo failed, should be a directory")
	}
	if fi.Name != "dir1" {
		t.Errorf("filename does not match")
	}

	err = analyzer.checkRoot()
	if err != nil {
		t.Errorf("checkroot failed with %s", err)
	}

	_ = analyzer.CleanUp()
}
