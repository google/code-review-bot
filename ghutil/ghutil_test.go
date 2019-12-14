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

	name := "John Doe"
	corporateEmail := "john@github.com"
	personalEmail := "john@gmail.com"
	login := "johndoe"

	personal := config.Account{
		Name:  name,
		Email: personalEmail,
		Login: login,
	}
	corporate := config.Account{
		Name:  name,
		Email: corporateEmail,
		Login: login,
	}

	claSigners := config.ClaSigners{
		Companies: []config.Company{
			{
				Name:   "Acme Inc.",
				People: []config.Account{corporate, personal},
			},
		},
	}
	commit := createCommit(corporate, personal)
	commitStatus := ghutil.ProcessCommit(commit, claSigners)
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

func createCommit(author config.Account, committer config.Account) *github.RepositoryCommit {
	// Uniqueness of SHA fingerprints for commits is not an invariant
	// that's required or enforced anywhere; we just need a non-null value
	// here, so it's OK to use the same value for all commits to avoid
	// dummy data in our test code.
	sha := "abc123def456"

	return &github.RepositoryCommit{
		SHA: &sha,
		Commit: &github.Commit{
			Author: &github.CommitAuthor{
				Name:  &author.Name,
				Email: &author.Email,
			},
			Committer: &github.CommitAuthor{
				Name:  &committer.Name,
				Email: &committer.Email,
			},
		},
		Author: &github.User{
			Login: &author.Login,
		},
		Committer: &github.User{
			Login: &committer.Login,
		},
	}
}

func TestCheckPullRequestCompliance_TwoCompliantCommits(t *testing.T) {
	setUp(t)
	defer tearDown(t)

	john, jane := createUserAccounts()

	commits := []*github.RepositoryCommit{
		createCommit(john, john),
		createCommit(jane, jane),
	}
	mockGhc.PullRequests.EXPECT().ListCommits(any, orgName, repoName, pullNumber, nil).Return(commits, nil, nil)

	prSpec := getSinglePullSpec()
	claSigners := config.ClaSigners{
		People: []config.Account{john, jane},
	}
	pullRequestStatus, err := ghc.CheckPullRequestCompliance(ghc, prSpec, claSigners)
	assert.True(t, pullRequestStatus.Compliant)
	assert.Equal(t, "", pullRequestStatus.NonComplianceReason)
	assert.Nil(t, err)
}

