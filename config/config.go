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

// Config is the configuration for the `crbot` tool to specify the
// authentication it should use and the scope at which it should run, whether
// for all repos in a single organization, or a single specific repo.
type Config struct {
	Auth string `json:"auth" yaml:"auth"`
	Org  string `json:"org,omitempty" yaml:"org,omitempty"`
	Repo string `json:"repo,omitempty" yaml:"repo,omitempty"`
}

// Account represents a single user record, whether human or a bot, with a name,
// email, and GitHub login.
type Account struct {
	Name  string `json:"name" yaml:"name"`
	Email string `json:"email" yaml:"email"`
	Login string `json:"github" yaml:"github"`
}

// Company represents a company record with a name, (optional) domain name(s),
// and user accounts.
type Company struct {
	Name    string    `json:"name" yaml:"name"`
	Domains []string  `json:"domains,omitempty" yaml:"domains,omitempty"`
	People  []Account `json:"people" yaml:"people"`
}

// ClaSigners provides the overall structure of the CLA config: individual CLA
// signers, bots, and corporate CLA signers.
type ClaSigners struct {
	People    []Account `json:"people,omitempty" yaml:"people,omitempty"`
	Bots      []Account `json:"bots,omitempty" yaml:"bots,omitempty"`
	Companies []Company `json:"companies,omitempty" yaml:"companies,omitempty"`
}

// ParseConfig parses the config from a YAML or JSON file and returns a `Config`
// object.
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
		err = errors.New("unrecognized file type")
	}

	if err != nil {
		logging.Fatalf("Error parsing config file (%s): %s", filename, err)
	}

	return config
}

// ParseClaSigners parses the CLA signers config from a YAML or JSON file and
// returns a `ClaSigners` object.
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
		err = errors.New("unrecognized file type")
	}

	if err != nil {
		logging.Fatalf("Error parsing CLA Signers file (%s): %s", filename, err)
	}

	return claSigners
}
