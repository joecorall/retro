package main

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"os"

	"github.com/joecorall/retro/internal/thirdparty"
)

type Message struct {
	GitHubActor string `json:"github_actor"`
	// comma separated list of GitHub organizations
	// not to include in the summary
	GitHubIgnoredOrgs string `json:"github_ignored_orgs"`
	SlackWebhookUrl   string `json:"slack_webhook_url"`
}

func init() {
	if os.Getenv("GITHUB_TOKEN") == "" || os.Getenv("OPENAI_API_KEY") == "" {
		slog.Error("GITHUB_TOKEN and OPENAI_API_KEY environment variables are required")
		os.Exit(1)
	}
}

func main() {
	http.HandleFunc("/", MessageHandler)
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	slog.Info("Server listening", "port", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		panic(err)
	}
}

func MessageHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	decoder := json.NewDecoder(r.Body)
	var m Message
	err := decoder.Decode(&m)
	if err != nil {
		slog.Error("Unable to summarize work", "err", err)
		http.Error(w, "Unprocessable Content", http.StatusInternalServerError)
		return
	}

	if m.GitHubActor == "" {
		http.Error(w, "No GitHub username passed", http.StatusBadRequest)
		return
	}

	work := thirdparty.FindGitHubIssuesAndCommits(m.GitHubActor, m.GitHubIgnoredOrgs)
	if work == "" {
		http.Error(w, "Unprocessable entity", http.StatusUnprocessableEntity)
		return
	}

	summary, err := thirdparty.GptSummarize(m.GitHubActor, work)
	if err != nil {
		slog.Error("Unable to summarize work", "err", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	if m.SlackWebhookUrl == "" && os.Getenv("SLACK_WEBHOOK_URL") == "" {
		if _, err := w.Write([]byte(summary.Choices[0].Message.Content)); err != nil {
			slog.Error("Error writing summary to responsewriter", "err", err)
			http.Error(w, "Internal error", http.StatusInternalServerError)
			return
		}
		return
	}

	err = thirdparty.SendToSlack(m.SlackWebhookUrl, summary.Choices[0].Message.Content)
	if err != nil {
		slog.Error("Unable to send summary to slack", "err", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	if _, err := w.Write([]byte("Successfully sent summary to slack")); err != nil {
		slog.Error("Error writing success to responsewriter", "err", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)
	}
}
