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

// TODO(mbrukman): in the future, consider using the recently-added
// `-copyright_filename` flag: https://github.com/golang/mock/pull/234
//
//go:generate mockgen -source ghutil.go -destination mock_ghutil.go -package ghutil -self_package github.com/google/code-review-bot/ghutil

// Package ghutil provides utility methods for determining CLA compliance of
// pull requests on GitHub repositories, and adding/removing labels and
// comments.
package ghutil

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/go-github/v21/github"

	"github.com/google/code-review-bot/config"
	"github.com/google/code-review-bot/logging"
)

// The CLA-related labels we expect to be predefined on a given repository.
const (
	LabelClaYes      = "cla: yes"
	LabelClaNo       = "cla: no"
	LabelClaExternal = "cla: external"
)

// OrganizationsService is the subset of `github.OrganizationsService` used by
// this module.
type OrganizationsService interface {
}

// RepositoriesService is the subset of `github.RepositoriesService` used by
// this module.
type RepositoriesService interface {
	Get(ctx context.Context, owner string, repo string) (*github.Repository, *github.Response, error)
	List(ctx context.Context, user string, opt *github.RepositoryListOptions) ([]*github.Repository, *github.Response, error)
}

// IssuesService is the subset of `github.IssuesService` used by this module.
type IssuesService interface {
	AddLabelsToIssue(ctx context.Context, owner string, repo string, number int, labels []string) ([]*github.Label, *github.Response, error)
	CreateComment(ctx context.Context, owner string, repo string, number int, comment *github.IssueComment) (*github.IssueComment, *github.Response, error)
	GetLabel(ctx context.Context, owner string, repo string, name string) (*github.Label, *github.Response, error)
	ListLabelsByIssue(ctx context.Context, owner string, repo string, number int, opt *github.ListOptions) ([]*github.Label, *github.Response, error)
	RemoveLabelForIssue(ctx context.Context, owner string, repo string, number int, label string) (*github.Response, error)
}

// PullRequestsService is the subset of `github.PullRequestsService` used by
// this module.
type PullRequestsService interface {
	List(ctx context.Context, owner string, repo string, opt *github.PullRequestListOptions) ([]*github.PullRequest, *github.Response, error)
	ListCommits(ctx context.Context, owner string, repo string, number int, opt *github.ListOptions) ([]*github.RepositoryCommit, *github.Response, error)
	Get(ctx context.Context, owner string, repo string, number int) (*github.PullRequest, *github.Response, error)
}

// GitHubUtilApi is the locally-defined API for interfacing with GitHub, using
// the methods in GitHubClient.
type GitHubUtilApi interface {
	GetAllRepos(*GitHubClient, string, string) []*github.Repository
	CheckPullRequestCompliance(*GitHubClient, GitHubProcessSinglePullSpec, config.ClaSigners) (PullRequestStatus, error)
	ProcessPullRequest(*GitHubClient, GitHubProcessSinglePullSpec, config.ClaSigners, RepoClaLabelStatus) error
	ProcessOrgRepo(*GitHubClient, GitHubProcessOrgRepoSpec, config.ClaSigners)
	GetIssueClaLabelStatus(*GitHubClient, string, string, int) IssueClaLabelStatus
	GetRepoClaLabelStatus(*GitHubClient, string, string) RepoClaLabelStatus
}

// GitHubClient provides an interface to the GitHub APIs used in this module.
type GitHubClient struct {
	// Note: we can't simply use `*GitHubUtilApi` to import all the
	// interface methods here, as they will not be assignable fields and
	// compile will error out with:
	//
	//     cannot use promoted field GitHubUtilApi.GetAllRepos in struct literal of type GitHubClient
	//
	// for each of the methods listed here.
	GetAllRepos                func(*GitHubClient, string, string) []*github.Repository
	CheckPullRequestCompliance func(*GitHubClient, GitHubProcessSinglePullSpec, config.ClaSigners) (PullRequestStatus, error)
	ProcessPullRequest         func(*GitHubClient, GitHubProcessSinglePullSpec, config.ClaSigners, RepoClaLabelStatus) error
	ProcessOrgRepo             func(*GitHubClient, GitHubProcessOrgRepoSpec, config.ClaSigners)
	GetIssueClaLabelStatus     func(*GitHubClient, string, string, int) IssueClaLabelStatus
	GetRepoClaLabelStatus      func(*GitHubClient, string, string) RepoClaLabelStatus

	Organizations OrganizationsService
	Repositories  RepositoriesService
	Issues        IssuesService
	PullRequests  PullRequestsService
}

