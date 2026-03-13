package tui

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"

	"oda/llm"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

var (
	bgMain  = tcell.GetColor("#0f1115")
	bgInput = tcell.GetColor("#14161b")

	fgText   = tcell.ColorWhite
	fgMuted  = tcell.GetColor("#9aa0a6")
	fgAccent = tcell.GetColor("#c792ea")
)

func formatMessage(role, text string) string {
	header := fmt.Sprintf("[#9aa0a6]> %s[-]\n", role)
	body := "  " + text + "\n"
	divider := "[#333333]────────────────────────────────────────[-]\n"
	return header + body + divider
}

func updateChat(view *tview.TextView, messages []string) {
	view.Clear()
	for _, msg := range messages {
		fmt.Fprint(view, msg)
	}
}

func StartChat(parentCtx context.Context, client llm.LLM) {
	ctx, cancel := context.WithCancel(parentCtx)
	defer cancel()

	app := tview.NewApplication()

	chatView := tview.NewTextView()
	chatView.
		SetDynamicColors(true).
		SetScrollable(true).
		SetWrap(true).
		SetWordWrap(true)
	chatView.SetBackgroundColor(bgMain)
	chatView.SetTextColor(fgText)

	chatView.SetChangedFunc(func() {
		app.Draw()
	})

	inputField := tview.NewInputField().
		SetFieldWidth(0).
		SetFieldBackgroundColor(bgInput).
		SetFieldTextColor(fgText)

	inputField.SetBackgroundColor(bgInput)

	messages := []string{
	`[#3bb88a]
  ██████╗ ██████╗  █████╗
 ██╔═══██╗██╔══██╗██╔══██╗
 ██║   ██║██║  ██║███████║
 ██║   ██║██║  ██║██╔══██║
 ╚██████╔╝██████╔╝██║  ██║
  ╚═════╝ ╚═════╝ ╚═╝  ╚═╝`,
	"[#9aa0a6] \n Type a message and press Enter.\n Ctrl+C to quit • Ctrl+L to clear.\n[-]\n",
	}

	updateChat(chatView, messages)

	var mu sync.Mutex

	inputField.SetDoneFunc(func(key tcell.Key) {
		if key != tcell.KeyEnter {
			return
		}

		input := strings.TrimSpace(inputField.GetText())
		if input == "" {
			return
		}

		inputField.SetText("")

		mu.Lock()
		messages = append(messages, formatMessage("user", input))
		updateChat(chatView, messages)
		mu.Unlock()

		go func(userInput string) {
			response, err := client.Ask(ctx, userInput)

			app.QueueUpdateDraw(func() {
				mu.Lock()
				defer mu.Unlock()

				if err != nil {
					messages = append(messages, formatMessage("error", err.Error()))
				} else {
					messages = append(messages, formatMessage("assistant", response))
				}
				updateChat(chatView, messages)
			})
		}(input)
	})

	inputFlex := tview.NewFlex().
		SetDirection(tview.FlexColumn).
		AddItem(nil, 1, 0, false).
		AddItem(inputField, 0, 1, true).
		AddItem(nil, 2, 0, false)

	inputFlex.SetBackgroundColor(bgInput)

	root := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(chatView, 0, 1, false).
		AddItem(inputFlex, 2, 0, true).
		AddItem(nil, 1, 0, false)

	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyCtrlC:
			cancel()
			app.Stop()
			return nil
		case tcell.KeyCtrlL:
			mu.Lock()
			messages = []string{}
			updateChat(chatView, messages)
			mu.Unlock()
			return nil
		}
		return event
	})

	app.SetBeforeDrawFunc(func(screen tcell.Screen) bool {
		screen.Fill(' ', tcell.StyleDefault.Background(bgMain))
		return false
	})

	if err := app.SetRoot(root, true).EnableMouse(true).Run(); err != nil {
		log.Fatalf("error starting chat: %v", err)
	}
}
