# retro

Help summarize what you did this week

## Why?

If you struggle to prepare for team reviews/syncs with a list of what you did last week, this service may be able to help.

## What?

Searches GitHub for your commit and PR history over the past week. 

Sends the commit message, PR title, and PR message to ChatGPT asking it to summarize.

If a `SLACK_WEBHOOK_URL` environment variable is set, sends the summary to a slack webhook URL. Otherwise prints the summary to stdout.

### Environment Variables

| Env Var Name       | Explanation                                                                                                                                            |
|------------------- |------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `GITHUB_TOKEN`     | Your GitHub token so you can read commmits and PRs from private repos                                                                                  |
| `GITHUB_ACTOR`     | Your GitHub username                                                                                                                                   |
| `OPENAI_API_KEY`   | Your OpenAI API Key that can write to `/v1/chat/completions`                                                                                           |
| `SLACK_WEBHOOK_URL`| (optional) Your summary will be sent to this URL using [a mrkdwn block section](https://api.slack.com/messaging/webhooks#advanced_message_formatting)  |
| `IGNORE_ORGS`      | (optional) comma separated list of GITHUB_ORGS to not include in summary                                                                               |
