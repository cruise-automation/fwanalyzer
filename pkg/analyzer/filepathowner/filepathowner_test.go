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

package filepathowner

import (
	"testing"

	"github.com/cruise-automation/fwanalyzer/pkg/analyzer"
	"github.com/cruise-automation/fwanalyzer/pkg/fsparser"
)

type OffenderCallack func(fn string)

type testAnalyzer struct {
	ocb      OffenderCallack
	testfile string
}

func (a *testAnalyzer) AddData(key, value string) {}
func (a *testAnalyzer) GetFileInfo(filepath string) (fsparser.FileInfo, error) {
	return fsparser.FileInfo{}, nil
}
func (a *testAnalyzer) RemoveFile(filepath string) error {
	return nil
}
func (a *testAnalyzer) FileGetSha256(filepath string) ([]byte, error) {
	return []byte(""), nil
}
func (a *testAnalyzer) FileGet(filepath string) (string, error) {
	return a.testfile, nil
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

func Test(t *testing.T) {

	a := &testAnalyzer{}

	cfg := `
[FilePathOwner."/bin"]
Uid = 0
Gid = 0
`

	g := New(cfg, a)

	g.Start()

	// uid/gid match
	triggered := false
	a.ocb = func(fp string) { triggered = true }
	fi := fsparser.FileInfo{Name: "test1", Uid: 0, Gid: 0}
	cbCheckOwnerPath(&fi, "/bin", &cbDataCheckOwnerPath{a, filePathOwner{0, 0}})
	if triggered {
		t.Errorf("checkOwnerPath failed")
	}

	// gid does not match
	triggered = false
	a.ocb = func(fp string) { triggered = true }
	fi = fsparser.FileInfo{Name: "test1", Uid: 0, Gid: 1}
	cbCheckOwnerPath(&fi, "/bin", &cbDataCheckOwnerPath{a, filePathOwner{0, 0}})
	if !triggered {
		t.Errorf("checkOwnerPath failed")
	}

	// do not call finalize() since we do not have a real source
}
