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

package ghutil_test

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"github.com/google/code-review-bot/config"
	"github.com/google/code-review-bot/ghutil"
	"github.com/google/go-github/v21/github"
)

type MockGitHubClient struct {
	Organizations *ghutil.MockOrganizationsService
	PullRequests  *ghutil.MockPullRequestsService
	Issues        *ghutil.MockIssuesService
	Repositories  *ghutil.MockRepositoriesService
}

func NewMockGitHubClient(ghc *ghutil.GitHubClient, ctrl *gomock.Controller) *MockGitHubClient {
	mockGhc := &MockGitHubClient{
		Organizations: ghutil.NewMockOrganizationsService(ctrl),
		PullRequests:  ghutil.NewMockPullRequestsService(ctrl),
		Issues:        ghutil.NewMockIssuesService(ctrl),
		Repositories:  ghutil.NewMockRepositoriesService(ctrl),
	}

	// Patch the original GitHubClient with our mock services.
	ghc.Organizations = mockGhc.Organizations
	ghc.PullRequests = mockGhc.PullRequests
	ghc.Issues = mockGhc.Issues
	ghc.Repositories = mockGhc.Repositories

	return mockGhc
}

// Common parameters used across most, if not all, tests.
var (
	ctrl    *gomock.Controller
	ghc     *ghutil.GitHubClient
	mockGhc *MockGitHubClient

	noLabel *github.Label = nil
)

const (
	orgName  = "org"
	repoName = "repo"
)

func setUp(t *testing.T) {
	ctrl = gomock.NewController(t)
	ghc = &ghutil.GitHubClient{}
	mockGhc = NewMockGitHubClient(ghc, ctrl)
}

func tearDown(_ *testing.T) {
	ctrl.Finish()
}

func TestGetAllRepos_OrgAndRepo(t *testing.T) {
	setUp(t)
	defer tearDown(t)

	repo := github.Repository{}
	ctx := context.Background()

	mockGhc.Repositories.EXPECT().Get(ctx, orgName, repoName).Return(&repo, nil, nil)

	repos := ghc.GetAllRepos(ctx, orgName, repoName)
	assert.Equal(t, 1, len(repos), "repos is not of length 1: %v", repos)
}

func TestGetAllRepos_OrgOnly(t *testing.T) {
	setUp(t)
	defer tearDown(t)

	expectedRepos := []*github.Repository{
		{},
		{},
	}
	ctx := context.Background()

	mockGhc.Repositories.EXPECT().List(ctx, orgName, nil).Return(expectedRepos, nil, nil)

	actualRepos := ghc.GetAllRepos(ctx, orgName, "")
	assert.Equal(t, len(expectedRepos), len(actualRepos), "Expected repos: %v, actual repos: %v", expectedRepos, actualRepos)
}

func TestVerifyRepoHasClaLabels_NoLabels(t *testing.T) {
	setUp(t)
	defer tearDown(t)

	ctx := context.Background()

	mockGhc.Issues.EXPECT().GetLabel(ctx, orgName, repoName, ghutil.LabelClaYes).Return(noLabel, nil, nil)
	mockGhc.Issues.EXPECT().GetLabel(ctx, orgName, repoName, ghutil.LabelClaNo).Return(noLabel, nil, nil)

	assert.False(t, ghc.VerifyRepoHasClaLabels(ctx, orgName, repoName))
}

func TestVerifyRepoHasClaLabels_HasYesOnly(t *testing.T) {
	setUp(t)
	defer tearDown(t)

	label := github.Label{}

	ctx := context.Background()

	mockGhc.Issues.EXPECT().GetLabel(ctx, orgName, repoName, ghutil.LabelClaYes).Return(&label, nil, nil)
	mockGhc.Issues.EXPECT().GetLabel(ctx, orgName, repoName, ghutil.LabelClaNo).Return(noLabel, nil, nil)

	assert.False(t, ghc.VerifyRepoHasClaLabels(ctx, orgName, repoName))
}

func TestVerifyRepoHasClaLabels_HasNoOnly(t *testing.T) {
	setUp(t)
	defer tearDown(t)

	label := github.Label{}

	ctx := context.Background()

	mockGhc.Issues.EXPECT().GetLabel(ctx, orgName, repoName, ghutil.LabelClaYes).Return(noLabel, nil, nil)
	mockGhc.Issues.EXPECT().GetLabel(ctx, orgName, repoName, ghutil.LabelClaNo).Return(&label, nil, nil)

	assert.False(t, ghc.VerifyRepoHasClaLabels(ctx, orgName, repoName))
}

