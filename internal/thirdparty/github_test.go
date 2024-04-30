package thirdparty

import (
	"os"
	"testing"
)

func TestFindGitHubIssuesAndCommits_NoCommits(t *testing.T) {

	work := FindGitHubIssuesAndCommits("joecorall+joecorall", "")
	if work != "" {
		t.Errorf("Found PRs or commits for a bad GitHub username: %s", work)
	}
}

func TestFindGitHubIssuesAndCommits_HasCommits(t *testing.T) {
	// the author will be set to the person pushing the commit to GitHub
	// so they should have at least one commit :P
	author := os.Getenv("GITHUB_ACTOR")
	work := FindGitHubIssuesAndCommits(author, "")
	if work == "" {
		t.Errorf("Found not PRs or commits for a the GitHub username: %s", author)
	}
}
