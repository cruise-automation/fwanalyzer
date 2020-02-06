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

package main

import (
	"io/ioutil"
	"strings"
	"testing"

	"github.com/cruise-automation/fwanalyzer/pkg/analyzer"
)

func TestMain(t *testing.T) {
	tests := []struct {
		inclFile string
		testFile string
		contains []string
	}{
		{
			`
[GlobalConfig]
FsType="dirfs"
# we can have comments
`,
			"/tmp/fwa_test_cfg_file.1",
			[]string{"GlobalConfig"},
		},
		{
			`
[Include."/tmp/fwa_test_cfg_file.1"]
[Test]
a = "a"
`,
			"/tmp/fwa_test_cfg_file.2",
			[]string{"Test"},
		},

		{
			`
[Include."/tmp/fwa_test_cfg_file.2"]
`,
			"/tmp/fwa_test_cfg_file.3",
			[]string{"Test", "GlobalConfig"},
		},
	}

	for _, test := range tests {
		err := ioutil.WriteFile(test.testFile, []byte(test.inclFile), 0644)
		if err != nil {
			t.Error(err)
		}
		cfg, err := readConfig(test.testFile, []string{})
		if err != nil {
			t.Error(err)
		}
		for _, c := range test.contains {
			if !strings.Contains(cfg, c) {
				t.Errorf("include didn't work")
			}
		}
		// this will panic if cfg contains an illegal config
		analyzer.NewFromConfig("dummy", cfg)
	}
}
