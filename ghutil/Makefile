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

LEVEL = ..
include $(LEVEL)/common.mk

SRC = ghutil.go
MOCK = mock_ghutil.go
HEADER = mock_header.txt

mock: $(MOCK)

$(MOCK): $(SRC) $(HEADER)
	$(VERB) (cat $(HEADER) && mockgen -source $(SRC) -package ghutil) > $(MOCK)

test:
	$(VERB) echo "mockgen diff test (if no output, then ok)"
	$(VERB) (cat $(HEADER) && mockgen -source $(SRC) -package ghutil) | diff -u $(MOCK) -
