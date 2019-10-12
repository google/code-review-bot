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

// Secrets contains the authentication credentials for interacting with GitHub.
type Secrets struct {
	Auth string `json:"auth" yaml:"auth"`
}

// Config is the configuration for the `crbot` tool to specify the scope at
// which it should run, whether for all repos in a single organization, or a
// single specific repo.
type Config struct {
	Org               string `json:"org,omitempty" yaml:"org,omitempty"`
	Repo              string `json:"repo,omitempty" yaml:"repo,omitempty"`
	UnknownAsExternal bool   `json:"unknown_as_external,omitempty" yaml:"unknown_as_external,omitempty"`
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

// ExternalClaSigners represents CLA signers managed by an external process,
// i.e., not covered by this tool. This is useful for handling migrations into
// or out of the system provided by Code Review Bot.
type ExternalClaSigners struct {
	People    []Account `json:"people,omitempty" yaml:"people,omitempty"`
	Bots      []Account `json:"bots,omitempty" yaml:"bots,omitempty"`
	Companies []Company `json:"companies,omitempty" yaml:"companies,omitempty"`
}

// ClaSigners provides the overall structure of the CLA config: individual CLA
// signers, bots, and corporate CLA signers.
type ClaSigners struct {
	People    []Account           `json:"people,omitempty" yaml:"people,omitempty"`
	Bots      []Account           `json:"bots,omitempty" yaml:"bots,omitempty"`
	Companies []Company           `json:"companies,omitempty" yaml:"companies,omitempty"`
	External  *ExternalClaSigners `json:"external,omitempty" yaml:"external,omitempty"`
}

// parseFile is a helper method for parsing any of the YAML or JSON files we
// need to load: secrets, config, or CLA signers.
func parseFile(filetype string, filename string, data interface{}) {
	fileContents, err := ioutil.ReadFile(filename)
	if err != nil {
		logging.Fatalf("Error reading %s file '%s': %s", filetype, filename, err)
	}

	if strings.HasSuffix(filename, ".json") {
		err = json.Unmarshal(fileContents, data)
	} else if strings.HasSuffix(filename, ".yaml") || strings.HasSuffix(filename, ".yml") {
		err = yaml.Unmarshal(fileContents, data)
	} else {
		err = errors.New("unsupported file type; accepted: *.json, *.yaml, *.yml")
	}

	if err != nil {
		logging.Fatalf("Error parsing %s file '%s': %s", filetype, filename, err)
	}
}

// ParseSecretes parses the secrets (including auth tokens) from a YAML or JSON file.
func ParseSecrets(filename string) Secrets {
	var secrets Secrets
	parseFile("secrets", filename, &secrets)
	return secrets
}

// ParseConfig parses the config from a YAML or JSON file.
func ParseConfig(filename string) Config {
	var config Config
	// This config file is optional, so we shouldn't fail if the filename
	// is an empty string, but just return an uninitialized Config struct.
	if filename != "" {
		parseFile("config", filename, &config)
	}
	return config
}

// ParseClaSigners parses the CLA signers config from a YAML or JSON file.
func ParseClaSigners(filename string) ClaSigners {
	var claSigners ClaSigners
	parseFile("CLA signers", filename, &claSigners)
	return claSigners
}
