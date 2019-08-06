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

package util

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"reflect"
	"strconv"
)

func MkTmpDir(prefix string) (string, error) {
	tmpDir, err := ioutil.TempDir(os.TempDir(), prefix)
	if err != nil {
		return "", err
	}
	return tmpDir, err
}

func DigestFileSha256(filepath string) []byte {
	f, err := os.Open(filepath)
	if err != nil {
		return nil
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return nil
	}

	return h.Sum(nil)
}

func loadJson(data []byte, item string) (interface{}, error) {
	var jd map[string]interface{}
	err := json.Unmarshal(data, &jd)
	return jd[item], err
}

func XtractJsonField(data []byte, items []string) (string, error) {
	idx := 0
	id, err := loadJson(data, items[idx])
	if err != nil {
		return "", err
	}
	if id == nil {
		return "", fmt.Errorf("JSON field not found: %s", items[idx])
	}
	idx++
	for {
		if id == nil {
			return "", fmt.Errorf("JSON field not found: %s", items[idx-1])
		}
		// keep for debugging
		//fmt.Printf("idx=%d, type=%s\n", idx, reflect.TypeOf(id).String())
		if reflect.TypeOf(id).String() == "map[string]interface {}" {
			idc := id.(map[string]interface{})
			id = idc[items[idx]]
			idx++
		} else if reflect.TypeOf(id).String() == "[]interface {}" {
			idc := id.([]interface{})
			index, _ := strconv.Atoi(items[idx])
			id = idc[index]
			idx++
		} else {
			switch id := id.(type) {
			case bool:
				if id {
					return "true", nil
				} else {
					return "false", nil
				}
			case float32, float64:
				return fmt.Sprintf("%f", id), nil
			case string:
				return id, nil
			default:
				return "", fmt.Errorf("can't handle type")
			}
		}
	}
}

func CleanPathDir(pathName string) string {
	cleaned := path.Clean(pathName)
	if cleaned[len(cleaned)-1] != '/' {
		cleaned += "/"
	}
	return cleaned
}
