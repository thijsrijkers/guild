package tui

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"oda/llm"
	"oda/prompt"
	"regexp"
	"strings"
	"sync"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// ── colours ────────────────────────────────────────────────────────────────

var (
	bgMain    = tcell.GetColor("#0f1115")
	bgInput   = tcell.GetColor("#14161b")
	bgSidebar = tcell.GetColor("#0d0f13")
	fgText    = tcell.ColorWhite
	fgMuted   = tcell.GetColor("#9aa0a6")
	fgGreen   = tcell.GetColor("#3bb88a")
	_         = fgMuted
)

// ── action parsing ──────────────────────────────────────────────────────────

var actionRegex = regexp.MustCompile(`(?s)<action>(.*?)</action>`)

type action struct {
	Type    string `json:"type"`
	Path    string `json:"path"`
	Content string `json:"content"`
	Old     string `json:"old"`
	New     string `json:"new"`
}

func parseAction(response string) *action {
	matches := actionRegex.FindStringSubmatch(response)
	if matches == nil {
		return nil
	}
	var a action
	if err := json.Unmarshal([]byte(strings.TrimSpace(matches[1])), &a); err != nil {
		return nil
	}
	return &a
}

func stripActions(response string) string {
	return strings.TrimSpace(actionRegex.ReplaceAllString(response, ""))
}

// ── message formatting ──────────────────────────────────────────────────────

func formatMessage(role, text string) string {
	var roleTag string
	switch role {
	case "user":
		roleTag = "[#c792ea]> you[-]"
	case "assistant":
		roleTag = "[#3bb88a]> oda[-]"
	case "error":
		roleTag = "[#f07178]> error[-]"
	default:
		roleTag = fmt.Sprintf("[#9aa0a6]> %s[-]", role)
	}
	header := roleTag + "\n"
	body := "  " + strings.ReplaceAll(text, "\n", "\n  ") + "\n"
	divider := "[#1e2025]────────────────────────────────────────[-]\n"
	return header + body + divider
}

func updateChat(view *tview.TextView, messages []string) {
	view.Clear()
	for _, msg := range messages {
		fmt.Fprint(view, msg)
	}
	view.ScrollToEnd()
}

// ── sidebar ─────────────────────────────────────────────────────────────────

func buildSidebar(entries []prompt.FileEntry, onSelect func(string)) *tview.List {
	list := tview.NewList()
	list.SetBackgroundColor(bgSidebar)
	list.SetMainTextColor(fgText)
	list.SetSelectedBackgroundColor(tcell.GetColor("#1e2530"))
	list.SetSelectedTextColor(fgGreen)
	list.SetTitle(" files ").SetTitleColor(fgGreen)
	list.SetBorder(true).SetBorderColor(tcell.GetColor("#1e2025"))
	list.ShowSecondaryText(false)

	for _, e := range entries {
		path := e.RelPath
		list.AddItem(path, "", 0, func() {
			onSelect(path)
		})
	}

	return list
}

// ── agentic ask loop ─────────────────────────────────────────────────────────

func agentAsk(
	ctx context.Context,
	client llm.LLM,
	systemPrompt string,
	userInput string,
	statusBar *tview.TextView,
	app *tview.Application,
) (string, error) {
	conversation := systemPrompt + "\n\nUser: " + userInput

	for range 5 {
		response, err := client.Ask(ctx, conversation)
		if err != nil {
			return "", err
		}

		a := parseAction(response)
		if a == nil {
			return stripActions(response), nil
		}

		switch a.Type {
		case "read_file":
			app.QueueUpdateDraw(func() {
				statusBar.SetText(fmt.Sprintf("  [#ffcb6b]reading %s...[-]", a.Path))
			})
			content, err := prompt.ReadFile(a.Path)
			if err != nil {
				conversation += fmt.Sprintf("\n\nCould not read %s: %v", a.Path, err)
			} else {
				conversation += fmt.Sprintf(
					"\n\nContents of %s:\n```\n%s\n```\nNow answer the user's question.",
					a.Path, content,
				)
			}

		case "write_file":
			app.QueueUpdateDraw(func() {
				statusBar.SetText(fmt.Sprintf("  [#ffcb6b]writing %s...[-]", a.Path))
			})
			if err := writeFile(a.Path, a.Content); err != nil {
				return stripActions(response), fmt.Errorf("write failed: %w", err)
			}
			return stripActions(response) + fmt.Sprintf("\n\n✅ Written to %s", a.Path), nil

		case "replace_in_file":
			app.QueueUpdateDraw(func() {
				statusBar.SetText(fmt.Sprintf("  [#ffcb6b]updating %s...[-]", a.Path))
			})
			if err := replaceInFile(a.Path, a.Old, a.New); err != nil {
				return stripActions(response), fmt.Errorf("replace failed: %w", err)
			}
			return stripActions(response) + fmt.Sprintf("\n\n✅ Updated %s", a.Path), nil

		default:
			return stripActions(response), nil
		}
	}

	return "", fmt.Errorf("agent exceeded maximum file read iterations")
}

// ── file helpers ─────────────────────────────────────────────────────────────

func writeFile(path, content string) error {
	return writeFileOS(path, []byte(content))
}

func replaceInFile(path, old, new string) error {
	data, err := readFileOS(path)
	if err != nil {
		return err
	}
	updated := strings.ReplaceAll(string(data), old, new)
	return writeFileOS(path, []byte(updated))
}

