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

package filestatcheck

import (
	"fmt"
	"testing"

	"github.com/cruise-automation/fwanalyzer/pkg/analyzer"
	"github.com/cruise-automation/fwanalyzer/pkg/fsparser"
)

type OffenderCallack func(fn string)

type testAnalyzer struct {
	ocb OffenderCallack
	fi  fsparser.FileInfo
	err error
}

func (a *testAnalyzer) AddData(key, value string) {}

func (a *testAnalyzer) GetFileInfo(filepath string) (fsparser.FileInfo, error) {
	return a.fi, a.err
}
func (a *testAnalyzer) RemoveFile(filepath string) error {
	return nil
}
func (a *testAnalyzer) FileGetSha256(filepath string) ([]byte, error) {
	return []byte{}, nil
}
func (a *testAnalyzer) FileGet(filepath string) (string, error) {
	return "", nil
}
func (a *testAnalyzer) AddOffender(filepath string, reason string) {
	a.ocb(filepath)
}
func (a *testAnalyzer) AddInformational(filepath string, reason string) {}
func (a *testAnalyzer) CheckAllFilesWithPath(cb analyzer.AllFilesCallback, cbdata analyzer.AllFilesCallbackData, filepath string) {
}
func (a *testAnalyzer) ImageInfo() analyzer.AnalyzerReport {
	return analyzer.AnalyzerReport{}
}

func TestGlobal(t *testing.T) {

	a := &testAnalyzer{}
	a.err = nil

	cfg := `
[FileStatCheck."/file1111"]
AllowEmpty = false
Uid = 1
Mode = "0755"
Desc = "this need to be this way"`

	g := New(cfg, a)

	// ensure gid/uid are set to correct values
	for _, item := range g.files.FileStatCheck {
		if item.Gid != -1 {
			t.Errorf("Gid should default to -1, is %d", item.Gid)
		}

		if item.Uid != 1 {
			t.Errorf("Uid should be 1, is %d", item.Uid)
		}
	}

	g.Start()

	fi := fsparser.FileInfo{}
	if g.CheckFile(&fi, "/") != nil {
		t.Errorf("checkfile failed")
	}

	tests := []struct {
		fi            fsparser.FileInfo
		err           error
		shouldTrigger bool
	}{

		{fsparser.FileInfo{Name: "file1111", Uid: 0, Gid: 0, Mode: 0755, Size: 1}, nil, true},
		{fsparser.FileInfo{Name: "file1111", Uid: 1, Gid: 0, Mode: 0755, Size: 0}, nil, true},
		{fsparser.FileInfo{Name: "file1111", Uid: 1, Gid: 1, Mode: 0755, Size: 1}, nil, false},
		{fsparser.FileInfo{Name: "file1111", Uid: 1, Gid: 0, Mode: 0754, Size: 1}, nil, true},
		{
			fsparser.FileInfo{Name: "filedoesnotexist", Uid: 0, Gid: 0, Mode: 0755, Size: 1},
			fmt.Errorf("file does not exist"),
			true,
		},
	}
	var triggered bool
	for _, test := range tests {
		triggered = false
		a.fi = test.fi
		a.err = test.err
		a.ocb = func(fn string) { triggered = true }
		g.Finalize()
		if triggered != test.shouldTrigger {
			t.Errorf("FileStatCheck failed")
		}
	}
}
