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

package dataextract

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/cruise-automation/fwanalyzer/pkg/analyzer"
	"github.com/cruise-automation/fwanalyzer/pkg/fsparser"
	"github.com/cruise-automation/fwanalyzer/pkg/util"
)

type testAnalyzer struct {
	Data     map[string]string
	testfile string
}

func (a *testAnalyzer) AddData(key, value string) {
	a.Data[key] = value
}

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

func TestRegex1(t *testing.T) {

	a := &testAnalyzer{}
	a.Data = make(map[string]string)

	cfg := `
[DataExtract."Version"]
File = "/tmp/datatestfileX.1"
RegEx = ".*Ver=(.+)\n"
Desc="Ver 1337 test"
`

	g := New(cfg, a)

	g.Start()

	a.testfile = "/tmp/datatestfileX.1"

	// must match
	fi := makeFile("sadkljhlksaj Ver=1337\naasas\n ", "datatestfileX.1")
	err := g.CheckFile(&fi, "/tmp")
	if err != nil {
		t.Errorf("CheckFile failed")
	}
	if data, ok := a.Data["Version"]; !ok || data != "1337" {
		t.Errorf("data extract failed Regex")
	}
	os.Remove("/tmp/datatestfileX.1")
	delete(a.Data, "Version")

	// must not match
	fi = makeFile("sadkljhlksaj ver=1337\naasas\n ", "datatestfileX.1")
	err = g.CheckFile(&fi, "/tmp")
	if err != nil {
		t.Errorf("CheckFile failed")
	}
	if data, ok := a.Data["Version"]; ok && data == "1337" {
		t.Errorf("data extract failed Regex")
	}
	os.Remove("/tmp/datatestfileX.1")

	g.Finalize()
}

func TestScript1(t *testing.T) {

	a := &testAnalyzer{}
	a.Data = make(map[string]string)

	cfg := `
[DataExtract.LastLine]
File = "/tmp/datatestfileX.1"
Script="/tmp/extractscripttest.sh"
Desc="last line test"
`

	script := `#!/bin/sh
tail -n 1 $1
`

	err := ioutil.WriteFile("/tmp/extractscripttest.sh", []byte(script), 0777)
	if err != nil {
		t.Error(err)
	}

	g := New(cfg, a)

	g.Start()

	a.testfile = "/tmp/datatestfileX.1"

	fi := makeFile("lskjadh\naskhj23832\n\nkjhf21987\nhello world\n", "datatestfileX.1")
	err = g.CheckFile(&fi, "/tmp")
	if err != nil {
		t.Errorf("CheckFile failed")
	}
	if data, ok := a.Data["LastLine"]; !ok || data != "hello world\n" {
		t.Errorf("data extract failed script")
	}
	os.Remove("/tmp/datatestfileX.1")
	os.Remove("/tmp/extractscripttest.sh")

	g.Finalize()
}

func TestMulti(t *testing.T) {

	a := &testAnalyzer{}
	a.Data = make(map[string]string)

	cfg := `
[DataExtract."1"]
File = "/tmp/datatestfileX.1"
RegEx = ".*Ver=(.+)\n"
Name = "Version"

[DataExtract."2"]
File = "/tmp/datatestfileX.1"
RegEx = ".*Version=(.+)\n"
Name = "Version"
`
	g := New(cfg, a)

	g.Start()

	a.testfile = "/tmp/datatestfileX.1"

	fi := makeFile("sadkljhlksaj Version=1337\naasas\n ", "datatestfileX.1")
	err := g.CheckFile(&fi, "/tmp")
	if err != nil {
		t.Errorf("CheckFile failed")
	}
	if data, ok := a.Data["Version"]; !ok || data != "1337" {
		t.Errorf("data extract failed Regex")
	}
	os.Remove("/tmp/datatestfileX.1")
	delete(a.Data, "Version")

	fi = makeFile("sadkljhlksaj Ver=1337\naasas\n ", "datatestfileX.1")
	err = g.CheckFile(&fi, "/tmp")
	if err != nil {
		t.Errorf("CheckFile failed")
	}
	if data, ok := a.Data["Version"]; !ok && data == "1337" {
		t.Errorf("data extract failed Regex")
	}
	os.Remove("/tmp/datatestfileX.1")

	g.Finalize()
}

