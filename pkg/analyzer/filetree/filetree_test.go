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

package filetree

import (
	"os"
	"strings"
	"testing"

	"github.com/cruise-automation/fwanalyzer/pkg/analyzer"
	"github.com/cruise-automation/fwanalyzer/pkg/fsparser"
)

type OffenderCallack func(fn string, reason string)

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
}
func (a *testAnalyzer) AddInformational(filepath string, reason string) {
	a.ocb(filepath, reason)
}
func (a *testAnalyzer) CheckAllFilesWithPath(cb analyzer.AllFilesCallback, cbdata analyzer.AllFilesCallbackData, filepath string) {
}
func (a *testAnalyzer) ImageInfo() analyzer.AnalyzerReport {
	return analyzer.AnalyzerReport{}
}

func TestGlobal(t *testing.T) {

	a := &testAnalyzer{}

	cfg := `
[FileTreeCheck]
OldTreeFilePath = "/tmp/blatreetest1337.json"
CheckPath = ["/"]
CheckPermsOwnerChange = true
CheckFileSize         = true
CheckFileDigest       = false
`

	g := New(cfg, a, "")
	g.Start()

	triggered := false
	a.ocb = func(fn string, reason string) {
		if strings.HasPrefix(reason, "CheckFileTree: new file:") {
			triggered = true
		}
	}
	fi := fsparser.FileInfo{Name: "test1"}
	err := g.CheckFile(&fi, "/")
	if err != nil {
		t.Errorf("CheckFile failed")
	}

	result := g.Finalize()
	if !triggered {
		t.Errorf("filetree check failed")
	}

	if result == "" {
		t.Errorf("Finalize should not return empty string")
	}

	// rename so we have input for the next test
	err = os.Rename("/tmp/blatreetest1337.json.new", "/tmp/blatreetest1337.json")
	if err != nil {
		t.Errorf("rename %s %s: failed", "/tmp/blatreetest1337.json.new", "/tmp/blatreetest1337.json")
	}

	// diff test
	g = New(cfg, a, "")

	g.Start()

	triggered = false
	a.ocb = func(fn string, reason string) {
		if strings.HasPrefix(reason, "CheckFileTree: file perms/owner/size/digest changed") {
			triggered = true
		}
	}
	fi = fsparser.FileInfo{Name: "test1", Uid: 1}
	err = g.CheckFile(&fi, "/")
	if err != nil {
		t.Errorf("CheckFile failed")
	}

	g.Finalize()
	if !triggered {
		t.Errorf("filetree check failed")
	}

	// delete test
	g = New(cfg, a, "")

	g.Start()

	triggered = false
	a.ocb = func(fn string, reason string) {
		if fn == "/test1" && strings.HasPrefix(reason, "CheckFileTree: file removed") {
			triggered = true
		}
	}

	g.Finalize()
	if !triggered {
		t.Errorf("filetree check failed")
	}

	os.Remove("/tmp/blatreetest1337.json")
	os.Remove("/tmp/blatreetest1337.json.new")
	os.Remove("/tmp/blatreetest1337.json.new.new")
}

func TestGlobalCheckPath1(t *testing.T) {

	a := &testAnalyzer{}

	cfg := `
[FileTreeCheck]
OldTreeFilePath = "/tmp/blatreetest1337.json"
CheckPath = []
CheckPermsOwnerChange = true
CheckFileSize         = true
CheckFileDigest       = false
`
	g := New(cfg, a, "")

	if len(g.config.CheckPath) != 0 {
		t.Error("CheckPath should ne empty")
	}
}

func TestGlobalCheckPath2(t *testing.T) {

	a := &testAnalyzer{}

	cfg := `
[FileTreeCheck]
OldTreeFilePath = "/tmp/blatreetest1337.json"
CheckPermsOwnerChange = true
CheckFileSize         = true
CheckFileDigest       = false
`
	g := New(cfg, a, "")

	if len(g.config.CheckPath) != 1 && g.config.CheckPath[0] != "/" {
		t.Error("CheckPath should be: /")
	}
}
