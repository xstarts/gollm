package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"gollm"
)

func main() {
	fmt.Println("Starting the enhanced prompt types example...")

	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Fatalf("OPENAI_API_KEY environment variable is not set")
	}

	llm, err := gollm.NewLLM(
		gollm.SetProvider("zhipu_view"),
		gollm.SetModel("cogview-3"),
		gollm.SetAPIKey(apiKey),
		gollm.SetMaxTokens(300),
		gollm.SetMaxRetries(3),
		gollm.SetDebugLevel(gollm.LogLevelDebug),
	)
	if err != nil {
		log.Fatalf("Failed to create LLM client: %v", err)
	}

	ctx := context.Background()

	// Example 1: Basic Prompt with Structured Output
	fmt.Println("\nExample 1: Basic Prompt with Structured Output")
	basicPrompt := gollm.NewPrompt("一个城市在水晶瓶中欢快生活的场景，水彩画风格，展现出微观与珠宝般的美丽。")
	basicResponse, err := llm.Generate(ctx, basicPrompt)
	if err != nil {
		log.Printf("Failed to generate basic response: %v", err)
	} else {
		fmt.Printf("Basic Prompt Response:\n%s\n", basicResponse)
	}

}