func TestAutoNaming(t *testing.T) {

	a := &testAnalyzer{}
	a.Data = make(map[string]string)

	cfg := `
[DataExtract."Version__9"]
File = "/tmp/datatestfileX.1"
RegEx = ".*Ver=(.+)\n"

[DataExtract."Version__0"]
File = "/tmp/datatestfileX.1"
RegEx = ".*Version=(.+)\n"
`
	g := New(cfg, a)

	g.Start()

	a.testfile = "/tmp/datatestfileX.1"

	fi := makeFile("sadkljhlksaj Version=1337\naasas\n ", "datatestfileX.1")
	err := g.CheckFile(&fi, "/tmp")
	if err != nil {
		t.Errorf("CheckFile failed")
	}
	if data, ok := a.Data["Version"]; !ok || data != "1337" {
		t.Errorf("data extract failed Regex")
	}
	os.Remove("/tmp/datatestfileX.1")
	delete(a.Data, "Version")

	fi = makeFile("sadkljhlksaj Ver=1337\naasas\n ", "datatestfileX.1")
	err = g.CheckFile(&fi, "/tmp")
	if err != nil {
		t.Errorf("CheckFile failed")
	}
	if data, ok := a.Data["Version"]; !ok && data == "1337" {
		t.Errorf("data extract failed Regex")
	}
	os.Remove("/tmp/datatestfileX.1")

	g.Finalize()
}

func TestJson1(t *testing.T) {

	a := &testAnalyzer{}
	a.Data = make(map[string]string)

	cfg := `
[DataExtract."Version__9"]
File = "/tmp/datatestfileX.1"
Json = "a"
`
	g := New(cfg, a)

	g.Start()

	a.testfile = "/tmp/datatestfileX.1"

	fi := makeFile(`{"a":"lalala"}`, "datatestfileX.1")
	err := g.CheckFile(&fi, "/tmp")
	if err != nil {
		t.Errorf("CheckFile failed")
	}
	if data, ok := a.Data["Version"]; !ok || data != "lalala" {
		t.Errorf("data extract failed Json")
	}
	os.Remove("/tmp/datatestfileX.1")
	delete(a.Data, "Version")

	g.Finalize()
}

func TestJson2(t *testing.T) {

	a := &testAnalyzer{}
	a.Data = make(map[string]string)

	cfg := `
[DataExtract."Version__9"]
File = "/tmp/datatestfileX.1"
Json = "a.b"
`
	g := New(cfg, a)

	g.Start()

	a.testfile = "/tmp/datatestfileX.1"

	fi := makeFile(`{"a":{"b": "lalala123"}}`, "datatestfileX.1")
	err := g.CheckFile(&fi, "/tmp")
	if err != nil {
		t.Errorf("CheckFile failed")
	}
	if data, ok := a.Data["Version"]; !ok || data != "lalala123" {
		t.Errorf("data extract failed Json")
	}
	os.Remove("/tmp/datatestfileX.1")
	delete(a.Data, "Version")

	g.Finalize()
}

func TestJson3Bool(t *testing.T) {

	a := &testAnalyzer{}
	a.Data = make(map[string]string)

	cfg := `
[DataExtract."Version__9"]
File = "/tmp/datatestfileX.1"
Json = "a.c"
`
	g := New(cfg, a)

	g.Start()

	a.testfile = "/tmp/datatestfileX.1"

	fi := makeFile(`{"a":{"c": true}}`, "datatestfileX.1")
	err := g.CheckFile(&fi, "/tmp")
	if err != nil {
		t.Errorf("CheckFile failed")
	}
	if data, ok := a.Data["Version"]; !ok || data != "true" {
		t.Errorf("data extract failed Json")
	}
	os.Remove("/tmp/datatestfileX.1")
	delete(a.Data, "Version")

	g.Finalize()
}