const statusDefault = "  [#9aa0a6]ctrl+c[-] quit   [#9aa0a6]ctrl+l[-] clear   [#9aa0a6]ctrl+b[-] files   [#9aa0a6]enter[-] send"
const statusSidebar = "  [#9aa0a6]ctrl+b[-] hide files   [#9aa0a6]ctrl+f[-] focus   [#9aa0a6]esc[-] back to input"

// ── main entry ────────────────────────────────────────────────────────────────

func StartChat(parentCtx context.Context, client llm.LLM) {
	ctx, cancel := context.WithCancel(parentCtx)
	defer cancel()

	entries, err := prompt.BuildFileList(".")
	if err != nil {
		log.Fatalf("could not scan project: %v", err)
	}
	systemPrompt := prompt.Build(entries)

	app := tview.NewApplication()

	// ── chat view ──
	chatView := tview.NewTextView()
	chatView.
		SetDynamicColors(true).
		SetScrollable(true).
		SetWrap(true).
		SetWordWrap(true)
	chatView.SetBackgroundColor(bgMain)
	chatView.SetTextColor(fgText)
	chatView.SetChangedFunc(func() { app.Draw() })

	// ── input field ──
	inputField := tview.NewInputField().
		SetFieldWidth(0).
		SetFieldBackgroundColor(bgInput).
		SetFieldTextColor(fgText)
	inputField.SetBackgroundColor(bgInput)

	// ── status bar ──
	statusBar := tview.NewTextView()
	statusBar.SetDynamicColors(true)
	statusBar.SetBackgroundColor(bgInput)
	statusBar.SetText(statusDefault)

	// ── sidebar ──
	sidebar := buildSidebar(entries, func(path string) {
		current := inputField.GetText()
		if current == "" {
			inputField.SetText("explain " + path)
		} else {
			inputField.SetText(current + " " + path)
		}
		app.SetFocus(inputField)
	})

	// ── messages ──
	messages := []string{
		`[#3bb88a]
  ██████╗ ██████╗  █████╗
 ██╔═══██╗██╔══██╗██╔══██╗
 ██║   ██║██║  ██║███████║
 ██║   ██║██║  ██║██╔══██║
 ╚██████╔╝██████╔╝██║  ██║
  ╚═════╝ ╚═════╝ ╚═╝  ╚═╝`,
		fmt.Sprintf("[#9aa0a6] \n Loaded %d project files into context.\n Type a message and press Enter.\n[-]\n", len(entries)),
	}
	updateChat(chatView, messages)

	var mu sync.Mutex

	// ── layout ──
	// Use ResizeItem to show/hide sidebar — avoids Clear() rebuilding issues
	inputFlex := tview.NewFlex().
		SetDirection(tview.FlexColumn).
		AddItem(nil, 1, 0, false).
		AddItem(inputField, 0, 1, true).
		AddItem(nil, 2, 0, false)
	inputFlex.SetBackgroundColor(bgInput)

	mainFlex := tview.NewFlex().
		SetDirection(tview.FlexColumn).
		AddItem(sidebar, 0, 0, false). // width=0 = hidden by default
		AddItem(chatView, 0, 1, false)

	root := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(mainFlex, 0, 1, false).
		AddItem(statusBar, 1, 0, false).
		AddItem(inputFlex, 2, 0, true).
		AddItem(nil, 1, 0, false)

	sidebarVisible := false

	// ── input handler ──
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
		statusBar.SetText("  [#9aa0a6]thinking...[-]")
		mu.Unlock()

		go func(userInput string) {
			response, err := agentAsk(ctx, client, systemPrompt, userInput, statusBar, app)

			app.QueueUpdateDraw(func() {
				mu.Lock()
				defer mu.Unlock()
				if err != nil {
					messages = append(messages, formatMessage("error", err.Error()))
					statusBar.SetText("  [#f07178]" + err.Error() + "[-]")
				} else {
					messages = append(messages, formatMessage("assistant", response))
					statusBar.SetText(statusDefault)
				}
				updateChat(chatView, messages)
			})
		}(input)
	})

	// ── global key bindings ──
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

		case tcell.KeyCtrlB:
			sidebarVisible = !sidebarVisible
			if sidebarVisible {
				mainFlex.ResizeItem(sidebar, 28, 0)
				statusBar.SetText(statusSidebar)
			} else {
				mainFlex.ResizeItem(sidebar, 0, 0)
				statusBar.SetText(statusDefault)
				app.SetFocus(inputField)
			}
			return nil

		case tcell.KeyCtrlF:
			if !sidebarVisible {
				sidebarVisible = true
				mainFlex.ResizeItem(sidebar, 28, 0)
				statusBar.SetText(statusSidebar)
			}
			app.SetFocus(sidebar)
			return nil

		case tcell.KeyEscape:
			app.SetFocus(inputField)
			return nil
		}
		return event
	})

	app.SetBeforeDrawFunc(func(screen tcell.Screen) bool {
		screen.Fill(' ', tcell.StyleDefault.Background(bgMain))
		return false
	})

	if err := app.SetRoot(root, true).SetFocus(inputField).EnableMouse(true).Run(); err != nil {
		log.Fatalf("error starting chat: %v", err)
	}
}