func TestCheckPullRequestCompliance_OneCompliantOneNot(t *testing.T) {
	setUp(t)
	defer tearDown(t)

	john, jane := createUserAccounts()

	commits := []*github.RepositoryCommit{
		createCommit(john, john),
		createCommit(jane, jane),
	}
	mockGhc.PullRequests.EXPECT().ListCommits(any, orgName, repoName, pullNumber, nil).Return(commits, nil, nil)

	prSpec := getSinglePullSpec()
	claSigners := config.ClaSigners{
		People: []config.Account{john},
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

func TestProcessPullRequest_RepoHasLabels_PullHasZeroLabels_External_Update(t *testing.T) {
	setUp(t)
	defer tearDown(t)

	runProcessPullRequestTestScenario(t, ProcessPullRequest_TestParams{
		RepoClaLabelStatus: ghutil.RepoClaLabelStatus{
			HasYes:      true,
			HasNo:       true,
			HasExternal: true,
		},
		IssueClaLabelStatus: ghutil.IssueClaLabelStatus{},
		PullRequestStatus: ghutil.PullRequestStatus{
			External: true,
		},
		UpdateRepo:  true,
		LabelsToAdd: []string{ghutil.LabelClaExternal},
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

func TestProcessPullRequest_RepoHasYesNoExternalHabels_PullHasYesLabel_External(t *testing.T) {
	setUp(t)
	defer tearDown(t)

	runProcessPullRequestTestScenario(t, ProcessPullRequest_TestParams{
		RepoClaLabelStatus: ghutil.RepoClaLabelStatus{
			HasYes:      true,
			HasNo:       true,
			HasExternal: true,
		},
		IssueClaLabelStatus: ghutil.IssueClaLabelStatus{
			HasYes: true,
		},
		PullRequestStatus: ghutil.PullRequestStatus{
			External: true,
		},
		UpdateRepo:     true,
		LabelsToAdd:    []string{ghutil.LabelClaExternal},
		LabelsToRemove: []string{ghutil.LabelClaYes},
	})
}

func TestProcessPullRequest_RepoHasYesNoHabels_PullHasYesLabel_External(t *testing.T) {
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
			External: true,
		},
		UpdateRepo: true,
		// The external label wouldn't be added in this case, since the
		// repo doesn't have it.
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

func TestProcessPullRequest_RepoHasLabels_PullHasNoLabel_External(t *testing.T) {
	setUp(t)
	defer tearDown(t)

	runProcessPullRequestTestScenario(t, ProcessPullRequest_TestParams{
		RepoClaLabelStatus: ghutil.RepoClaLabelStatus{
			HasYes:      true,
			HasNo:       true,
			HasExternal: true,
		},
		IssueClaLabelStatus: ghutil.IssueClaLabelStatus{
			HasNo: true,
		},
		PullRequestStatus: ghutil.PullRequestStatus{
			External: true,
		},
		UpdateRepo:     true,
		LabelsToAdd:    []string{ghutil.LabelClaExternal},
		LabelsToRemove: []string{ghutil.LabelClaNo},
	})
}

func TestProcessOrgRepo_SpecifiedPrs(t *testing.T) {
	setUp(t)
	defer tearDown(t)

	localRepoName := repoName
	repos := []*github.Repository{
		{
			Name: &localRepoName,
		},
	}

	ghc.GetAllRepos = mockGhc.Api.GetAllRepos
	mockGhc.Api.EXPECT().GetAllRepos(ghc, orgName, repoName).Return(repos)

	pullNumber1 := 42
	pullTitle1 := "pull 42 title"
	pullRequest1 := github.PullRequest{
		Number: &pullNumber1,
		Title:  &pullTitle1,
	}
	pullNumber2 := 43
	pullTitle2 := "pull 43 title"
	pullRequest2 := github.PullRequest{
		Number: &pullNumber2,
		Title:  &pullTitle2,
	}
	mockGhc.PullRequests.EXPECT().Get(any, orgName, repoName, pullNumber1).Return(&pullRequest1, nil, nil)
	mockGhc.PullRequests.EXPECT().Get(any, orgName, repoName, pullNumber2).Return(&pullRequest2, nil, nil)

	repoClaLabelStatus := ghutil.RepoClaLabelStatus{}

	ghc.GetRepoClaLabelStatus = mockGhc.Api.GetRepoClaLabelStatus
	mockGhc.Api.EXPECT().GetRepoClaLabelStatus(ghc, orgName, repoName).Return(repoClaLabelStatus)

	claSigners := config.ClaSigners{}

	prSpec1 := ghutil.GitHubProcessSinglePullSpec{
		Org:  orgName,
		Repo: repoName,
		Pull: &pullRequest1,
	}
	prSpec2 := ghutil.GitHubProcessSinglePullSpec{
		Org:  orgName,
		Repo: repoName,
		Pull: &pullRequest2,
	}
	ghc.ProcessPullRequest = mockGhc.Api.ProcessPullRequest
	mockGhc.Api.EXPECT().ProcessPullRequest(ghc, prSpec1, claSigners, repoClaLabelStatus)
	mockGhc.Api.EXPECT().ProcessPullRequest(ghc, prSpec2, claSigners, repoClaLabelStatus)

	repoSpec := ghutil.GitHubProcessOrgRepoSpec{
		Org:   orgName,
		Repo:  repoName,
		Pulls: []int{pullNumber1, pullNumber2},
	}
	ghc.ProcessOrgRepo(ghc, repoSpec, claSigners)
}

func TestProcessOrgRepo_AllPrs(t *testing.T) {
	setUp(t)
	defer tearDown(t)

	localRepoName := repoName
	repos := []*github.Repository{
		{
			Name: &localRepoName,
		},
	}

	ghc.GetAllRepos = mockGhc.Api.GetAllRepos
	mockGhc.Api.EXPECT().GetAllRepos(ghc, orgName, repoName).Return(repos)

	pullNumber1 := 42
	pullTitle1 := "pull 42 title"
	pullNumber2 := 43
	pullTitle2 := "pull 43 title"
	pullRequests := []*github.PullRequest{
		{
			Number: &pullNumber1,
			Title:  &pullTitle1,
		},
		{
			Number: &pullNumber2,
			Title:  &pullTitle2,
		},
	}
	mockGhc.PullRequests.EXPECT().List(any, orgName, repoName, nil).Return(pullRequests, nil, nil)

	repoClaLabelStatus := ghutil.RepoClaLabelStatus{}

	ghc.GetRepoClaLabelStatus = mockGhc.Api.GetRepoClaLabelStatus
	mockGhc.Api.EXPECT().GetRepoClaLabelStatus(ghc, orgName, repoName).Return(repoClaLabelStatus)

	claSigners := config.ClaSigners{}

	ghc.ProcessPullRequest = mockGhc.Api.ProcessPullRequest
	for _, pull := range pullRequests {
		prSpec := ghutil.GitHubProcessSinglePullSpec{
			Org:  orgName,
			Repo: repoName,
			Pull: pull,
		}
		mockGhc.Api.EXPECT().ProcessPullRequest(ghc, prSpec, claSigners, repoClaLabelStatus)
	}

	repoSpec := ghutil.GitHubProcessOrgRepoSpec{
		Org:  orgName,
		Repo: repoName,
	}
	ghc.ProcessOrgRepo(ghc, repoSpec, claSigners)
}

func createUserAccounts() (config.Account, config.Account) {
	john := config.Account{
		Name:  "John Doe",
		Email: "john@example.com",
		Login: "john-doe",
	}
	jane := config.Account{
		Name:  "Jane Doe",
		Email: "jane@example.com",
		Login: "jane-doe",
	}
	return john, jane
}

func TestIsExternal_JustJohnInPeople(t *testing.T) {
	setUp(t)
	defer tearDown(t)

	john, jane := createUserAccounts()

	claSigners := config.ClaSigners{
		People: []config.Account{
			john,
		},
	}

	commits := []*github.RepositoryCommit{
		createCommit(john, jane),
	}

	for _, commit := range commits {
		assert.False(t, ghutil.IsExternal(commit, claSigners, false),
			"commit should not be considered external: %v", *commit)
	}
}

func TestIsExternal_JohnAndJaneInPeople(t *testing.T) {
	setUp(t)
	defer tearDown(t)

	john, jane := createUserAccounts()

	claSigners := config.ClaSigners{
		People: []config.Account{
			john,
			jane,
		},
	}

	commits := []*github.RepositoryCommit{
		createCommit(john, john),
		createCommit(john, jane),
		createCommit(jane, john),
		createCommit(jane, jane),
	}

	for _, commit := range commits {
		assert.False(t, ghutil.IsExternal(commit, claSigners, false),
			"commit should not be considered external: %v", *commit)
	}
}

func TestIsExternal_JaneIsABot(t *testing.T) {
	setUp(t)
	defer tearDown(t)

	john, jane := createUserAccounts()

	claSigners := config.ClaSigners{
		People: []config.Account{
			john,
		},
		Bots: []config.Account{
			jane,
		},
	}

	commits := []*github.RepositoryCommit{
		createCommit(john, john),
		createCommit(john, jane),
		createCommit(jane, john),
		createCommit(jane, jane),
	}

	for _, commit := range commits {
		assert.False(t, ghutil.IsExternal(commit, claSigners, false),
			"commit should not be considered external: %v", *commit)
	}
}

func TestIsExternal_JaneIsExternalPerson(t *testing.T) {
	setUp(t)
	defer tearDown(t)

	john, jane := createUserAccounts()

	claSigners := config.ClaSigners{
		People: []config.Account{
			john,
		},
		External: &config.ExternalClaSigners{
			People: []config.Account{
				jane,
			},
		},
	}

	commits := []*github.RepositoryCommit{
		createCommit(john, jane),
		createCommit(jane, jane),
		createCommit(jane, john),
	}

	for _, commit := range commits {
		assert.True(t, ghutil.IsExternal(commit, claSigners, false),
			"commit should be considered external: %v", *commit)
	}
}

func TestIsExternal_JaneIsExternalBot(t *testing.T) {
	setUp(t)
	defer tearDown(t)

	john, jane := createUserAccounts()

	claSigners := config.ClaSigners{
		People: []config.Account{
			john,
		},
		External: &config.ExternalClaSigners{
			Bots: []config.Account{
				jane,
			},
		},
	}

	commits := []*github.RepositoryCommit{
		createCommit(john, jane),
		createCommit(jane, jane),
		createCommit(jane, john),
	}

	for _, commit := range commits {
		assert.True(t, ghutil.IsExternal(commit, claSigners, false),
			"commit should be considered external: %v", *commit)
	}
}

func TestIsExternal_JaneIsExternalCorporate(t *testing.T) {
	setUp(t)
	defer tearDown(t)

	john, jane := createUserAccounts()

	claSigners := config.ClaSigners{
		People: []config.Account{
			john,
		},
		External: &config.ExternalClaSigners{
			Companies: []config.Company{
				{
					Name: "company",
					People: []config.Account{
						jane,
					},
				},
			},
		},
	}

	commits := []*github.RepositoryCommit{
		createCommit(john, jane),
		createCommit(jane, jane),
		createCommit(jane, john),
	}

	for _, commit := range commits {
		assert.True(t, ghutil.IsExternal(commit, claSigners, false),
			"commit should be considered external: %v", *commit)
	}
}

func TestIsExternal_JaneIsCorporate_UnknownAsExternal(t *testing.T) {
	setUp(t)
	defer tearDown(t)

	john, jane := createUserAccounts()

	claSigners := config.ClaSigners{
		People: []config.Account{
			john,
		},
		Companies: []config.Company{
			{
				Name: "company",
				People: []config.Account{
					jane,
				},
			},
		},
	}

	commits := []*github.RepositoryCommit{
		createCommit(john, jane),
		createCommit(jane, jane),
		createCommit(jane, john),
	}

	for _, commit := range commits {
		assert.False(t, ghutil.IsExternal(commit, claSigners, true),
			"commit should not be considered external: %v", *commit)
	}
}

func TestIsExternal_JaneIsUnlisted_UnknownAsExternal(t *testing.T) {
	setUp(t)
	defer tearDown(t)

	john, jane := createUserAccounts()

	claSigners := config.ClaSigners{
		People: []config.Account{
			john,
		},
	}

	commits := []*github.RepositoryCommit{
		createCommit(jane, jane),
	}

	for _, commit := range commits {
		assert.True(t, ghutil.IsExternal(commit, claSigners, true),
			"commit should be considered external: %v", *commit)
	}
}
