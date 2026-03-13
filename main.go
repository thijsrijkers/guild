package main

import (
	"context"
	"log"
	"oda/llm"
	"oda/tui"
)

func main() {
	ctx := context.Background()

	client, err := llm.NewFromEnv()
	if err != nil {
		log.Fatal(err)
	}

	tui.StartChat(ctx, client)
}
