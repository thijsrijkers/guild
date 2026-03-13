package main

import (
	"context"
	"log"
	"guild/llm"
	"guild/tui"
)

func main() {
	ctx := context.Background()

	client, err := llm.NewFromEnv()
	if err != nil {
		log.Fatal(err)
	}

	tui.StartChat(ctx, client)
}
