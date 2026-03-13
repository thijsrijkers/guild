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

// ── conversation history ───────────────────────────────────────────────────

// turn represents a single exchange in the conversation history.
type turn struct {
	role    string // "user" or "assistant"
	content string
}

// historyToPrompt builds a full prompt string from system prompt + history.
func historyToPrompt(systemPrompt string, history []turn) string {
	var sb strings.Builder
	sb.WriteString(systemPrompt)
	sb.WriteString("\n\n")
	for _, t := range history {
		sb.WriteString(t.role)
		sb.WriteString(": ")
		sb.WriteString(t.content)
		sb.WriteString("\n\n")
	}
	return sb.String()
}

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
	history []turn,
	statusBar *tview.TextView,
	app *tview.Application,
) (string, error) {
	// Build prompt from full history so the model has context of past turns
	conversation := historyToPrompt(systemPrompt, history)

	for range 10 {
		response, err := client.Ask(ctx, conversation)
		if err != nil {
			return "", err
		}

		a := parseAction(response)
		if a == nil {
			return stripActions(response), nil
		}

		text := stripActions(response)

		switch a.Type {
		case "read_file":
			app.QueueUpdateDraw(func() {
				statusBar.SetText(fmt.Sprintf("  [#ffcb6b]reading %s...[-]", a.Path))
			})
			fileContent, err := prompt.ReadFile(a.Path)
			if err != nil {
				conversation += fmt.Sprintf("assistant: %s\n\nsystem: Could not read %s: %v. Try a different path.\n\n", text, a.Path, err)
			} else {
				conversation += fmt.Sprintf(
					"assistant: %s\n\nsystem: Contents of %s:\n```\n%s\n```\nNow apply the change using write_file.\n\n",
					text, a.Path, fileContent,
				)
			}

		case "write_file":
			app.QueueUpdateDraw(func() {
				statusBar.SetText(fmt.Sprintf("  [#ffcb6b]writing %s...[-]", a.Path))
			})
			if err := writeFile(a.Path, a.Content); err != nil {
				conversation += fmt.Sprintf("assistant: %s\n\nsystem: write_file failed: %v\n\n", text, err)
			} else {
				return text + fmt.Sprintf("\n\n✅ Written to %s", a.Path), nil
			}

		case "replace_in_file":
			app.QueueUpdateDraw(func() {
				statusBar.SetText(fmt.Sprintf("  [#ffcb6b]updating %s...[-]", a.Path))
			})
			existing, err := prompt.ReadFile(a.Path)
			if err != nil {
				conversation += fmt.Sprintf("assistant: %s\n\nsystem: Could not read %s: %v\n\n", text, a.Path, err)
			} else if !strings.Contains(existing, a.Old) {
				conversation += fmt.Sprintf(
					"assistant: %s\n\nsystem: replace_in_file failed — exact \"old\" string not found in %s. Use write_file with the full corrected content instead.\n\nCurrent file:\n```\n%s\n```\n\n",
					text, a.Path, existing,
				)
			} else if err := replaceInFile(a.Path, a.Old, a.New); err != nil {
				conversation += fmt.Sprintf("assistant: %s\n\nsystem: replace_in_file failed: %v\n\n", text, err)
			} else {
				return text + fmt.Sprintf("\n\n✅ Updated %s", a.Path), nil
			}

		default:
			return stripActions(response), nil
		}
	}

	return "", fmt.Errorf("could not complete the change after multiple attempts")
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

	// ── state ──
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

	// history holds all user/assistant turns for context
	var history []turn
	var mu sync.Mutex

	// ── layout ──
	inputFlex := tview.NewFlex().
		SetDirection(tview.FlexColumn).
		AddItem(nil, 1, 0, false).
		AddItem(inputField, 0, 1, true).
		AddItem(nil, 2, 0, false)
	inputFlex.SetBackgroundColor(bgInput)

	mainFlex := tview.NewFlex().
		SetDirection(tview.FlexColumn).
		AddItem(sidebar, 0, 0, false).
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
		// Add user turn to history
		history = append(history, turn{role: "user", content: input})
		historySnapshot := make([]turn, len(history))
		copy(historySnapshot, history)
		messages = append(messages, formatMessage("user", input))
		updateChat(chatView, messages)
		statusBar.SetText("  [#9aa0a6]thinking...[-]")
		mu.Unlock()

		go func(snapshot []turn) {
			response, err := agentAsk(ctx, client, systemPrompt, snapshot, statusBar, app)

			app.QueueUpdateDraw(func() {
				mu.Lock()
				defer mu.Unlock()
				if err != nil {
					messages = append(messages, formatMessage("error", err.Error()))
					statusBar.SetText("  [#f07178]" + err.Error() + "[-]")
				} else {
					// Add assistant turn to history so next message has full context
					history = append(history, turn{role: "assistant", content: response})
					messages = append(messages, formatMessage("assistant", response))
					statusBar.SetText(statusDefault)
				}
				updateChat(chatView, messages)
			})
		}(historySnapshot)
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
			history = []turn{} // also clear history so model forgets too
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
