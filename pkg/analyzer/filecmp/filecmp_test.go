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

package filecmp

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/cruise-automation/fwanalyzer/pkg/analyzer"
	"github.com/cruise-automation/fwanalyzer/pkg/fsparser"
)

type OffenderCallack func(fn string, info bool)

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
	a.ocb(reason, false)
}
func (a *testAnalyzer) AddInformational(filepath string, reason string) {
	a.ocb(reason, true)
}
func (a *testAnalyzer) CheckAllFilesWithPath(cb analyzer.AllFilesCallback, cbdata analyzer.AllFilesCallbackData, filepath string) {
}
func (a *testAnalyzer) ImageInfo() analyzer.AnalyzerReport {
	return analyzer.AnalyzerReport{}
}

func TestCmp(t *testing.T) {

	a := &testAnalyzer{}

	cfg := `
[FileCmp."Test1"]
File ="/cmp_test_1"
Script = "diff.sh"
ScriptOptions = [""]
OldFilePath = "/tmp/analyzer_filecmp_1"
`

	g := New(cfg, a, "")
	g.Start()

	called := false
	infoText := ""
	a.ocb = func(name string, info bool) {
		called = true
		infoText = name
	}

	// same file should not produce output

	data := `
	aaa
	bbb
	ccc
	ddd
	`

	err := ioutil.WriteFile("/tmp/analyzer_filecmp_1", []byte(data), 0755)
	if err != nil {
		t.Error(err)
	}

	a.testfile = "/tmp/analyzer_filecmp_1"

	called = false
	infoText = ""

	fi := fsparser.FileInfo{Name: "cmp_test_1", Mode: 100755}
	err = g.CheckFile(&fi, "/")
	if err != nil {
		t.Error(err)
	}

	if called {
		t.Errorf("should not produce offender: %s", infoText)
	}

	// should cause an offender

	called = false
	infoText = ""

	data = `
	aaa
	bbb
	ccc
	ddd
	`

	err = ioutil.WriteFile("/tmp/analyzer_filecmp_1", []byte(data), 0755)
	if err != nil {
		t.Error(err)
	}

	data = `
	aaa
	ddd
	ccc
	`

	err = ioutil.WriteFile("/tmp/analyzer_filecmp_2", []byte(data), 0755)
	if err != nil {
		t.Error(err)
	}

	a.testfile = "/tmp/analyzer_filecmp_2"

	fi = fsparser.FileInfo{Name: "cmp_test_1", Mode: 100755}
	err = g.CheckFile(&fi, "/")
	if err != nil {
		t.Error(err)
	}

	if !called {
		t.Errorf("should produce offender: %s", infoText)
	}
}

func TestCmpInfo(t *testing.T) {

	a := &testAnalyzer{}

	cfg := `
[FileCmp."Test1"]
File ="/cmp_test_1"
Script = "diff.sh"
ScriptOptions = [""]
InformationalOnly = true
OldFilePath = "/tmp/analyzer_filecmp_1"
`

	g := New(cfg, a, "")
	g.Start()

	called := false
	infoText := ""
	infoO := false
	a.ocb = func(name string, info bool) {
		called = true
		infoText = name
		infoO = info
	}

	// should cause an informational

	data := `
	aaa
	bbb
	ccc
	ddd
	`

	err := ioutil.WriteFile("/tmp/analyzer_filecmp_1", []byte(data), 0755)
	if err != nil {
		t.Error(err)
	}

	data = `
	aaa
	ddd
	ccc
	`

	err = ioutil.WriteFile("/tmp/analyzer_filecmp_2", []byte(data), 0755)
	if err != nil {
		t.Error(err)
	}

	a.testfile = "/tmp/analyzer_filecmp_2"

	fi := fsparser.FileInfo{Name: "cmp_test_1", Mode: 100755}
	err = g.CheckFile(&fi, "/")
	if err != nil {
		t.Error(err)
	}

	if !called || !infoO {
		t.Errorf("should produce informational: %s", infoText)
	}
}

func TestCmpNoOld(t *testing.T) {

	a := &testAnalyzer{}

	cfg := `
[FileCmp."Test1"]
File ="/cmp_test_1"
Script = "diff.sh"
ScriptOptions = [""]
OldFilePath = "/tmp/analyzer_filecmp_99"
`

	g := New(cfg, a, "")
	g.Start()

	called := false
	infoText := ""
	infoO := false
	a.ocb = func(name string, info bool) {
		called = true
		infoText = name
		infoO = info
	}

	// should cause an informational

	data := `
	aaa
	bbb
	ccc
	ddd
	`

	err := ioutil.WriteFile("/tmp/analyzer_filecmp_1", []byte(data), 0755)
	if err != nil {
		t.Error(err)
	}

	a.testfile = "/tmp/analyzer_filecmp_1"

	os.Remove("/tmp/analyzer_filecmp_99.new")

	fi := fsparser.FileInfo{Name: "cmp_test_1", Mode: 100755}
	err = g.CheckFile(&fi, "/")
	if err != nil {
		t.Error(err)
	}

	if !called || !infoO {
		t.Errorf("should produce informational: %s", infoText)
	}

	inData, err := ioutil.ReadFile("/tmp/analyzer_filecmp_99.new")
	if err != nil {
		t.Error(err)
	}
	if string(inData) != data {
		t.Errorf("files not equal after save")
	}
}
