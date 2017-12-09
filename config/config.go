// Copyright 2017 Google Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package config

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"strings"

	"github.com/go-yaml/yaml"
	"github.com/google/code-review-bot/logging"
)

type Config struct {
	Auth string `json:"auth" yaml:"auth"`
	Org  string `json:"org,omitempty" yaml:"org,omitempty"`
	Repo string `json:"repo,omitempty" yaml:"repo,omitempty"`
}

type Account struct {
	Name  string `json:"name" yaml:"name"`
	Email string `json:"email" yaml:"email"`
	Login string `json:"github" yaml:"github"`
}

type Company struct {
	Name    string    `json:"name" yaml:"name"`
	Domains []string  `json:"domains,omitempty" yaml:"domains,omitempty"`
	People  []Account `json:"people" yaml:"people"`
}

type ClaSigners struct {
	People    []Account `json:"people,omitempty" yaml:"people,omitempty"`
	Bots      []Account `json:"bots,omitempty" yaml:"bots,omitempty"`
	Companies []Company `json:"companies,omitempty" yaml:"companies,omitempty"`
}

func ParseConfig(filename string) Config {
	fileContents, err := ioutil.ReadFile(filename)
	if err != nil {
		logging.Fatalf("Error reading config file (%s): %s", filename, err)
	}

	var config Config
	if strings.HasSuffix(filename, ".json") {
		err = json.Unmarshal(fileContents, &config)
	} else if strings.HasSuffix(filename, ".yaml") || strings.HasSuffix(filename, ".yml") {
		err = yaml.Unmarshal(fileContents, &config)
	} else {
		err = errors.New("Unrecognized file type")
	}

	if err != nil {
		logging.Fatalf("Error parsing config file (%s): %s", filename, err)
	}

	return config
}

func ParseClaSigners(filename string) ClaSigners {
	fileContents, err := ioutil.ReadFile(filename)
	if err != nil {
		logging.Fatalf("Error reading CLA Signers file (%s): %s", filename, err)
	}

	var claSigners ClaSigners
	if strings.HasSuffix(filename, ".json") {
		err = json.Unmarshal(fileContents, &claSigners)
	} else if strings.HasSuffix(filename, ".yaml") || strings.HasSuffix(filename, ".yml") {
		err = yaml.Unmarshal(fileContents, &claSigners)
	} else {
		err = errors.New("Unrecognized file type")
	}

	if err != nil {
		logging.Fatalf("Error parsing CLA Signers file (%s): %s", filename, err)
	}

	return claSigners
}
