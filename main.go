package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/google/go-github/v61/github"
	"golang.org/x/oauth2"
)

type GptResponse struct {
	Choices []Choice `json:"choices"`
}
type Choice struct {
	Message Message `json:"message"`
}
type Message struct {
	Content string `json:"content"`
}

var author string

func init() {
	author = os.Getenv("GITHUB_ACTOR")

	if os.Getenv("GITHUB_TOKEN") == "" || author == "" || os.Getenv("OPENAI_API_KEY") == "" || os.Getenv("SLACK_WEBHOOK_URL") == "" {
		slog.Error("GITHUB_TOKEN, GITHUB_ACTOR, OPENAI_API_KEY, and SLACK_WEBHOOK_URL environment variables are required")
		os.Exit(1)
	}
}

func main() {
	work := findGitHubIssuesAndCommits()
	summary, err := gptSummarize(work)
	if err != nil {
		slog.Error("Unable to summarize work", "err", err)
		os.Exit(1)
	}

	// todo: other options to get summary?
	err = sendToSlack(summary.Choices[0].Message.Content)
	if err != nil {
		slog.Error("Unable to send summary to slack", "err", err)
		os.Exit(1)
	}

	slog.Info("Successfully sent summary to slack")
}

func strInSlice(e string, s []string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

func findGitHubIssuesAndCommits() string {
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
	filterOrg := os.Getenv("IGNORE_ORGS")
	ignoredOrgs := strings.Split(filterOrg, ",")

	re := regexp.MustCompile(`https://api\.github\.com/repos/([^/]+)/`)
	var allPRs []*github.Issue
	for {
		results, resp, err := client.Search.Issues(ctx, query, opt)
		if err != nil {
			slog.Error("Error searching pull request", "err", err)
			os.Exit(1)
		}

		for _, pr := range results.Issues {
			match := re.FindStringSubmatch(*pr.URL)
			if match == nil || len(match) != 2 {
				continue
			}
			if strInSlice(match[1], ignoredOrgs) {
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
			os.Exit(1)
		}
		for _, commit := range results.Commits {
			match := re.FindStringSubmatch(*commit.Commit.URL)

			if match == nil || len(match) != 2 {
				continue
			}
			if strInSlice(match[1], ignoredOrgs) {
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

func gptSummarize(work string) (GptResponse, error) {

	var gpt GptResponse
	url := "https://api.openai.com/v1/chat/completions"

	systemPrompt := []string{
		"You are an assistant who explains programming work concisely",
		"Your summaries will be used to share what work has been done in a given time span with other members of an engineering team",
		"Individuals asking for summaries are not asking so they can understand what work has been done",
		"Instead, your summary's audience are primarly read by team members and outside stakeholders that had no involvement with the work",
		"Your summaries help other team members and stakeholders be made aware of what work has been done in a general sense",
		"You avoid technical jargon when you can, understanding sometimes there are no other words to describe",
		"When you reference a PR just provide the URL to it with no markdown formatting",
		"Your summaries should be able to be pasted in Slack so only use markdown slack can render",
		"Whenever you are asked to summarize work, after you've provided your summary, your last sentence should be: \"In addition, there were X commits and Y PRs across Z repositories worked on this week.\" where you replace X, Y, Z with the numbers you summarized",
		"A great example summary you should aim to produce based on the input you receive is like:",
		"When someone gives you their github username, make sure you use it in the first sentence in the summary so everyone reading knows who did what",
		`This week, joecorall worked on several projects primarily related to the Islandora Repository. Here is a brief rundown:

* PR https://github.com/Islandora-Devops/isle-dc/pull/390 was about adding a test for config export functionality through UI in Drupal admin.
* The program warnings when indexing in Solr were fixed in https://github.com/Islandora/controlled_access_terms/pull/114.
* Functional JavaScript tests were enabled with PR [Islandora/islandora/pull/1013](https://github.com/Islandora/islandora/pull/1013.
* Additional tools for website development like wget and git were declared as requirements in PR https://github.com/Islandora-Devops/isle-site-template/pull/35.
* A novel attempt at adding Windows testing support was done with PR https://github.com/Islandora-Devops/isle-dc/pull/385.
* Also, there were changes made to https://github.com/lehigh-university-libraries/rollout to allow passing payloads to rollout route and minor bump in release.

In addition, various commits were made addressing an array of features and corrections like helper function to decide whether to hide hOCR on Mirador, adding favicons, Docker prune to nightly backup, prod deployment to sequence diagram and several others.

In total there were 20 commits and 10 PRs across 5 repositories worked on this week.`,
	}
	payload := map[string]interface{}{
		"model": "gpt-4",
		"messages": []map[string]string{
			{
				"role":    "system",
				"content": strings.Join(systemPrompt, ". "),
			},
			{
				"role":    "user",
				"content": fmt.Sprintf("My github username is %s. Summarize my PR and commit history this week.: %s", author, work),
			},
		},
	}
	jsonData, err := json.Marshal(payload)
	if err != nil {
		fmt.Println("Error marshalling JSON:", err)
		return gpt, err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Println("Error creating request:", err)
		return gpt, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+os.Getenv("OPENAI_API_KEY"))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error sending request:", err)
		return gpt, err
	}
	defer resp.Body.Close()

	if err := json.NewDecoder(resp.Body).Decode(&gpt); err != nil {
		return gpt, err
	}

	return gpt, nil
}

func sendToSlack(summary string) error {
	webhookURL := os.Getenv("SLACK_WEBHOOK_URL")
	if webhookURL == "" {
		return fmt.Errorf("SLACK_WEBHOOK_URL is not set")

	}

	jsonData := []byte(fmt.Sprintf(`{"msg": "%s"}`, summary))
	req, err := http.NewRequest("POST", webhookURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("error creating request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Send the request using the default client
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error sending request:", err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad response code from slack: %d", resp.StatusCode)
	}

	return nil
}