// GitHubProcessOrgRepoSpec is the specification of the work to be done for an
// organization and repo (possibly multiple PRs).
type GitHubProcessOrgRepoSpec struct {
	Org               string
	Repo              string
	Pulls             []int
	UpdateRepo        bool
	UnknownAsExternal bool
}

// GitHubProcessSinglePullSpec is the specification of work to be processed for
// a single PR, carrying over the rest of the configuration settings from
// GitHubProcessOrgRepoSpec.
type GitHubProcessSinglePullSpec struct {
	Org               string
	Repo              string
	Pull              *github.PullRequest
	UpdateRepo        bool
	UnknownAsExternal bool
}

// NewClient creates a client to work with the GitHub API.
func NewClient(tc *http.Client) *GitHubClient {
	client := github.NewClient(tc)
	client.UserAgent = "cla-helper"

	ghc := NewBasicClient()
	ghc.Organizations = client.Organizations
	ghc.PullRequests = client.PullRequests
	ghc.Issues = client.Issues
	ghc.Repositories = client.Repositories

	return ghc
}

// NewBasicClient returns a new client with only local methods bound; no
// external methods (which require an `http.Client`) are available so this
// client is only partially-constructed and can be used either in production
// with additional bindings added in `NewClient` or for testing by assigning
// mocked methods for the other services.
func NewBasicClient() *GitHubClient {
	ghc := GitHubClient{
		GetAllRepos:                getAllRepos,
		CheckPullRequestCompliance: checkPullRequestCompliance,
		ProcessPullRequest:         processPullRequest,
		ProcessOrgRepo:             processOrgRepo,
		GetIssueClaLabelStatus:     getIssueClaLabelStatus,
		GetRepoClaLabelStatus:      getRepoClaLabelStatus,
	}

	return &ghc
}

// AuthorLogin retrieves the author from a `RepositoryCommit`.
//
// Per go-github project docs in `github/repos_commits.go`:
//
// > RepositoryCommit represents a commit in a repo.
// > Note that it's wrapping a Commit, so author/committer information is
// > in two places, but contain different details about them: in
// > RepositoryCommit "github details", in Commit - "git details".
func AuthorLogin(c *github.RepositoryCommit) string {
	if c.Author != nil {
		if c.Author.Login != nil {
			return *c.Author.Login
		}
	}
	return ""
}

// CommitterLogin retrieves the committer from a `RepositoryCommit`.
//
// See also the docs for `AuthorLogin` for additional information.
func CommitterLogin(c *github.RepositoryCommit) string {
	if c.Committer != nil {
		if c.Committer.Login != nil {
			return *c.Committer.Login
		}
	}
	return ""
}

// getAllRepos retrieves either a single repository (if `repoName` is non-empty)
// or all repositories in an organization of `repoName` is empty.
func getAllRepos(ghc *GitHubClient, orgName string, repoName string) []*github.Repository {
	ctx := context.Background()
	if repoName == "" {
		repos, _, err := ghc.Repositories.List(ctx, orgName, nil)
		if err != nil {
			logging.Fatalf("Error listing all repos in org %s: %s", orgName, err)
		}
		return repos
	}
	repo, _, err := ghc.Repositories.Get(ctx, orgName, repoName)
	if err != nil {
		logging.Fatalf("Error looking up %s/%s: %s", orgName, repoName, err)
	}
	return []*github.Repository{repo}
}

// RepoClaLabelStatus provides the availability of CLA-related labels in the repo.
type RepoClaLabelStatus struct {
	HasYes      bool
	HasNo       bool
	HasExternal bool
}

// getRepoClaLabelStatus checks whether the given GitHub repo has the
// CLA-related labels defined.
func getRepoClaLabelStatus(ghc *GitHubClient, orgName string, repoName string) (repoClaLabelStatus RepoClaLabelStatus) {
	ctx := context.Background()
	repoHasLabel := func(labelName string) bool {
		label, _, err := ghc.Issues.GetLabel(ctx, orgName, repoName, labelName)
		return label != nil && err == nil
	}

	repoClaLabelStatus.HasYes = repoHasLabel(LabelClaYes)
	repoClaLabelStatus.HasNo = repoHasLabel(LabelClaNo)
	repoClaLabelStatus.HasExternal = repoHasLabel(LabelClaExternal)
	return
}

