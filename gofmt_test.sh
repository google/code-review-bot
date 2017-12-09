#!/bin/bash -u
#
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

# Verifies that all *.go files are formatted according to `gofmt`.

declare -r VERBOSE="${VERBOSE:-}"

declare -i global_status=0
declare -i local_status=0

declare -i num_files_passed=0
declare -i num_files_failed=0

for gosrc in `find . -name \*\.go`; do
  if [[ "${VERBOSE}" -eq 1 ]]; then
    diff -u "${gosrc}" <(gofmt "${gosrc}")
  else
    diff -u "${gosrc}" <(gofmt "${gosrc}") > /dev/null 2>&1
  fi
  local_status=$?
  if [[ ${local_status} != 0 ]]; then
    echo "failed: ${gosrc}"
    global_status=${local_status}
    num_files_failed=$((num_files_failed + 1))
  else
    num_files_passed=$((num_files_passed + 1))
  fi
done

echo "gofmt files passed: ${num_files_passed} / $((num_files_passed + num_files_failed))"
exit ${global_status}
