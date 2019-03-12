# Copyright 2017 Google Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

LEVEL = .
include $(LEVEL)/common.mk

go_test:
	$(VERB) echo
	$(VERB) echo "Running tests via 'go test' ..."
	$(VERB) go test -v ./...

gofmt_test:
	$(VERB) echo
	$(VERB) echo "Running 'go fmt' test ..."
	$(VERB) ./gofmt_test.sh

ghutil_test:
	$(VERB) echo
	$(VERB) echo "Running tests in 'ghutil' recursively ..."
	$(VERB) $(MAKE) VERBOSE=$(VERBOSE) -s -C ghutil test

test: go_test gofmt_test ghutil_test
