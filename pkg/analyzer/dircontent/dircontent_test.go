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

package dircontent

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

func TestDirCheck(t *testing.T) {
	a := &testAnalyzer{}
	cfg := `
[DirContent."/temp"]
Allowed = ["file1", "file2"]
Required = ["file10"]
	`

	tests := []struct {
		path               string
		file               string
		shouldTrigger      bool
		shouldTriggerFinal bool
	}{
		{
			"/temp", "file1", false, true, // file allowed
		},
		{
			"/temp", "file4", true, true, // file not allowed
		},
		{
			"/temp1", "file4", false, true, // wrong dir, shouldn't matter
		},
		{
			"/temp", "file10", false, false, // file is required
		},
	}

	g := New(cfg, a)
	g.Start()

	for _, test := range tests {
		triggered := false
		a.ocb = func(fp string) { triggered = true }
		fi := fsparser.FileInfo{Name: test.file}
		err := g.CheckFile(&fi, test.path)
		if err != nil {
			t.Errorf("CheckFile returned error for %s", fi.Name)
		}
		if triggered != test.shouldTrigger {
			t.Errorf("incorrect result for %s/%s, wanted %v got %v", test.path, test.file, test.shouldTrigger, triggered)
		}

		triggered = false
		g.Finalize()
		if triggered != test.shouldTriggerFinal {
			t.Errorf("incorrect result for %s/%s on Finalize(), wanted %v got %v", test.path, test.file, test.shouldTriggerFinal, triggered)
		}
	}
}
