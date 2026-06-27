package llm

import (
	"bytes"
	"encoding/json"
	"net/http"
)

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatRequest struct {
	Model     string        `json:"model"`
	Messages  []ChatMessage `json:"messages"`
	MaxTokens int           `json:"max_tokens"`
}

type ChatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

type Client struct {
	Url   string
	Model string
}

func New(url, model string) *Client {
	return &Client{Url: url, Model: model}
}

func (c *Client) Chat(messages []ChatMessage, maxTokens int) (string, error) {
	req := ChatRequest{
		Model:     c.Model,
		Messages:  messages,
		MaxTokens: maxTokens,
	}

	body, _ := json.Marshal(req)
	resp, err := http.Post(
		c.Url+"/v1/chat/completions",
		"application/json",
		bytes.NewBuffer(body),
	)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result ChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	if len(result.Choices) == 0 {
		return "", nil
	}

	return result.Choices[0].Message.Content, nil
}
