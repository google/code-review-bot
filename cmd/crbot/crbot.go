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

package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path"
	"strconv"
	"strings"

	"golang.org/x/oauth2"

	"github.com/google/code-review-bot/config"
	"github.com/google/code-review-bot/ghutil"
	"github.com/google/code-review-bot/logging"
)

func main() {
	secretsFileFlag := flag.String("secrets", "", "Path to secrets file; required")
	configFileFlag := flag.String("config", "", "Path to config file; optional")
	claSignersFileFlag := flag.String("cla-signers", "", "Path to CLA signers; required")
	orgFlag := flag.String("org", "", "Name of organization or username; required if not set in config file")
	repoFlag := flag.String("repo", "", "Name of repo; if empty, implies all repos in org")
	prFlag := flag.String("pr", "", "Comma-separated list of PRs to process")
	updateRepoFlag := flag.Bool("update-repo", false, "Update labels on the repo")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Syntax: %s [flags]\n\nFlags:\n", path.Base(os.Args[0]))
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nNote: -cla-signers, -config and -secrets accept YAML and JSON files.\n")
	}

	flag.Parse()

	if *secretsFileFlag == "" {
		logging.Fatalf("-secrets flag is required")
	} else if *claSignersFileFlag == "" {
		logging.Fatalf("-cla-signers flag is required")
	}

	// Read and parse required auth, config, and CLA signers files.
	secrets := config.ParseSecrets(*secretsFileFlag)
	cfg := config.ParseConfig(*configFileFlag)
	claSigners := config.ParseClaSigners(*claSignersFileFlag)

	// Get the org name from command-line flags or config file.
	var orgName string
	if *orgFlag != "" {
		orgName = *orgFlag
	} else if cfg.Org != "" {
		orgName = cfg.Org
	} else {
		logging.Fatalf("-org must be non-empty or `org` must be specified in config file")
	}

	// Get the repo name from command-line flags or config file.
	repoName := *repoFlag
	if repoName == "" {
		repoName = cfg.Repo
	}

	prNumbers := make([]uint64, 0)
	if *prFlag != "" {
		prElements := strings.Split(*prFlag, ",")
		prNumbers := make([]uint64, len(prElements))
		for idx, elt := range prElements {
			num, err := strconv.ParseUint(elt, 10, 32)
			if err != nil {
				logging.Fatalf("Invalid value for flag -pr: %s", *prFlag)
			}
			prNumbers[idx] = num
		}
	}

	// Configure authentication and connect to GitHub.
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: secrets.Auth},
	)
	tc := oauth2.NewClient(context.Background(), ts)

	// Process org and repo(s) specified on the command-line.
	ghc := ghutil.NewClient(tc)
	repoSpec := ghutil.GitHubProcessSpec{
		Org:        orgName,
		Repo:       repoName,
		Pulls:      prNumbers,
		UpdateRepo: *updateRepoFlag,
	}
	ghc.ProcessOrgRepo(ghc, repoSpec, claSigners)
}
