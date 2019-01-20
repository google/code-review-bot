#!/bin/bash -u
#
# Copyright 2019 Google Inc.
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

# Verifies that we have run `go mod tidy` to keep our modules config clean.

declare -r GO_MOD="go.mod"
declare -r GO_SUM="go.sum"

declare -r GO_MOD_ORIG="go.mod.orig"
declare -r GO_SUM_ORIG="go.sum.orig"

declare -i success=0

cp "${GO_MOD}" "${GO_MOD_ORIG}"
cp "${GO_SUM}" "${GO_SUM_ORIG}"

go mod tidy

diff -u "${GO_MOD}" "${GO_MOD_ORIG}" || success=1
diff -u "${GO_SUM}" "${GO_SUM_ORIG}" || success=1

mv "${GO_MOD_ORIG}" "${GO_MOD}"
mv "${GO_SUM_ORIG}" "${GO_SUM}"

if [[ ${success} == 0 ]]; then
  echo PASSED
else
  echo FAILED
fi
exit ${success}
