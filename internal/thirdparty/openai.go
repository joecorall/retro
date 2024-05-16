package thirdparty

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
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

func GptSummarize(author, work string) (GptResponse, error) {

	var gpt GptResponse
	url := "https://api.openai.com/v1/chat/completions"

	systemPrompt := []string{
		"You are an assistant who explains programming work concisely",
		"Your summaries will be used to share what work has been done in a given time span with other members of an engineering team",
		"Individuals asking for summaries are not asking so they can understand the work you're summarizing",
		"Instead, your summary's audience consists of team members and outside stakeholders that had no involvement with the work",
		"Your summaries help other team members and stakeholders get up to speed of what work has been done",
		"You avoid technical jargon when you can, understanding sometimes there are no other words that can accurately describe something",
		"Your summaries will be sent to a Slack webook that can not render markdown so make sure URLS always end in a space so slack renders them correctly",
		"Make sure every URL you provide starts and ends with a space or new line character since slack won't render the link correctly otherwise",
		"Whenever you are asked to summarize work, after you've provided your summary, your last sentence should be: \"In total there were X commits and Y PRs across Z repositories worked on this week.\" where you replace X, Y, Z with the numbers you summarized",
		"When someone gives you their github username, make sure you use it in the first sentence in the summary so everyone reading knows who did what",
		"A great example summary you should aim to produce based on the input you receive is like:",
		`This week, joecorall worked on several projects primarily related to the Islandora Repository. Here is a brief rundown:

• PR https://github.com/Islandora-Devops/isle-dc/pull/390 was about adding a test for config export functionality through UI in Drupal admin.
• The program warnings when indexing in Solr were fixed in https://github.com/Islandora/controlled_access_terms/pull/114
• Functional JavaScript tests were enabled with PR https://github.com/Islandora/islandora/pull/1013
• Additional tools for website development like wget and git were declared as requirements in PR https://github.com/Islandora-Devops/isle-site-template/pull/35
• A novel attempt at adding Windows testing support was done with PR https://github.com/Islandora-Devops/isle-dc/pull/385
• Also, there were changes made to https://github.com/lehigh-university-libraries/rollout to allow passing payloads to rollout route and minor bump in release.

In addition, various commits were made addressing an array of features and corrections like helper function to decide whether to hide hOCR on Mirador, adding favicons, Docker prune to nightly backup, prod deployment to sequence diagram and several others.

In total there were 20 commits and 10 PRs across 5 repositories worked on this week.`,
	}
	payload := map[string]interface{}{
		"model": "gpt-4o",
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
