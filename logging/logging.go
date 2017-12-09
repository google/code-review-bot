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

package logging

import (
	"fmt"
	"log"
	"os"
)

func Errorf(format string, a ...interface{}) (int, error) {
	return fmt.Fprintf(os.Stderr, format+"\n", a...)
}

func Error(a ...interface{}) (int, error) {
	return fmt.Fprintln(os.Stderr, a...)
}

func Info(a ...interface{}) (int, error) {
	return fmt.Println(a...)
}

func Infof(format string, a ...interface{}) (int, error) {
	return fmt.Printf(format+"\n", a...)
}

func Fatal(a ...interface{}) {
	log.Fatal(a...)
}

func Fatalf(format string, a ...interface{}) {
	log.Fatalf(format+"\n", a...)
}
