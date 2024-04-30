package thirdparty

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/google/go-github/v61/github"
	"golang.org/x/oauth2"
)

func FindGitHubIssuesAndCommits(author, filterOrg string) string {
	ctx := context.Background()
	token := os.Getenv("GITHUB_TOKEN")
	// auth to GitHub
	tokenSource := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tokenClient := oauth2.NewClient(ctx, tokenSource)
	client := github.NewClient(tokenClient)

	oneWeekAgo := time.Now().AddDate(0, 0, -7)
	query := fmt.Sprintf("type:pr author:%s", author)
	opt := &github.SearchOptions{
		Sort: "updated",
		ListOptions: github.ListOptions{
			PerPage: 10,
		},
	}

	// do not summarize commits on unrelated orgs
	if filterOrg == "" {
		filterOrg = os.Getenv("IGNORE_ORGS")
	}
	ignoredOrgs := strings.Split(filterOrg, ",")
	for i, org := range ignoredOrgs {
		ignoredOrgs[i] = strings.ToLower(strings.TrimSpace(org))
	}

	re := regexp.MustCompile(`https://api\.github\.com/repos/([^/]+)/`)
	var allPRs []*github.Issue
	for {
		results, resp, err := client.Search.Issues(ctx, query, opt)
		if err != nil {
			slog.Error("Error searching pull request", "err", err)
			return ""
		}

		for _, pr := range results.Issues {
			match := re.FindStringSubmatch(*pr.URL)
			if match == nil || len(match) != 2 {
				continue
			}
			if strInSlice(strings.ToLower(match[1]), ignoredOrgs) {
				continue
			}
			// you can not filter by date on PRs
			// so instead sort by updated and bail when we're past the target date
			if pr.UpdatedAt.Before(oneWeekAgo) {
				resp.NextPage = 0
				break
			}
			if pr.GetPullRequestLinks() != nil {
				allPRs = append(allPRs, pr)
			}
		}
		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}

	lines := []string{}
	for _, pr := range allPRs {
		lines = append(lines, fmt.Sprintf("PR Title: %s\n", *pr.Title))
		if pr.Body != nil {
			lines = append(lines, fmt.Sprintf("PR Message: %s\n", *pr.Body))
		}
		lines = append(lines, fmt.Sprintf("PR URL: %s\n", *pr.HTMLURL))
		lines = append(lines, "-----------------------------------")
	}

	query = fmt.Sprintf("author:%s author-date:>%s", author, oneWeekAgo.Format("2006-01-02"))
	var allCommits []*github.Commit
	for {
		results, resp, err := client.Search.Commits(ctx, query, opt)
		if err != nil {
			slog.Error("Error searching pull requests", "err", err)
			return ""
		}
		for _, commit := range results.Commits {
			match := re.FindStringSubmatch(*commit.Commit.URL)

			if match == nil || len(match) != 2 {
				continue
			}
			if strInSlice(strings.ToLower(match[1]), ignoredOrgs) {
				continue
			}

			allCommits = append(allCommits, commit.Commit)
		}
		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}

	for _, commit := range allCommits {
		lines = append(lines, fmt.Sprintf("Commit Message: %s\n", *commit.Message))
		lines = append(lines, fmt.Sprintf("Commit URL: %s\n", *commit.URL))
		lines = append(lines, "-----------------------------------")
	}

	return strings.Join(lines, "\n")
}
