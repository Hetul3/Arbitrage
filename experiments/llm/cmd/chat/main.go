package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"time"

	openai "github.com/sashabaranov/go-openai"
)

const (
	defaultBaseURL   = "https://api.tokenfactory.nebius.com/v1/"
	defaultModel     = "openai/gpt-oss-120b"
	defaultSystemMsg = "You are a concise, helpful assistant."
)

func main() {
	apiKey := strings.TrimSpace(os.Getenv("NEBIUS_API_KEY"))
	if apiKey == "" {
		log.Fatal("NEBIUS_API_KEY is not set. Copy experiments/.env.template to experiments/.env and add your key.")
	}

	baseURL := strings.TrimSpace(getenv("NEBIUS_BASE_URL", defaultBaseURL))

	cfg := openai.DefaultConfig(apiKey)
	cfg.BaseURL = baseURL

	client := openai.NewClientWithConfig(cfg)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	fmt.Printf("Nebius LLM CLI ready (model: %s, endpoint: %s)\n", defaultModel, baseURL)
	fmt.Println("Type a prompt and press enter. Type 'exit' or 'quit' to leave.")

	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("> ")
		if !scanner.Scan() {
			fmt.Println("\nGoodbye.")
			return
		}
		prompt := strings.TrimSpace(scanner.Text())
		if prompt == "" {
			continue
		}
		if strings.EqualFold(prompt, "exit") || strings.EqualFold(prompt, "quit") {
			fmt.Println("Goodbye.")
			return
		}

		ctxReq, cancel := context.WithTimeout(ctx, 60*time.Second)
		reply, err := sendPrompt(ctxReq, client, prompt)
		cancel()

		if err != nil {
			log.Printf("request failed: %v", err)
			continue
		}

		fmt.Printf("[response]\n%s\n", reply)
	}
}

func sendPrompt(ctx context.Context, client *openai.Client, prompt string) (string, error) {
	req := openai.ChatCompletionRequest{
		Model: defaultModel,
		Messages: []openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleSystem, Content: defaultSystemMsg},
			{Role: openai.ChatMessageRoleUser, Content: prompt},
		},
		MaxTokens:   512,
		Temperature: 0.7,
	}

	resp, err := client.CreateChatCompletion(ctx, req)
	if err != nil {
		return "", err
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no choices returned")
	}

	return strings.TrimSpace(resp.Choices[0].Message.Content), nil
}

func getenv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}
