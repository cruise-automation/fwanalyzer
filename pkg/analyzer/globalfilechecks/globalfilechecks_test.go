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

package globalfilechecks

import (
	"testing"

	"github.com/cruise-automation/fwanalyzer/pkg/analyzer"
	"github.com/cruise-automation/fwanalyzer/pkg/fsparser"
)

type OffenderCallack func(fn string)

type testAnalyzer struct {
	ocb OffenderCallack
}

func (a *testAnalyzer) AddData(key, value string) {}
func (a *testAnalyzer) GetFileInfo(filepath string) (fsparser.FileInfo, error) {
	return fsparser.FileInfo{}, nil
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
	cfg := `
[GlobalFileChecks]
Suid = true
SuidWhiteList = ["/shouldbesuid"]
SeLinuxLabel = true
WorldWrite = true
Uids = [0]
Gids = [0]
BadFiles = ["/file99", "/file1", "**.h"]
`

	g := New(cfg, a)
	g.Start()

	tests := []struct {
		fi            fsparser.FileInfo
		path          string
		shouldTrigger bool
	}{
		{fsparser.FileInfo{Name: "suid", Mode: 0004000}, "/", true},
		{fsparser.FileInfo{Name: "sgid", Mode: 0002000}, "/", true},
		{fsparser.FileInfo{Name: "sgid", Mode: 0000000}, "/", false},
		// Whitelisted
		{fsparser.FileInfo{Name: "shouldbesuid", Mode: 0004000}, "/", false},
		// World write
		{fsparser.FileInfo{Name: "ww", Mode: 0007}, "/", true},
		{fsparser.FileInfo{Name: "ww", Mode: 0004}, "/", false},
		{fsparser.FileInfo{Name: "label", SELinuxLabel: "-"}, "/", true},
		{fsparser.FileInfo{Name: "label", SELinuxLabel: "label"}, "/", false},
		{fsparser.FileInfo{Name: "uidfile", SELinuxLabel: "uidfile", Uid: 1, Gid: 1}, "/", true},
		// Bad files
		{fsparser.FileInfo{Name: "file99", SELinuxLabel: "uidfile"}, "/", true},
		{fsparser.FileInfo{Name: "test.h", SELinuxLabel: "uidfile"}, "/usr/", true},
	}

	var triggered bool
	var err error
	for _, test := range tests {
		triggered = false
		a.ocb = func(fn string) { triggered = true }
		err = g.CheckFile(&test.fi, test.path)
		if err != nil {
			t.Errorf("CheckFile failed")
		}
		if triggered != test.shouldTrigger {
			t.Errorf("%s test failed", test.fi.Name)
		}
	}

	g.Finalize()
}
