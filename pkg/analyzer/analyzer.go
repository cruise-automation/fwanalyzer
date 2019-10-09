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
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io/ioutil"
	"os"
	"path"
	"strings"

	"github.com/BurntSushi/toml"

	"github.com/cruise-automation/fwanalyzer/pkg/dirparser"
	"github.com/cruise-automation/fwanalyzer/pkg/extparser"
	"github.com/cruise-automation/fwanalyzer/pkg/fsparser"
	"github.com/cruise-automation/fwanalyzer/pkg/squashfsparser"
	"github.com/cruise-automation/fwanalyzer/pkg/ubifsparser"
	"github.com/cruise-automation/fwanalyzer/pkg/util"
	"github.com/cruise-automation/fwanalyzer/pkg/vfatparser"
)

type AnalyzerPluginType interface {
	Name() string
	Start()
	Finalize() string
	CheckFile(fi *fsparser.FileInfo, path string) error
}

type AnalyzerType interface {
	GetFileInfo(filepath string) (fsparser.FileInfo, error)
	RemoveFile(filepath string) error
	FileGetSha256(filepath string) ([]byte, error)
	FileGet(filepath string) (string, error)
	AddOffender(filepath string, reason string)
	AddInformational(filepath string, reason string)
	CheckAllFilesWithPath(cb AllFilesCallback, cbdata AllFilesCallbackData, filepath string)
	AddData(key, value string)
	ImageInfo() AnalyzerReport
}

type AllFilesCallbackData interface{}
type AllFilesCallback func(fi *fsparser.FileInfo, fullpath string, data AllFilesCallbackData)

type globalConfigType struct {
	FSType        string
	FSTypeOptions string
	DigestImage   bool
}

type AnalyzerReport struct {
	FSType        string                   `json:"fs_type"`
	ImageName     string                   `json:"image_name"`
	ImageDigest   string                   `json:"image_digest,omitempty"`
	Data          map[string]interface{}   `json:"data,omitempty"`
	Offenders     map[string][]interface{} `json:"offenders,omitempty"`
	Informational map[string][]interface{} `json:"informational,omitempty"`
}

type Analyzer struct {
	fsparser      fsparser.FsParser
	tmpdir        string
	config        globalConfigType
	analyzers     []AnalyzerPluginType
	PluginReports map[string]interface{}
	AnalyzerReport
}

func New(fsp fsparser.FsParser, cfg globalConfigType) *Analyzer {
	var a Analyzer
	a.config = cfg
	a.fsparser = fsp
	a.FSType = cfg.FSType
	a.ImageName = fsp.ImageName()
	a.tmpdir, _ = util.MkTmpDir("analyzer")
	a.Offenders = make(map[string][]interface{})
	a.Informational = make(map[string][]interface{})
	a.Data = make(map[string]interface{})
	a.PluginReports = make(map[string]interface{})

	if cfg.DigestImage {
		a.ImageDigest = hex.EncodeToString(util.DigestFileSha256(a.ImageName))
	}

	return &a
}

func NewFromConfig(imagepath string, cfgdata string) *Analyzer {
	type globalconfig struct {
		GlobalConfig globalConfigType
	}
	var config globalconfig

	_, err := toml.Decode(cfgdata, &config)
	if err != nil {
		panic("can't read config data: " + err.Error())
	}

	var fsp fsparser.FsParser
	// Set the parser based on the FSType in the config
	if strings.EqualFold(config.GlobalConfig.FSType, "extfs") {
		fsp = extparser.New(imagepath, config.GlobalConfig.FSTypeOptions == "selinux")
	} else if strings.EqualFold(config.GlobalConfig.FSType, "dirfs") {
		fsp = dirparser.New(imagepath)
	} else if strings.EqualFold(config.GlobalConfig.FSType, "vfatfs") {
		fsp = vfatparser.New(imagepath)
	} else if strings.EqualFold(config.GlobalConfig.FSType, "squashfs") {
		fsp = squashfsparser.New(imagepath)
	} else if strings.EqualFold(config.GlobalConfig.FSType, "ubifs") {
		fsp = ubifsparser.New(imagepath)
	} else {
		panic("Cannot find an appropriate parser: " + config.GlobalConfig.FSType)
	}

	return New(fsp, config.GlobalConfig)
}

func (a *Analyzer) FsTypeSupported() (bool, string) {
	if !a.fsparser.Supported() {
		return false, a.config.FSType + ": requires additional tools, please refer to documentation."
	}
	return true, ""
}

func (a *Analyzer) ImageInfo() AnalyzerReport {
	// only provide the meta information, don't include offenders and other report data
	return AnalyzerReport{
		FSType:      a.FSType,
		ImageName:   a.ImageName,
		ImageDigest: a.ImageDigest,
	}
}

func (a *Analyzer) AddAnalyzerPlugin(aplug AnalyzerPluginType) {
	a.analyzers = append(a.analyzers, aplug)
}