// IssueClaLabelStatus provides the settings of CLA-related labels for a
// particular issue.
type IssueClaLabelStatus struct {
	HasYes      bool
	HasNo       bool
	HasExternal bool
}

// getIssueClaLabelStatus computes the settings of CLA-related Labels for a
// specific issue.
func getIssueClaLabelStatus(ghc *GitHubClient, orgName string, repoName string, pullNumber int) (issueClaLabelStatus IssueClaLabelStatus) {
	ctx := context.Background()
	labels, _, err := ghc.Issues.ListLabelsByIssue(ctx, orgName, repoName, pullNumber, nil)
	if err != nil {
		logging.Errorf("Error listing labels for repo '%s/%s, PR %d: %v", orgName, repoName, pullNumber, err)
		return
	}
	for _, label := range labels {
		if strings.EqualFold(*label.Name, LabelClaYes) {
			issueClaLabelStatus.HasYes = true
		} else if strings.EqualFold(*label.Name, LabelClaNo) {
			issueClaLabelStatus.HasNo = true
		} else if strings.EqualFold(*label.Name, LabelClaExternal) {
			issueClaLabelStatus.HasExternal = true
		}
	}
	return
}

// CanonicalizeEmail returns a canonical version of the email address. For all
// addresses, it will lowercase the email. For Gmail addresses, it will also
// remove the periods in the email address, as those are ignored, and hence
// "user.name@gmail.com" is equivalent to "username@gmail.com" .
func CanonicalizeEmail(email string) string {
	email = strings.ToLower(email)
	gmailSuffixes := [...]string{"@gmail.com", "@googlemail.com"}
	for _, suffix := range gmailSuffixes {
		if strings.HasSuffix(email, suffix) {
			username := strings.TrimSuffix(email, suffix)
			username = strings.Replace(username, ".", "", -1)
			email = fmt.Sprintf("%s%s", username, suffix)
		}
	}
	return email
}

// MatchAccount returns whether the provided account matches any of the accounts
// in the passed-in configuration for enforcing the CLA.
func MatchAccount(account config.Account, accounts []config.Account) bool {
	for _, account2 := range accounts {
		if account.Name == account2.Name &&
			CanonicalizeEmail(account.Email) == CanonicalizeEmail(account2.Email) &&
			strings.EqualFold(account.Login, account2.Login) {
			return true
		}
	}
	return false
}

// CommitStatus provides a signal as to the CLA-compliance of a specific
// commit.
type CommitStatus struct {
	Compliant           bool
	NonComplianceReason string
	External            bool
}

