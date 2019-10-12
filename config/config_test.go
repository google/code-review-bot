// Copyright 2019 Google Inc.
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
	"testing"

	"github.com/go-yaml/yaml"
	"github.com/stretchr/testify/assert"
)

func parseClaSigners(t *testing.T, claYaml string, claSigners *ClaSigners) {
	err := yaml.Unmarshal([]byte(claYaml), claSigners)
	if err != nil {
		t.Logf("Error parsing YAML: %v", err)
		t.Fail()
	}
}

func TestParseClaSignersEmpty(t *testing.T) {
	var claSigners ClaSigners
	parseClaSigners(t, "", &claSigners)
	assert.Equal(t, 0, len(claSigners.People))
	assert.Equal(t, 0, len(claSigners.Bots))
	assert.Equal(t, 0, len(claSigners.Companies))
	assert.Nil(t, claSigners.External)
}

func TestParseClaSignersSimple(t *testing.T) {
	claYaml := `
people:
  - name: First Last
    email: first@example.com
    github: first-last
`
	var claSigners ClaSigners
	parseClaSigners(t, claYaml, &claSigners)
	assert.Equal(t, 1, len(claSigners.People), "Should have exactly 1 entry in the `people` section")
	person := claSigners.People[0]
	assert.Equal(t, "First Last", person.Name)
	assert.Equal(t, "first@example.com", person.Email)
	assert.Equal(t, "first-last", person.Login)

	assert.Equal(t, 0, len(claSigners.Bots))
	assert.Equal(t, 0, len(claSigners.Companies))
	assert.Nil(t, claSigners.External)
}

func TestParseClaSignersWithExternalNamed(t *testing.T) {
	claYaml := `
people:
  - name: First Last
    email: first@example.com
    github: first-last

external:
  people:
    - name: User Name
      email: user@name.example
      github: user-name
`
	var claSigners ClaSigners
	parseClaSigners(t, claYaml, &claSigners)
	assert.Equal(t, 1, len(claSigners.People), "Should have exactly 1 entry in the `people` section")
	person := claSigners.People[0]
	assert.Equal(t, "First Last", person.Name)
	assert.Equal(t, "first@example.com", person.Email)
	assert.Equal(t, "first-last", person.Login)

	assert.Equal(t, 0, len(claSigners.Bots))
	assert.Equal(t, 0, len(claSigners.Companies))
	assert.NotNil(t, claSigners.External)

	external := claSigners.External
	assert.Equal(t, 1, len(external.People), "Should have exactly 1 entry in the external `people` section")
	extPerson := external.People[0]
	assert.Equal(t, "User Name", extPerson.Name)
	assert.Equal(t, "user@name.example", extPerson.Email)
	assert.Equal(t, "user-name", extPerson.Login)
	assert.Equal(t, 0, len(external.Bots))
	assert.Equal(t, 0, len(external.Companies))
}
