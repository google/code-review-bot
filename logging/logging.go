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

// Errorf outputs an error log line with a formatting string.
func Errorf(format string, a ...interface{}) (int, error) {
	return fmt.Fprintf(os.Stderr, format+"\n", a...)
}

// Error outputs an error log line without a formatting string.
func Error(a ...interface{}) (int, error) {
	return fmt.Fprintln(os.Stderr, a...)
}

// Infof outputs an info log line with a formatting string.
func Infof(format string, a ...interface{}) (int, error) {
	return fmt.Printf(format+"\n", a...)
}

// Info outputs an info log line without a formatting string.
func Info(a ...interface{}) (int, error) {
	return fmt.Println(a...)
}

// Fatalf outputs a fatal log line with a formatting string.
func Fatalf(format string, a ...interface{}) {
	log.Fatalf(format+"\n", a...)
}

// Fatal outputs a fatal log line without a formatting string.
func Fatal(a ...interface{}) {
	log.Fatal(a...)
}
