package main

import (
	"log/slog"
	"os"

	"github.com/joecorall/retro/internal/thirdparty"
)

func init() {
	if os.Getenv("GITHUB_TOKEN") == "" || os.Getenv("GITHUB_ACTOR") == "" || os.Getenv("OPENAI_API_KEY") == "" || os.Getenv("SLACK_WEBHOOK_URL") == "" {
		slog.Error("GITHUB_TOKEN, GITHUB_ACTOR, OPENAI_API_KEY, and SLACK_WEBHOOK_URL environment variables are required")
		os.Exit(1)
	}
}

func main() {
	author := os.Getenv("GITHUB_ACTOR")
	work := thirdparty.FindGitHubIssuesAndCommits(author)
	summary, err := thirdparty.GptSummarize(author, work)
	if err != nil {
		slog.Error("Unable to summarize work", "err", err)
		os.Exit(1)
	}

	// todo: other options to get summary?
	err = thirdparty.SendToSlack(summary.Choices[0].Message.Content)
	if err != nil {
		slog.Error("Unable to send summary to slack", "err", err)
		os.Exit(1)
	}

	slog.Info("Successfully sent summary to slack")
}
