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
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"

	"github.com/BurntSushi/toml"

	"github.com/cruise-automation/fwanalyzer/pkg/analyzer"
	"github.com/cruise-automation/fwanalyzer/pkg/analyzer/dataextract"
	"github.com/cruise-automation/fwanalyzer/pkg/analyzer/dircontent"
	"github.com/cruise-automation/fwanalyzer/pkg/analyzer/filecmp"
	"github.com/cruise-automation/fwanalyzer/pkg/analyzer/filecontent"
	"github.com/cruise-automation/fwanalyzer/pkg/analyzer/filepathowner"
	"github.com/cruise-automation/fwanalyzer/pkg/analyzer/filestatcheck"
	"github.com/cruise-automation/fwanalyzer/pkg/analyzer/filetree"
	"github.com/cruise-automation/fwanalyzer/pkg/analyzer/globalfilechecks"
)

func readFileWithCfgPath(filepath string, cfgpath []string) (string, error) {
	for _, cp := range cfgpath {
		data, err := ioutil.ReadFile(path.Join(cp, filepath))
		if err == nil {
			return string(data), nil
		}
	}
	data, err := ioutil.ReadFile(filepath)
	return string(data), err
}

// read config file and parse Include statement reading all config files that are included
func readConfig(filepath string, cfgpath []string) (string, error) {
	cfg := ""
	cfgBytes, err := readFileWithCfgPath(filepath, cfgpath)
	cfg = string(cfgBytes)
	if err != nil {
		return cfg, err
	}

	type includeCfg struct {
		Include map[string]interface{}
	}

	var include includeCfg
	_, err = toml.Decode(cfg, &include)
	if err != nil {
		return cfg, err
	}
	for inc := range include.Include {
		incCfg, err := readConfig(inc, cfgpath)
		if err != nil {
			return cfg, err
		}
		cfg = cfg + incCfg
	}
	return cfg, nil
}

type arrayFlags []string

func (af *arrayFlags) String() string {
	return strings.Join(*af, " ")
}

func (af *arrayFlags) Set(value string) error {
	*af = append(*af, value)
	return nil
}

func main() {
	var cfgpath arrayFlags
	var in = flag.String("in", "", "filesystem image file or path to directory")
	var out = flag.String("out", "-", "output to file (use - for stdout)")
	var extra = flag.String("extra", "", "overwrite directory to read extra data from (filetree, cmpfile, ...)")
	var cfg = flag.String("cfg", "", "config file")
	flag.Var(&cfgpath, "cfgpath", "path to config file and included files (can be repated)")
	var errorExit = flag.Bool("ee", false, "exit with error if offenders are present")
	var invertMatch = flag.Bool("invertMatch", false, "invert RegEx Match")
	flag.Parse()

	if *in == "" || *cfg == "" {
		fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
		flag.PrintDefaults()
		os.Exit(1)
	}

	cfgdata, err := readConfig(*cfg, cfgpath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not read config file: %s, error: %s\n", *cfg, err)
		os.Exit(1)
	}

	// if no alternative extra data directory is given use the directory "config filepath"
	if *extra == "" {
		*extra = path.Dir(*cfg)
	}

	analyzer := analyzer.NewFromConfig(*in, string(cfgdata))

	supported, msg := analyzer.FsTypeSupported()
	if !supported {
		fmt.Fprintf(os.Stderr, "%s\n", msg)
		os.Exit(1)
	}

	analyzer.AddAnalyzerPlugin(globalfilechecks.New(string(cfgdata), analyzer))
	analyzer.AddAnalyzerPlugin(filecontent.New(string(cfgdata), analyzer, *invertMatch))
	analyzer.AddAnalyzerPlugin(filecmp.New(string(cfgdata), analyzer, *extra))
	analyzer.AddAnalyzerPlugin(dataextract.New(string(cfgdata), analyzer))
	analyzer.AddAnalyzerPlugin(dircontent.New(string(cfgdata), analyzer))
	analyzer.AddAnalyzerPlugin(filestatcheck.New(string(cfgdata), analyzer))
	analyzer.AddAnalyzerPlugin(filepathowner.New(string(cfgdata), analyzer))
	analyzer.AddAnalyzerPlugin(filetree.New(string(cfgdata), analyzer, *extra))

	analyzer.RunPlugins()

	report := analyzer.JsonReport()
	if *out == "" {
		fmt.Fprintln(os.Stderr, "Use '-' for stdout or provide a filename.")
	} else if *out == "-" {
		fmt.Println(report)
	} else {
		err := ioutil.WriteFile(*out, []byte(report), 0644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Can't write report to: %s, error: %s\n", *out, err)
		}
	}

	_ = analyzer.CleanUp()

	// signal offenders by providing a error exit code
	if *errorExit && analyzer.HasOffenders() {
		os.Exit(1)
	}
}
