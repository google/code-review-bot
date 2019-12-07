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
	"errors"
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
	Api           *ghutil.MockGitHubUtilApi
}

func NewMockGitHubClient(ghc *ghutil.GitHubClient, ctrl *gomock.Controller) *MockGitHubClient {
	mockGhc := &MockGitHubClient{
		Organizations: ghutil.NewMockOrganizationsService(ctrl),
		PullRequests:  ghutil.NewMockPullRequestsService(ctrl),
		Issues:        ghutil.NewMockIssuesService(ctrl),
		Repositories:  ghutil.NewMockRepositoriesService(ctrl),
		Api:           ghutil.NewMockGitHubUtilApi(ctrl),
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
	any                   = gomock.Any()
)

const (
	orgName    = "org"
	repoName   = "repo"
	pullNumber = 42
)

func setUp(t *testing.T) {
	ctrl = gomock.NewController(t)
	ghc = ghutil.NewBasicClient()
	mockGhc = NewMockGitHubClient(ghc, ctrl)
}

func tearDown(_ *testing.T) {
	ctrl.Finish()
}

func TestGetAllRepos_OrgAndRepo(t *testing.T) {
	setUp(t)
	defer tearDown(t)

	repo := github.Repository{}

	mockGhc.Repositories.EXPECT().Get(any, orgName, repoName).Return(&repo, nil, nil)

	repos := ghc.GetAllRepos(ghc, orgName, repoName)
	assert.Equal(t, 1, len(repos), "repos is not of length 1: %v", repos)
}

func TestGetAllRepos_OrgOnly(t *testing.T) {
	setUp(t)
	defer tearDown(t)

	expectedRepos := []*github.Repository{
		{},
		{},
	}

	mockGhc.Repositories.EXPECT().List(any, orgName, nil).Return(expectedRepos, nil, nil)

	actualRepos := ghc.GetAllRepos(ghc, orgName, "")
	assert.Equal(t, len(expectedRepos), len(actualRepos), "Expected repos: %v, actual repos: %v", expectedRepos, actualRepos)
}

func expectRepoLabels(orgName string, repoName string, hasYes bool, hasNo bool, hasExternal bool) {
	labels := map[string]bool{
		ghutil.LabelClaYes:      hasYes,
		ghutil.LabelClaNo:       hasNo,
		ghutil.LabelClaExternal: hasExternal,
	}
	for label, exists := range labels {
		var ghLabel *github.Label
		if exists {
			ghLabel = &github.Label{}
		}
		mockGhc.Issues.EXPECT().GetLabel(any, orgName, repoName, label).Return(ghLabel, nil, nil)
	}
}

func TestVerifyRepoHasClaLabels_NoLabels(t *testing.T) {
	setUp(t)
	defer tearDown(t)

	expectRepoLabels(orgName, repoName, false, false, false)

	repoClaLabelStatus := ghc.GetRepoClaLabelStatus(ghc, orgName, repoName)
	assert.False(t, repoClaLabelStatus.HasYes)
	assert.False(t, repoClaLabelStatus.HasNo)
	assert.False(t, repoClaLabelStatus.HasExternal)
}

func TestGetRepoClaLabelStatus_HasYesOnly(t *testing.T) {
	setUp(t)
	defer tearDown(t)

	expectRepoLabels(orgName, repoName, true, false, false)

	repoClaLabelStatus := ghc.GetRepoClaLabelStatus(ghc, orgName, repoName)
	assert.True(t, repoClaLabelStatus.HasYes)
	assert.False(t, repoClaLabelStatus.HasNo)
	assert.False(t, repoClaLabelStatus.HasExternal)
}

func TestGetRepoClaLabelStatus_HasNoOnly(t *testing.T) {
	setUp(t)
	defer tearDown(t)

	expectRepoLabels(orgName, repoName, false, true, false)

	repoClaLabelStatus := ghc.GetRepoClaLabelStatus(ghc, orgName, repoName)
	assert.False(t, repoClaLabelStatus.HasYes)
	assert.True(t, repoClaLabelStatus.HasNo)
	assert.False(t, repoClaLabelStatus.HasExternal)
}

func TestGetRepoClaLabelStatus_YesAndNoLabels(t *testing.T) {
	setUp(t)
	defer tearDown(t)

	expectRepoLabels(orgName, repoName, true, true, false)

	repoClaLabelStatus := ghc.GetRepoClaLabelStatus(ghc, orgName, repoName)
	assert.True(t, repoClaLabelStatus.HasYes)
	assert.True(t, repoClaLabelStatus.HasNo)
	assert.False(t, repoClaLabelStatus.HasExternal)
}

func TestGetRepoClaLabelStatus_YesNoAndExternalLabels(t *testing.T) {
	setUp(t)
	defer tearDown(t)

	expectRepoLabels(orgName, repoName, true, true, true)

	repoClaLabelStatus := ghc.GetRepoClaLabelStatus(ghc, orgName, repoName)
	assert.True(t, repoClaLabelStatus.HasYes)
	assert.True(t, repoClaLabelStatus.HasNo)
	assert.True(t, repoClaLabelStatus.HasExternal)
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

	commitStatus := ghutil.ProcessCommit(&commit, claSigners)
	assert.True(t, commitStatus.Compliant, "Commit should have been marked compliant; reason: ", commitStatus.NonComplianceReason)
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

func getSinglePullSpec() ghutil.GitHubProcessSinglePullSpec {
	localPullNumber := pullNumber
	localPullTitle := "no title"
	pull := github.PullRequest{
		Number: &localPullNumber,
		Title:  &localPullTitle,
	}

	return ghutil.GitHubProcessSinglePullSpec{
		Org:  orgName,
		Repo: repoName,
		Pull: &pull,
	}
}

func TestCheckPullRequestCompliance_ListCommitsError(t *testing.T) {
	setUp(t)
	defer tearDown(t)

	err := errors.New("Invalid PR")
	mockGhc.PullRequests.EXPECT().ListCommits(any, orgName, repoName, pullNumber, nil).Return(nil, nil, err)

	prSpec := getSinglePullSpec()
	claSigners := config.ClaSigners{}
	pullRequestStatus, retErr := ghc.CheckPullRequestCompliance(ghc, prSpec, claSigners)
	assert.False(t, pullRequestStatus.Compliant)
	assert.Equal(t, "", pullRequestStatus.NonComplianceReason)
	assert.Equal(t, err, retErr)
}

func createCommit(sha string, name string, email string, login string) *github.RepositoryCommit {
	return &github.RepositoryCommit{
		SHA: &sha,
		Commit: &github.Commit{
			Author: &github.CommitAuthor{
				Name:  &name,
				Email: &email,
			},
			Committer: &github.CommitAuthor{
				Name:  &name,
				Email: &email,
			},
		},
		Author: &github.User{
			Login: &login,
		},
		Committer: &github.User{
			Login: &login,
		},
	}
}

func TestCheckPullRequestCompliance_TwoCompliantCommits(t *testing.T) {
	setUp(t)
	defer tearDown(t)

	sha1 := "12345abcde"
	name1 := "John Doe"
	email1 := "john@example.com"
	login1 := "john-doe"

	sha2 := "abcde24680"
	name2 := "Jane Doe"
	email2 := "jane@example.com"
	login2 := "jane-doe"

	commits := []*github.RepositoryCommit{
		createCommit(sha1, name1, email1, login1),
		createCommit(sha2, name2, email2, login2),
	}
	mockGhc.PullRequests.EXPECT().ListCommits(any, orgName, repoName, pullNumber, nil).Return(commits, nil, nil)

	prSpec := getSinglePullSpec()
	claSigners := config.ClaSigners{
		People: []config.Account{
			{
				Name:  name1,
				Email: email1,
				Login: login1,
			},
			{
				Name:  name2,
				Email: email2,
				Login: login2,
			},
		},
	}
	pullRequestStatus, err := ghc.CheckPullRequestCompliance(ghc, prSpec, claSigners)
	assert.True(t, pullRequestStatus.Compliant)
	assert.Equal(t, "", pullRequestStatus.NonComplianceReason)
	assert.Nil(t, err)
}

func TestCheckPullRequestCompliance_OneCompliantOneNot(t *testing.T) {
	setUp(t)
	defer tearDown(t)

	sha1 := "12345abcde"
	name1 := "John Doe"
	email1 := "john@example.com"
	login1 := "john-doe"

	sha2 := "abcde24680"
	name2 := "Jane Doe"
	email2 := "jane@example.com"
	login2 := "jane-doe"

	commits := []*github.RepositoryCommit{
		createCommit(sha1, name1, email1, login1),
		createCommit(sha2, name2, email2, login2),
	}
	mockGhc.PullRequests.EXPECT().ListCommits(any, orgName, repoName, pullNumber, nil).Return(commits, nil, nil)

	prSpec := getSinglePullSpec()
	claSigners := config.ClaSigners{
		People: []config.Account{
			{
				Name:  name1,
				Email: email1,
				Login: login1,
			},
		},
	}
	pullRequestStatus, err := ghc.CheckPullRequestCompliance(ghc, prSpec, claSigners)
	assert.False(t, pullRequestStatus.Compliant)
	assert.Equal(t, "Committer of one or more commits is not listed as a CLA signer, either individual or as a member of an organization.", pullRequestStatus.NonComplianceReason)
	assert.Nil(t, err)
}

type ProcessPullRequest_TestParams struct {
	RepoClaLabelStatus  ghutil.RepoClaLabelStatus
	IssueClaLabelStatus ghutil.IssueClaLabelStatus
	PullRequestStatus   ghutil.PullRequestStatus
	UpdateRepo          bool
	LabelsToAdd         []string
	LabelsToRemove      []string
}

func runProcessPullRequestTestScenario(t *testing.T, params ProcessPullRequest_TestParams) {
	// Dummy CLA signers data as we don't actually need to use it here,
	// since we're mocking out the functions that would otherwise process
	// this data.
	claSigners := config.ClaSigners{}

	prSpec := getSinglePullSpec()
	prSpec.UpdateRepo = params.UpdateRepo

	ghc.CheckPullRequestCompliance = mockGhc.Api.CheckPullRequestCompliance
	mockGhc.Api.EXPECT().CheckPullRequestCompliance(ghc, prSpec, claSigners).Return(params.PullRequestStatus, nil)

	ghc.GetIssueClaLabelStatus = mockGhc.Api.GetIssueClaLabelStatus
	mockGhc.Api.EXPECT().GetIssueClaLabelStatus(ghc, orgName, repoName, pullNumber).Return(params.IssueClaLabelStatus)

	if params.UpdateRepo {
		for _, label := range params.LabelsToAdd {
			mockGhc.Issues.EXPECT().AddLabelsToIssue(any, orgName, repoName, pullNumber, []string{label}).Return(nil, nil, nil)
		}

		for _, label := range params.LabelsToRemove {
			mockGhc.Issues.EXPECT().RemoveLabelForIssue(any, orgName, repoName, pullNumber, label).Return(nil, nil)
		}
	}

	err := ghc.ProcessPullRequest(ghc, prSpec, claSigners, params.RepoClaLabelStatus)
	assert.Nil(t, err)
}

func TestProcessPullRequest_RepoHasLabels_PullHasZeroLabels_Compliant_Update(t *testing.T) {
	setUp(t)
	defer tearDown(t)

	runProcessPullRequestTestScenario(t, ProcessPullRequest_TestParams{
		RepoClaLabelStatus: ghutil.RepoClaLabelStatus{
			HasYes: true,
			HasNo:  true,
		},
		IssueClaLabelStatus: ghutil.IssueClaLabelStatus{},
		PullRequestStatus: ghutil.PullRequestStatus{
			Compliant: true,
		},
		UpdateRepo:  true,
		LabelsToAdd: []string{ghutil.LabelClaYes},
	})
}

func TestProcessPullRequest_RepoHasLabels_PullHasZeroLabels_NonCompliant_Update(t *testing.T) {
	setUp(t)
	defer tearDown(t)

	// When adding a "cla: no" label, we will also add a comment to the
	// effect of why this PR got that label.
	nonComplianceReason := "Your PR is not compliant"
	issueComment := github.IssueComment{
		Body: &nonComplianceReason,
	}
	mockGhc.Issues.EXPECT().CreateComment(any, orgName, repoName, pullNumber, &issueComment).Return(nil, nil, nil)

	runProcessPullRequestTestScenario(t, ProcessPullRequest_TestParams{
		RepoClaLabelStatus: ghutil.RepoClaLabelStatus{
			HasYes: true,
			HasNo:  true,
		},
		IssueClaLabelStatus: ghutil.IssueClaLabelStatus{},
		PullRequestStatus: ghutil.PullRequestStatus{
			Compliant:           false,
			NonComplianceReason: nonComplianceReason,
		},
		UpdateRepo:  true,
		LabelsToAdd: []string{ghutil.LabelClaNo},
	})
}

func TestProcessPullRequest_RepoHasHabels_PullHasYesLabel_Compliant(t *testing.T) {
	setUp(t)
	defer tearDown(t)

	runProcessPullRequestTestScenario(t, ProcessPullRequest_TestParams{
		RepoClaLabelStatus: ghutil.RepoClaLabelStatus{
			HasYes: true,
			HasNo:  true,
		},
		IssueClaLabelStatus: ghutil.IssueClaLabelStatus{
			HasYes: true,
		},
		PullRequestStatus: ghutil.PullRequestStatus{
			Compliant: true,
		},
		UpdateRepo: true,
		// No labels to be added or removed in this case.
	})
}

func TestProcessPullRequest_RepoHasLabels_PullHasYesLabel_NonCompliant(t *testing.T) {
	setUp(t)
	defer tearDown(t)

	// When adding a "cla: no" label, we will also add a comment to the
	// effect of why this PR got that label.
	nonComplianceReason := "Your PR is not compliant"
	issueComment := github.IssueComment{
		Body: &nonComplianceReason,
	}
	mockGhc.Issues.EXPECT().CreateComment(any, orgName, repoName, pullNumber, &issueComment).Return(nil, nil, nil)

	runProcessPullRequestTestScenario(t, ProcessPullRequest_TestParams{
		RepoClaLabelStatus: ghutil.RepoClaLabelStatus{
			HasYes: true,
			HasNo:  true,
		},
		IssueClaLabelStatus: ghutil.IssueClaLabelStatus{
			HasYes: true,
		},
		PullRequestStatus: ghutil.PullRequestStatus{
			Compliant:           false,
			NonComplianceReason: nonComplianceReason,
		},
		UpdateRepo:     true,
		LabelsToAdd:    []string{ghutil.LabelClaNo},
		LabelsToRemove: []string{ghutil.LabelClaYes},
	})
}

func TestProcessPullRequest_RepoHasLabels_HasNoLabel_Compliant(t *testing.T) {
	setUp(t)
	defer tearDown(t)

	runProcessPullRequestTestScenario(t, ProcessPullRequest_TestParams{
		RepoClaLabelStatus: ghutil.RepoClaLabelStatus{
			HasYes: true,
			HasNo:  true,
		},
		IssueClaLabelStatus: ghutil.IssueClaLabelStatus{
			HasNo: true,
		},
		PullRequestStatus: ghutil.PullRequestStatus{
			Compliant: true,
		},
		UpdateRepo:     true,
		LabelsToAdd:    []string{ghutil.LabelClaYes},
		LabelsToRemove: []string{ghutil.LabelClaNo},
	})
}

func TestProcessPullRequest_RepoHasLabels_PullHasNoLabel_NonCompliant(t *testing.T) {
	setUp(t)
	defer tearDown(t)

	runProcessPullRequestTestScenario(t, ProcessPullRequest_TestParams{
		RepoClaLabelStatus: ghutil.RepoClaLabelStatus{
			HasYes: true,
			HasNo:  true,
		},
		IssueClaLabelStatus: ghutil.IssueClaLabelStatus{
			HasNo: true,
		},
		PullRequestStatus: ghutil.PullRequestStatus{
			Compliant: false,
		},
		UpdateRepo: true,
		// No labels to be added or removed in this case.
	})
}