func TestVerifyRepoHasClaLabels_YesAndNoLabels(t *testing.T) {
	setUp(t)
	defer tearDown(t)

	labelYes := github.Label{}
	labelNo := github.Label{}
	ctx := context.Background()

	mockGhc.Issues.EXPECT().GetLabel(ctx, orgName, repoName, ghutil.LabelClaYes).Return(&labelYes, nil, nil)
	mockGhc.Issues.EXPECT().GetLabel(ctx, orgName, repoName, ghutil.LabelClaNo).Return(&labelNo, nil, nil)

	assert.True(t, ghc.VerifyRepoHasClaLabels(ctx, orgName, repoName))
}

func TestMatchAccount_MatchesCase(t *testing.T) {
	setUp(t)
	defer tearDown(t)

	// Credentials as provided by the user.
	account := config.Account{
		Name:  "Jane Doe",
		Email: "jane@example.com",
		Login: "JaneDoe",
	}

	// CLA as configured by the project.
	accounts := []config.Account{
		{
			Name:  "Jane Doe",
			Email: "jane@example.com",
			Login: "JaneDoe",
		},
	}

	assert.True(t, ghutil.MatchAccount(account, accounts))
}

func TestMatchAccount_DoesNotMatchCase(t *testing.T) {
	setUp(t)
	defer tearDown(t)

	// Credentials as provided by the user.
	account := config.Account{
		Name:  "Jane Doe",
		Email: "Jane@example.com",
		Login: "janedoe",
	}

	// CLA as configured by the project.
	accounts := []config.Account{
		{
			Name:  "Jane Doe",
			Email: "jane@example.com",
			Login: "JaneDoe",
		},
	}

	assert.True(t, ghutil.MatchAccount(account, accounts))
}

func TestDifferentAuthorAndCommitter(t *testing.T) {
	setUp(t)
	defer tearDown(t)

	sha := "12345abcde"
	name := "John Doe"
	corporateEmail := "john@github.com"
	personalEmail := "john@gmail.com"
	login := "johndoe"

	claSigners := config.ClaSigners{
		Companies: []config.Company{
			{
				Name: "Acme Inc.",
				People: []config.Account{
					{
						// Corporate account
						Name:  name,
						Email: corporateEmail,
						Login: login,
					},
					{
						// Personal account
						Name:  name,
						Email: personalEmail,
						Login: login,
					},
				},
			},
		},
	}
	commit := github.RepositoryCommit{
		SHA: &sha,
		Commit: &github.Commit{
			Author: &github.CommitAuthor{
				Name:  &name,
				Email: &corporateEmail,
			},
			Committer: &github.CommitAuthor{
				Name:  &name,
				Email: &personalEmail,
			},
		},
		Author: &github.User{
			Login: &login,
		},
		Committer: &github.User{
			Login: &login,
		},
	}

	commitIsCompliant, commitNonComplianceReason := ghutil.ProcessCommit(&commit, claSigners)
	assert.True(t, commitIsCompliant, "Commit should have been marked compliant; reason: ", commitNonComplianceReason)
}

func TestCanonicalizeEmail_Gmail(t *testing.T) {
	setUp(t)
	defer tearDown(t)

	var goldenInputOutput = map[string]string{
		"username@gmail.com":         "username@gmail.com",
		"user.name@gmail.com":        "username@gmail.com",
		"UserName@Gmail.Com":         "username@gmail.com",
		"User.Name@Gmail.Com":        "username@gmail.com",
		"U.s.e.r.N.a.m.e.@Gmail.Com": "username@gmail.com",
	}

	for input, expected := range goldenInputOutput {
		assert.Equal(t, expected, ghutil.CanonicalizeEmail(input))
	}
}

func TestGmailAddress_PeriodsDoNotMatchCLA(t *testing.T) {
	setUp(t)
	defer tearDown(t)

	var accountVsCla = map[string]string{
		"jane.doe@gmail.com":      "janedoe@gmail.com",
		"JaneDoe@gmail.com":       "Jane.Doe@gmail.com",
		"janeDoe@gmail.com":       "JaneDoe@gmail.com",
		"jane.doe@googlemail.com": "janedoe@googlemail.com",
		"JaneDoe@googlemail.com":  "Jane.Doe@googlemail.com",
		"janeDoe@googlemail.com":  "JaneDoe@googlemail.com",
	}

	for acctEmail, claEmail := range accountVsCla {
		// Credentials as provided by the user.
		account := config.Account{
			Name:  "Jane Doe",
			Email: acctEmail,
			Login: "janedoe",
		}

		// CLA as configured by the project.
		accounts := []config.Account{
			{
				Name:  "Jane Doe",
				Email: claEmail,
				Login: "janedoe",
			},
		}

		assert.True(t, ghutil.MatchAccount(account, accounts))
	}
}