// ProcessCommit processes a single commit and returns compliance status and
// failure reason, if any.
func ProcessCommit(commit *github.RepositoryCommit, claSigners config.ClaSigners) CommitStatus {
	logging.Infof("  - commit: %s", *commit.SHA)

	commitStatus := CommitStatus{
		Compliant: true,
		External:  false,
	}

	authorLogin := AuthorLogin(commit)
	committerLogin := CommitterLogin(commit)
	var authorName, authorEmail string
	var committerName, committerEmail string

	// Only Git information can be found here (name and email only).
	if commit.Commit != nil {
		if commit.Commit.Author != nil {
			commitAuthor := commit.Commit.Author
			if commitAuthor.Name != nil {
				authorName = *commitAuthor.Name
			}
			if commitAuthor.Email != nil {
				authorEmail = *commitAuthor.Email
			}
		}

		if commit.Commit.Committer != nil {
			commitCommitter := commit.Commit.Committer
			if commitCommitter.Name != nil {
				committerName = *commitCommitter.Name
			}
			if commitCommitter.Email != nil {
				committerEmail = *commitCommitter.Email
			}
		}
	}

	if authorName == "" || authorEmail == "" || authorLogin == "" {
		commitStatus.Compliant = false
		commitStatus.NonComplianceReason = "Please verify the author name, email, and GitHub username association are all correct and match CLA records."
	}

	if committerName == "" || committerEmail == "" || committerLogin == "" {
		commitStatus.Compliant = false
		commitStatus.NonComplianceReason = "Please verify the committer name, email, and GitHub username association are all correct and match CLA records."
	}

	// Assuming the commit is compliant thus far, verify that both the author
	// and committer (which could be the same person) have signed the CLA.
	if commitStatus.Compliant {
		authorClaMatchFound := false
		committerClaMatchFound := false

		matchAccount := func(account config.Account, accounts []config.Account) bool {
			for _, account2 := range accounts {
				if account.Name == account2.Name && account.Email == account2.Email &&
					account.Login == account2.Login {
					return true
				}
			}
			return false
		}

		author := config.Account{
			Name:  authorName,
			Email: authorEmail,
			Login: authorLogin,
		}

		committer := config.Account{
			Name:  committerName,
			Email: committerEmail,
			Login: committerLogin,
		}

		authorClaMatchFound = authorClaMatchFound || matchAccount(author, claSigners.People)
		committerClaMatchFound = committerClaMatchFound || matchAccount(committer, claSigners.People)
		committerClaMatchFound = committerClaMatchFound || matchAccount(committer, claSigners.Bots)

		for _, company := range claSigners.Companies {
			authorClaMatchFound = authorClaMatchFound || matchAccount(author, company.People)
			committerClaMatchFound = committerClaMatchFound || matchAccount(committer, company.People)
		}

		if !authorClaMatchFound {
			commitStatus.NonComplianceReason = "Author of one or more commits is not listed as a CLA signer, either individual or as a member of an organization."
		}

		if !committerClaMatchFound {
			commitStatus.NonComplianceReason = "Committer of one or more commits is not listed as a CLA signer, either individual or as a member of an organization."
		}

		commitStatus.Compliant = commitStatus.Compliant && authorClaMatchFound && committerClaMatchFound
	}

	// Put it all together now for display.
	logging.Infof("    author: %s <%s>, GitHub: %s", authorName, authorEmail, authorLogin)
	logging.Infof("    committer: %s <%s>, GitHub: %s", committerName, committerEmail, committerLogin)
	return commitStatus
}

// PullRequestStatus provides the CLA status for the entire PR, which considers
// all of the commits. In this case, any single commit being out of compliance
// (or external) marks the entire PR as being out of compliance (or external).
// The only way to have a fully-compliant PR is to have all commits on the PR
// compliant.
type PullRequestStatus struct {
	Compliant           bool
	NonComplianceReason string
	External            bool
}

// checkPullRequestCompliance reports the compliance status of a pull request,
// considering each of the commits included in the pull request.
func checkPullRequestCompliance(ghc *GitHubClient, prSpec GitHubProcessSinglePullSpec, claSigners config.ClaSigners) (PullRequestStatus, error) {
	ctx := context.Background()
	pullRequestStatus := PullRequestStatus{
		Compliant: false,
		External:  false,
	}

	pullNumber := *prSpec.Pull.Number

	// List all commits for this PR
	commits, _, err := ghc.PullRequests.ListCommits(ctx, prSpec.Org, prSpec.Repo, pullNumber, nil)
	if err != nil {
		logging.Error("Error finding all commits on PR", pullNumber)
		return pullRequestStatus, err
	}

	// Start off with the base case that the PR is compliant and disqualify it if
	// anything is amiss.
	pullRequestStatus.Compliant = true

	for _, commit := range commits {
		// Don't bother processing if either the author's or committer's CLA is managed
		// externally, as it will be picked up by another tool or bot.
		isExternal := IsExternal(commit, claSigners, prSpec.UnknownAsExternal)
		if isExternal {
			pullRequestStatus.External = true
			break
		}

		commitStatus := ProcessCommit(commit, claSigners)

		if commitStatus.Compliant {
			logging.Info("    compliant: true")
		} else {
			logging.Info("    compliant: false:", commitStatus.NonComplianceReason)
			pullRequestStatus.NonComplianceReason = commitStatus.NonComplianceReason
			pullRequestStatus.Compliant = false
		}
	}
	return pullRequestStatus, nil
}

