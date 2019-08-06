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

package filecontent

import (
	"io/ioutil"
	"os"
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

	if filepath == "/tmp/datatestfile.1" {
		return []byte("AABBCCDDEEFF11223344"), nil
	}
	return []byte("AABBCCDDEEFF11223341"), nil
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

func makeFile(data string, fn string) fsparser.FileInfo {
	err := ioutil.WriteFile("/tmp/"+fn, []byte(data), 0666)
	if err != nil {
		panic(err)
	}
	return fsparser.FileInfo{Name: fn, Size: 1}
}

func TestRegex(t *testing.T) {

	a := &testAnalyzer{}

	cfg := `
[FileContent."RegExTest1"]
RegEx = ".*Ver=1337.*"
Match = true
File ="/tmp/datatestfile.1"

[FileContent."RegExTest2"]
RegEx = ".*Ver=1337.*"
Match = true
File ="/tmp/datatestfile.1"
`

	g := New(cfg, a, false)

	g.Start()

	a.testfile = "/tmp/datatestfile.1"

	// match
	triggered := false
	a.ocb = func(fn string) { triggered = true }
	fi := makeFile("sadkljhlksaj Ver=1337  \naasas\n ", "datatestfile.1")
	err := g.CheckFile(&fi, "/tmp")
	if err != nil {
		t.Errorf("CheckFile failed")
	}
	if !triggered {
		t.Errorf("file content failed Regex")
	}
	os.Remove("/tmp/datatestfile.1")

	// do not match
	triggered = false
	a.ocb = func(fn string) { triggered = true }
	fi = makeFile("sadkljhlksaj Ver=1338\nasdads\nadaasd\n", "datatestfile.1")
	err = g.CheckFile(&fi, "/tmp")
	if err != nil {
		t.Errorf("CheckFile failed")
	}
	if triggered {
		t.Errorf("file content failed regex")
	}
	os.Remove("/tmp/datatestfile.1")

	// ensure file isn't flagged as not-found
	g.Finalize()
	if triggered {
		t.Errorf("file content failed, found file flagged as not-found")
	}
}

func TestDigest(t *testing.T) {

	a := &testAnalyzer{}

	cfg := `
[FileContent."digest test 1"]
Digest = "4141424243434444454546463131323233333434"
File = "/tmp/datatestfile.1"

[FileContent."digest test 2"]
Digest = "4141424243434444454546463131323233333435"
File ="/tmp/datatestfile.2"
`

	g := New(cfg, a, false)

	g.Start()

	a.testfile = "/tmp/datatestfile.1"

	// match
	triggered := false
	a.ocb = func(fn string) { triggered = true }
	fi := makeFile("sadkljhlksaj Ver=1337  \naasas\n ", "datatestfile.1")
	err := g.CheckFile(&fi, "/tmp")
	if err != nil {
		t.Errorf("CheckFile failed")
	}
	if triggered {
		t.Errorf("file content failed digest")
	}
	os.Remove("/tmp/datatestfile.1")

	// do not match
	triggered = false
	a.ocb = func(fn string) { triggered = true }
	fi = makeFile("sadkljhlksaj Ver=1338\nasdads\nadaasd\n", "datatestfile.2")
	err = g.CheckFile(&fi, "/tmp")
	if err != nil {
		t.Errorf("CheckFile failed")
	}
	if !triggered {
		t.Errorf("file content failed digest")
	}
	os.Remove("/tmp/datatestfile.2")

	g.Finalize()
}

func TestScript(t *testing.T) {

	a := &testAnalyzer{}

	cfg := `
[FileContent."script test 1"]
Script="/tmp/testfilescript.sh"
File = "/tmp/datatestfile.1"
`

	script := `#!/bin/sh
cat $1
`

	err := ioutil.WriteFile("/tmp/testfilescript.sh", []byte(script), 0777)
	if err != nil {
		t.Error(err)
	}

	g := New(cfg, a, false)

	g.Start()

	a.testfile = "/tmp/datatestfile.1"

	// match
	triggered := false
	a.ocb = func(fn string) { triggered = true }
	fi := makeFile("sadkljhlksaj Ver=1337  \naasas\n ", "datatestfile.1")
	err = g.CheckFile(&fi, "/tmp")
	if err != nil {
		t.Errorf("CheckFile failed")
	}
	if !triggered {
		t.Errorf("file content script test failed")
	}
	os.Remove("/tmp/datatestfile.1")
	os.Remove("/tmp/testfilescript.sh")

	g.Finalize()
}

func TestValidateItem(t *testing.T) {

	a := &testAnalyzer{}

	cfg := `
[FileContent."digest test 1"]
Digest = "4141424243434444454546463131323233333434"
Script = "asdf.sh"
File = "/tmp/datatestfile.1"
`
	triggered := false
	a.ocb = func(fn string) { triggered = true }
	g := New(cfg, a, false)
	if !triggered {
		t.Errorf("file content failed validate with multiple check types")
	}
	g.Finalize()

	triggered = false
	cfg = `
[FileContent."digest test 1"]
File = "/tmp/datatestfile.1"
`

	New(cfg, a, false)
	if !triggered {
		t.Errorf("file content failed validate without check type")
	}
}

func TestMissingFile(t *testing.T) {
	a := &testAnalyzer{}

	cfg := `
[FileContent."RegExTest1"]
RegEx = ".*Ver=1337.*"
Match = true
File ="/tmp/datatestfile.notfound"
`
	g := New(cfg, a, false)
	g.Start()
	a.testfile = "/tmp/datatestfile.1"

	// match
	triggered := false
	a.ocb = func(fn string) { triggered = true }
	fi := makeFile("sadkljhlksaj Ver=1337  \naasas\n ", "datatestfile.1")
	err := g.CheckFile(&fi, "/tmp")
	if err != nil {
		t.Errorf("CheckFile failed")
	}
	// pass should still be false here because CheckFile did not see the file
	if triggered {
		t.Errorf("file content failed, missing file checked")
	}

	os.Remove("/tmp/datatestfile.1")
	g.Finalize()
	// triggered should be true here because Finalize should call AddOffender
	if !triggered {
		t.Errorf("file content failed, missing file not found")
	}
}

func TestJson(t *testing.T) {

	a := &testAnalyzer{}

	cfg := `
[FileContent."json test 1"]
Json="a.b:test123"
File = "/tmp/datatestfile.1"
`
	g := New(cfg, a, false)

	g.Start()

	a.testfile = "/tmp/datatestfile.1"

	triggered := false
	a.ocb = func(fn string) { triggered = true }
	fi := makeFile(`{"a":{"b": "test123"}}`, "datatestfile.1")
	err := g.CheckFile(&fi, "/tmp")
	if err != nil {
		t.Errorf("CheckFile failed")
	}
	if triggered {
		t.Errorf("file content json failed")
	}
	os.Remove("/tmp/datatestfile.1")

	g.Finalize()
}

func TestJsonDoesNotMatch(t *testing.T) {

	a := &testAnalyzer{}

	cfg := `
[FileContent."json test 1"]
Json="a.b:test12A"
File = "/tmp/datatestfile.1"
`
	g := New(cfg, a, false)

	g.Start()

	a.testfile = "/tmp/datatestfile.1"

	triggered := false
	a.ocb = func(fn string) { triggered = true }
	fi := makeFile(`{"a":{"b": "test123"}}`, "datatestfile.1")
	err := g.CheckFile(&fi, "/tmp")
	if err != nil {
		t.Errorf("CheckFile failed")
	}
	if !triggered {
		t.Errorf("file content json failed")
	}
	os.Remove("/tmp/datatestfile.1")

	g.Finalize()
}

func TestGlobalInvert(t *testing.T) {

	a := &testAnalyzer{}

	cfg := `[FileContent."RegExTest1"]
RegEx = ".*Ver=1337.*"
Match = true
File ="/tmp/datatestfile.1"

[FileContent."RegExTest2"]
RegEx = ".*Ver=1337.*"
Match = true
File ="/tmp/datatestfile.1"
`

	g := New(cfg, a, true)

	g.Start()

	a.testfile = "/tmp/datatestfile.1"

	// match
	triggered := false
	a.ocb = func(fn string) { triggered = true }
	fi := makeFile("sadkljhlksaj Ver=1337  \naasas\n ", "datatestfile.1")
	err := g.CheckFile(&fi, "/tmp")
	if err != nil {
		t.Errorf("CheckFile failed")
	}
	if triggered {
		t.Errorf("file content failed Regex")
	}
	os.Remove("/tmp/datatestfile.1")

	// dont match
	triggered = false
	a.ocb = func(fn string) { triggered = true }
	fi = makeFile("sadkljhlksaj Ver=1338\nasdads\nadaasd\n", "datatestfile.1")
	err = g.CheckFile(&fi, "/tmp")
	if err != nil {
		t.Errorf("CheckFile failed")
	}
	if !triggered {
		t.Errorf("file content failed regex")
	}
	os.Remove("/tmp/datatestfile.1")

	// ensure file isn't flagged as not-found
	g.Finalize()
	if !triggered {
		t.Errorf("file content failed, found file flagged as not-found")
	}
}