func (a *Analyzer) iterateFiles(curpath string) error {
	dir, err := a.fsparser.GetDirInfo(curpath)
	if err != nil {
		return err
	}
	cp := curpath
	for _, fi := range dir {
		for _, ap := range a.analyzers {
			err = ap.CheckFile(&fi, cp)
			if err != nil {
				return err
			}
		}

		if fi.IsDir() {
			err = a.iterateFiles(path.Join(curpath, fi.Name))
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (a *Analyzer) checkRoot() error {
	fi, err := a.fsparser.GetFileInfo("/")
	if err != nil {
		return err
	}

	for _, ap := range a.analyzers {
		err = ap.CheckFile(&fi, "/")
		if err != nil {
			return err
		}
	}

	return nil
}

func (a *Analyzer) addPluginReport(report string) {
	var data map[string]interface{}

	err := json.Unmarshal([]byte(report), &data)
	if err != nil {
		return
	}
	for k := range data {
		a.PluginReports[k] = data[k]
	}
}

func (a *Analyzer) RunPlugins() {
	for _, ap := range a.analyzers {
		ap.Start()
	}

	err := a.checkRoot()
	if err != nil {
		panic("RunPlugins error: " + err.Error())
	}

	err = a.iterateFiles("/")
	if err != nil {
		panic("RunPlugins error: " + err.Error())
	}

	for _, ap := range a.analyzers {
		res := ap.Finalize()
		a.addPluginReport(res)
	}
}

func (a *Analyzer) CleanUp() error {
	err := os.RemoveAll(a.tmpdir)
	return err
}

func (a *Analyzer) GetFileInfo(filepath string) (fsparser.FileInfo, error) {
	return a.fsparser.GetFileInfo(filepath)
}

func (a *Analyzer) FileGet(filepath string) (string, error) {
	tmpfile, _ := ioutil.TempFile(a.tmpdir, "")
	tmpname := tmpfile.Name()
	tmpfile.Close()
	if a.fsparser.CopyFile(filepath, tmpname) {
		return tmpname, nil
	}
	return "", errors.New("error copying file")
}

func (a *Analyzer) FileGetSha256(filepath string) ([]byte, error) {
	tmpname, err := a.FileGet(filepath)
	if err != nil {
		return nil, err
	}

	defer os.Remove(tmpname)
	digest := util.DigestFileSha256(tmpname)
	return digest, nil
}

func (a *Analyzer) RemoveFile(filepath string) error {
	os.Remove(filepath)
	return nil
}

func (a *Analyzer) iterateAllDirs(curpath string, cb AllFilesCallback, cbdata AllFilesCallbackData) error {
	dir, err := a.fsparser.GetDirInfo(curpath)
	if err != nil {
		return err
	}
	for _, fi := range dir {
		cb(&fi, curpath, cbdata)
		if fi.IsDir() {
			err := a.iterateAllDirs(path.Join(curpath, fi.Name), cb, cbdata)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (a *Analyzer) CheckAllFilesWithPath(cb AllFilesCallback, cbdata AllFilesCallbackData, filepath string) {
	if cb == nil {
		return
	}
	err := a.iterateAllDirs(filepath, cb, cbdata)
	if err != nil {
		panic("iterateAllDirs failed")
	}
}

func (a *Analyzer) AddOffender(filepath string, reason string) {
	var data map[string]interface{}
	// this is valid json?
	if err := json.Unmarshal([]byte(reason), &data); err == nil {
		// yes: store as json
		a.Offenders[filepath] = append(a.Offenders[filepath], json.RawMessage(reason))
	} else {
		// no: store as plain text
		a.Offenders[filepath] = append(a.Offenders[filepath], reason)
	}
}

func (a *Analyzer) AddInformational(filepath string, reason string) {
	var data map[string]interface{}
	// this is valid json?
	if err := json.Unmarshal([]byte(reason), &data); err == nil {
		// yes: store as json
		a.Informational[filepath] = append(a.Informational[filepath], json.RawMessage(reason))
	} else {
		// no: store as plain text
		a.Informational[filepath] = append(a.Informational[filepath], reason)
	}
}

func (a *Analyzer) HasOffenders() bool {
	return len(a.Offenders) > 0
}

func (a *Analyzer) AddData(key string, value string) {
	// this is a valid json object?
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(value), &data); err == nil {
		// yes: store as json
		a.Data[key] = json.RawMessage(value)
		return
	}

	// this is valid json array?
	var array []interface{}
	if err := json.Unmarshal([]byte(value), &array); err == nil {
		// yes: store as json
		a.Data[key] = json.RawMessage(value)
	} else {
		// no: store as plain text
		a.Data[key] = value
	}
}

func (a *Analyzer) addReportData(report []byte) ([]byte, error) {
	var data map[string]interface{}

	err := json.Unmarshal(report, &data)
	if err != nil {
		return report, err
	}

	for k := range a.PluginReports {
		data[k] = a.PluginReports[k]
	}

	jdata, err := json.Marshal(&data)
	return jdata, err
}

func (a *Analyzer) JsonReport() string {
	ar := AnalyzerReport{
		FSType:        a.FSType,
		Offenders:     a.Offenders,
		Informational: a.Informational,
		Data:          a.Data,
		ImageName:     a.ImageName,
		ImageDigest:   a.ImageDigest,
	}

	jdata, _ := json.Marshal(ar)
	jdata, _ = a.addReportData(jdata)

	// make json look pretty
	var prettyJson bytes.Buffer
	_ = json.Indent(&prettyJson, jdata, "", "\t")
	return prettyJson.String()
}