// processPullRequest validates all the commits for a particular pull request,
// and optionally adds/removes labels and comments on a pull request (if the PR
// is non-compliant) to alert the code author and reviewers that they need to
// hold off on reviewing thes changes until the relevant CLA has been signed.
func processPullRequest(ghc *GitHubClient, prSpec GitHubProcessSinglePullSpec, claSigners config.ClaSigners, repoClaLabelStatus RepoClaLabelStatus) error {
	ctx := context.Background()

	orgName := prSpec.Org
	repoName := prSpec.Repo
	pull := prSpec.Pull
	updateRepo := prSpec.UpdateRepo

	logging.Infof("PR %d: %s", *pull.Number, *pull.Title)

	pullRequestStatus, err := ghc.CheckPullRequestCompliance(ghc, prSpec, claSigners)
	if err != nil {
		return err
	}

	issueClaLabelStatus := ghc.GetIssueClaLabelStatus(ghc, orgName, repoName, *pull.Number)
	logging.Infof("  CLA label status [%s]: %v, [%s]: %v, [%s]: %v",
		LabelClaYes, issueClaLabelStatus.HasYes, LabelClaNo, issueClaLabelStatus.HasNo,
		LabelClaExternal, issueClaLabelStatus.HasExternal)

	addLabel := func(label string) {
		logging.Infof("  Adding label [%s] to repo '%s/%s' PR %d...", label, orgName, repoName, *pull.Number)
		if updateRepo {
			_, _, err := ghc.Issues.AddLabelsToIssue(ctx, orgName, repoName, *pull.Number, []string{label})
			if err != nil {
				logging.Errorf("Error adding label [%s] to repo '%s/%s' PR %d: %v", label, orgName, repoName, *pull.Number, err)
			}
		} else {
			logging.Info("  ... but -update-repo flag is disabled; skipping")
		}
	}

	removeLabel := func(label string) {
		logging.Infof("  Removing label [%s] from repo '%s/%s' PR %d...", label, orgName, repoName, *pull.Number)
		if updateRepo {
			_, err := ghc.Issues.RemoveLabelForIssue(ctx, orgName, repoName, *pull.Number, label)
			if err != nil {
				logging.Errorf("  Error removing label [%s] from repo '%s/%s' PR %d: %v", label, orgName, repoName, *pull.Number, err)
			}
		} else {
			logging.Info("  ... but -update-repo flag is disabled; skipping")
		}
	}

	addComment := func(comment string) {
		logging.Infof("  Adding comment to repo '%s/%s/ PR %d: %s", orgName, repoName, *pull.Number, comment)
		if updateRepo {
			issueComment := github.IssueComment{
				Body: &comment,
			}
			_, _, err := ghc.Issues.CreateComment(ctx, orgName, repoName, *pull.Number, &issueComment)
			if err != nil {
				logging.Errorf("  Error leaving comment on PR %d: %v", *pull.Number, err)
			}
		} else {
			logging.Info("  ... but -update-repo flag is disabled; skipping")
		}
	}

	if pullRequestStatus.External {
		logging.Info("  PR has externally-managed CLA signer")

		if issueClaLabelStatus.HasExternal {
			logging.Infof("  PR already has [%s] label", LabelClaExternal)
		} else {
			logging.Infof("  PR doesn't have [%s] label, but should", LabelClaExternal)
			if repoClaLabelStatus.HasExternal {
				addLabel(LabelClaExternal)
			}
		}
		if issueClaLabelStatus.HasYes {
			removeLabel(LabelClaYes)
		}
		if issueClaLabelStatus.HasNo {
			removeLabel(LabelClaNo)
		}

		// No need to add any other CLA-related labels or comments to this PR.
		return nil
	} else {
		if issueClaLabelStatus.HasExternal {
			logging.Infof("  PR has [%s] label, but shouldn't", LabelClaExternal)
			removeLabel(LabelClaExternal)
		} else {
			logging.Infof("  PR doesn't have [%s] label, and shouldn't", LabelClaExternal)
			// Nothing to do here.
		}
	}

	if pullRequestStatus.Compliant {
		logging.Info("  PR is CLA-compliant")
	} else {
		logging.Info("  PR is NOT CLA-compliant:", pullRequestStatus.NonComplianceReason)
	}

	// Add or remove [cla: yes] and [cla: no] labels, as appropriate.
	if pullRequestStatus.Compliant {
		// if PR has [cla: no] label, remove it.
		if issueClaLabelStatus.HasNo {
			removeLabel(LabelClaNo)
		} else {
			logging.Infof("  No action needed: [%s] label already missing", LabelClaNo)
		}
		// if PR doesn't have [cla: yes] label, add it.
		if !issueClaLabelStatus.HasYes {
			if repoClaLabelStatus.HasYes {
				addLabel(LabelClaYes)
			}
		} else {
			logging.Infof("  No action needed: [%s] label already added", LabelClaYes)
		}
	} else /* !pullRequestIsCompliant */ {
		shouldAddComment := false
		// if PR doesn't have [cla: no] label, add it.
		if !issueClaLabelStatus.HasNo {
			if repoClaLabelStatus.HasNo {
				addLabel(LabelClaNo)
			}
			shouldAddComment = true
		} else {
			logging.Infof("  No action needed: [%s] label already added", LabelClaNo)
		}
		// if PR has [cla: yes] label, remove it.
		if issueClaLabelStatus.HasYes {
			removeLabel(LabelClaYes)
			shouldAddComment = true
		} else {
			logging.Infof("  No action needed: [%s] label already missing", LabelClaYes)
		}

		if shouldAddComment {
			addComment(pullRequestStatus.NonComplianceReason)
		}
	}

	return nil
}

