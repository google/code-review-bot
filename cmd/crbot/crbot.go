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
	"strconv"
	"strings"

	"golang.org/x/oauth2"

	"github.com/google/code-review-bot/config"
	"github.com/google/code-review-bot/ghutil"
	"github.com/google/code-review-bot/logging"
)

func main() {
	configFileFlag := flag.String("config", "config.json", "Path to cfg file; required")
	claSignersFileFlag := flag.String("cla-signers", "cla.json", "Path to CLA signers, required")
	orgFlag := flag.String("org", "", "Name of organization or username; required if not set via cfg")
	repoFlag := flag.String("repo", "", "Name of repo; if empty, implies all repos in org")
	prFlag := flag.String("pr", "", "Comma-separated list of PRs to process")
	updateRepoFlag := flag.Bool("update-repo", false, "Update labels on the repo")
	flag.Parse()

	// Read and parse the general cfg file.
	cfg := config.ParseConfig(*configFileFlag)

	// Read and parse the CLA signers file.
	claSigners := config.ParseClaSigners(*claSignersFileFlag)

	// Get the org name from command-line flags or config file.
	var orgName string
	if *orgFlag != "" {
		orgName = *orgFlag
	} else if cfg.Org != "" {
		orgName = cfg.Org
	} else {
		logging.Fatalf("-org must be non-empty or `org` must be specified in cfg file")
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

	ctx := context.Background()

	// Configure authentication and connect to GitHub.
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: cfg.Auth},
	)
	tc := oauth2.NewClient(ctx, ts)

	// Process org and repo(s) specified on the command-line.
	ghc := ghutil.NewClient(tc)
	repoSpec := ghutil.GitHubProcessSpec{
		Org:        orgName,
		Repo:       repoName,
		Pulls:      prNumbers,
		UpdateRepo: *updateRepoFlag,
	}
	ghc.ProcessOrgRepo(ctx, repoSpec, claSigners)
}
