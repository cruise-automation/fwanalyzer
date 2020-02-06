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

package dircontent

import (
	"fmt"
	"path"

	"github.com/BurntSushi/toml"
	"github.com/bmatcuk/doublestar"

	"github.com/cruise-automation/fwanalyzer/pkg/analyzer"
	"github.com/cruise-automation/fwanalyzer/pkg/fsparser"
)

type dirContentType struct {
	Path     string          // path of directory to check
	Allowed  []string        // list of files that are allowed to be there
	Required []string        // list of files that must be there
	found    map[string]bool // whether or not there was a match for this file
}

type dirContentCheckType struct {
	dirs map[string]dirContentType
	a    analyzer.AnalyzerType
}

func addTrailingSlash(path string) string {
	if path[len(path)-1] != '/' {
		return path + "/"
	}
	return path
}

func validateItem(item dirContentType) bool {
	// ensure items in the Allowed/Required lists are valid for doublestar.Match()
	for _, allowed := range item.Allowed {
		_, err := doublestar.Match(allowed, "")
		if err != nil {
			return false
		}
	}
	for _, required := range item.Required {
		_, err := doublestar.Match(required, "")
		if err != nil {
			return false
		}
	}
	return true
}

func New(config string, a analyzer.AnalyzerType) *dirContentCheckType {
	type dirCheckListType struct {
		DirContent map[string]dirContentType
	}

	cfg := dirContentCheckType{a: a, dirs: make(map[string]dirContentType)}

	var dec dirCheckListType
	_, err := toml.Decode(config, &dec)
	if err != nil {
		panic("can't read config data: " + err.Error())
	}

	for name, item := range dec.DirContent {
		if !validateItem(item) {
			a.AddOffender(name, "invalid DirContent entry")
		}
		item.Path = addTrailingSlash(name)
		if _, ok := cfg.dirs[item.Path]; ok {
			a.AddOffender(name, "only one DirContent is allowed per path")
		}
		item.found = make(map[string]bool)
		for _, req := range item.Required {
			item.found[req] = false
		}
		cfg.dirs[item.Path] = item
	}

	return &cfg
}

func (state *dirContentCheckType) Start() {}

func (state *dirContentCheckType) Finalize() string {
	for _, item := range state.dirs {
		for fn, found := range item.found {
			if !found {
				state.a.AddOffender(fn, fmt.Sprintf("DirContent: required file %s not found in directory %s", fn, item.Path))
			}
		}
	}
	return ""
}

func (state *dirContentCheckType) Name() string {
	return "DirContent"
}

func (state *dirContentCheckType) CheckFile(fi *fsparser.FileInfo, dirpath string) error {
	dp := addTrailingSlash(dirpath)

	item, ok := state.dirs[dp]
	if !ok {
		return nil
	}
	found := false
	for _, fn := range item.Allowed {
		// allow globs for Allowed
		m, err := doublestar.Match(fn, fi.Name)
		if err != nil {
			// shouldn't happen because we check these in validateItem()
			return err
		}
		if m {
			found = true
		}
	}

	for _, fn := range item.Required {
		m, err := doublestar.Match(fn, fi.Name)
		if err != nil {
			return err
		}
		if m {
			item.found[fn] = true
			found = true
		}
	}

	if !found {
		state.a.AddOffender(path.Join(dirpath, fi.Name), fmt.Sprintf("DirContent: File %s not allowed in directory %s", fi.Name, dirpath))
	}
	return nil
}
