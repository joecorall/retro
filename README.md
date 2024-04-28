# retro

Help summarize what you did this week

## Why?

If you struggle to prepare for team syncs/retros with a list of what you did last week, this service may be able to help.

## What?

Searches GitHub for your commit and PR history over the past week. 

Sends the commit message, PR title, and PR message to ChatGPT asking it to summarize.

Sends the summary to a slack webhook URL

### Environment Variables

| Env Var Name       | Explanation                                                                |
|------------------- |--------------------------------------------------------------------------- |
| `SLACK_WEBHOOK_URL`| Your summary will be sent to this URL as `curl -d {"text": "YOUR_SUMMARY"}` |
| `GITHUB_TOKEN`     | Your GitHub token so you can read commmits and PRs from private repos      |
| `GITHUB_ACTOR`     | Your GitHub username                                                       |
| `OPENAI_API_KEY`   | Your OpenAI API Key that can write to `/v1/chat/completions`               |
| `IGNORE_ORGS`      | (optional) comma separated list of GITHUB_ORGS to not include in summary   |