func TestJsonError(t *testing.T) {

	a := &testAnalyzer{}
	a.Data = make(map[string]string)

	cfg := `
[DataExtract."Version__9"]
File = "/tmp/datatestfileX.1"
Json = "a.c"
`
	g := New(cfg, a)

	g.Start()

	a.testfile = "/tmp/datatestfileX.1"

	fi := makeFile(`{"a":{"c": true}`, "datatestfileX.1")
	err := g.CheckFile(&fi, "/tmp")
	if err != nil {
		t.Errorf("CheckFile failed")
	}
	if data, ok := a.Data["Version"]; !ok || data == "true" {
		t.Errorf("data extract failed Json: %s", a.Data["Version"])
	}
	os.Remove("/tmp/datatestfileX.1")
	delete(a.Data, "Version")

	g.Finalize()
}

func TestJson4Num(t *testing.T) {

	a := &testAnalyzer{}
	a.Data = make(map[string]string)

	cfg := `
[DataExtract."Version__9"]
File = "/tmp/datatestfileX.1"
Json = "a.d"
`
	g := New(cfg, a)

	g.Start()

	a.testfile = "/tmp/datatestfileX.1"

	fi := makeFile(`{"a":{"d": 123}}`, "datatestfileX.1")
	err := g.CheckFile(&fi, "/tmp")
	if err != nil {
		t.Errorf("CheckFile failed")
	}
	if data, ok := a.Data["Version"]; !ok || data != "123.000000" {
		t.Errorf("data extract failed Json, %s", a.Data["Version"])
	}
	os.Remove("/tmp/datatestfileX.1")
	delete(a.Data, "Version")

	g.Finalize()
}

func TestJson5Deep(t *testing.T) {

	a := &testAnalyzer{}
	a.Data = make(map[string]string)

	cfg := `
[DataExtract."Version__9"]
File = "/tmp/datatestfileX.1"
Json = "a.b.c.d.e.f"
`
	g := New(cfg, a)

	g.Start()

	a.testfile = "/tmp/datatestfileX.1"

	fi := makeFile(`{"a":{"b":{"c":{"d":{"e":{"f": "deep"}}}}}}`, "datatestfileX.1")
	err := g.CheckFile(&fi, "/tmp")
	if err != nil {
		t.Errorf("CheckFile failed")
	}
	if data, ok := a.Data["Version"]; !ok || data != "deep" {
		t.Errorf("data extract failed Json, %s", a.Data["Version"])
	}
	os.Remove("/tmp/datatestfileX.1")
	delete(a.Data, "Version")

	g.Finalize()
}

func TestJson6array(t *testing.T) {

	a := &testAnalyzer{}
	a.Data = make(map[string]string)

	cfg := `
[DataExtract."Version__9"]
File = "/tmp/datatestfileX.1"
Json = "a.0.c"
`
	g := New(cfg, a)

	g.Start()

	a.testfile = "/tmp/datatestfileX.1"

	fi := makeFile(`{"a":[{"c": true}]}`, "datatestfileX.1")
	err := g.CheckFile(&fi, "/tmp")
	if err != nil {
		t.Errorf("CheckFile failed")
	}
	if data, ok := a.Data["Version"]; !ok || data != "true" {
		t.Errorf("data extract failed Json")
	}
	os.Remove("/tmp/datatestfileX.1")
	delete(a.Data, "Version")

	g.Finalize()
}

func TestJsonContent(t *testing.T) {
	cfg := `
[GlobalConfig]
FsType = "dirfs"

[DataExtract."jsonfile.json"]
File = "/jsonfile.json"
RegEx = "(.*)\\n"
`
	analyzer := analyzer.NewFromConfig("../../../test/testdir", cfg)
	analyzer.AddAnalyzerPlugin(New(string(cfg), analyzer))
	analyzer.RunPlugins()

	report := analyzer.JsonReport()

	item, err := util.XtractJsonField([]byte(report), []string{"data", "jsonfile.json", "test_str"})
	if err != nil {
		t.Errorf("error %s", err)
	}
	if item != "yolo" {
		t.Errorf("data was not json encoded: %s", report)
	}

	_ = analyzer.CleanUp()
}