// IsExternal computes whether the given commit should be processed by this
// tool, or if it should be covered by an external CLA management tool.
func IsExternal(commit *github.RepositoryCommit, claSigners config.ClaSigners, unknownAsExternal bool) bool {
	var logins []string
	if authorLogin := AuthorLogin(commit); authorLogin != "" {
		logins = append(logins, authorLogin)
	}
	if committerLogin := CommitterLogin(commit); committerLogin != "" {
		logins = append(logins, committerLogin)
	}

	matchLogins := func(logins []string, accounts []config.Account) bool {
		for _, account := range accounts {
			for _, username := range logins {
				if username == account.Login {
					return true
				}
			}
		}
		return false
	}

	if claSigners.External != nil {
		external := claSigners.External
		if matchLogins(logins, external.People) ||
			matchLogins(logins, external.Bots) {
			return true
		}

		for _, company := range external.Companies {
			if matchLogins(logins, company.People) {
				return true
			}
		}
	}

	// If the logins don't match any of the CLA Signers *and* the
	// `unknownAsExternal` is true, then this is an externally-managed
	// contributor.
	if !matchLogins(logins, claSigners.People) && !matchLogins(logins, claSigners.Bots) {
		claEntryFound := false
		for _, company := range claSigners.Companies {
			if matchLogins(logins, company.People) {
				claEntryFound = true
				break
			}
		}
		if !claEntryFound && unknownAsExternal {
			return true
		}
	}

	return false
}

// processOrgRepo handles all PRs in specified repos in the organization or user
// account. If `repoName` is empty, it processes all repos, if `repoName` is
// non-empty, it processes the specified repo.
func processOrgRepo(ghc *GitHubClient, repoSpec GitHubProcessOrgRepoSpec, claSigners config.ClaSigners) {
	ctx := context.Background()
	// Retrieve all repositories for the given organization or user.
	orgName := repoSpec.Org
	repos := ghc.GetAllRepos(ghc, orgName, repoSpec.Repo)

	// For repository, find all outstanding (non-closed / non-merged PRs)
	for _, repo := range repos {
		repoName := *repo.Name

		logging.Infof("Repo: %s/%s", orgName, repoName)

		var pulls []*github.PullRequest
		if len(repoSpec.Pulls) > 0 {
			for _, pullNumber := range repoSpec.Pulls {
				pullRequest, _, err := ghc.PullRequests.Get(ctx, orgName, repoName, pullNumber)
				if err == nil {
					pulls = append(pulls, pullRequest)
				}
			}
		} else {
			// Find all pull requests for the given repo, if not specified.
			retrievedPulls, _, err := ghc.PullRequests.List(ctx, orgName, repoName, nil)
			if err != nil {
				logging.Fatalf("Error listing pull requests for %s/%s: %s", orgName, repoName, err)
			}
			pulls = retrievedPulls
		}

		// Process each pull request for author & commiter CLA status.
		repoClaLabelStatus := ghc.GetRepoClaLabelStatus(ghc, orgName, repoName)
		for _, pull := range pulls {
			prSpec := GitHubProcessSinglePullSpec{
				Org:               orgName,
				Repo:              repoName,
				Pull:              pull,
				UpdateRepo:        repoSpec.UpdateRepo,
				UnknownAsExternal: repoSpec.UnknownAsExternal,
			}
			err := ghc.ProcessPullRequest(ghc, prSpec, claSigners, repoClaLabelStatus)
			if err != nil {
				logging.Errorf("Error processing PR %d: %s", *pull.Number, err)
			}
		}
	}
}
