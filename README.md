```
  ██████╗ ██████╗  █████╗
 ██╔═══██╗██╔══██╗██╔══██╗
 ██║   ██║██║  ██║███████║
 ██║   ██║██║  ██║██╔══██║
 ╚██████╔╝██████╔╝██║  ██║
  ╚═════╝ ╚═════╝ ╚═╝  ╚═╝
```
Oda is an AI-powered terminal assistant designed to answer questions, interact through a terminal UI, and make code adjustments.

## Setup & installation

### 1. Set environment variables

The app reads configuration from environment variables to decide which LLM provider to use.

**Ollama (local, free):**
```bash
export LLM_PROVIDER=ollama
export LLM_MODEL=mistral
```

**Claude Haiku (recommended — cheap cloud model):**
```bash
export LLM_PROVIDER=claude
export LLM_MODEL=claude-haiku-4-5-20251001
export ANTHROPIC_API_KEY=your_key_here
```
Get your API key at [console.anthropic.com](https://console.anthropic.com).

**Claude Sonnet (more capable, higher cost):**
```bash
export LLM_PROVIDER=claude
export LLM_MODEL=claude-sonnet-4-6
export ANTHROPIC_API_KEY=your_key_here
```

**Gemini (free tier, EU region may have restrictions):**
```bash
export LLM_PROVIDER=gemini
export LLM_MODEL=gemini-2.0-flash
export GEMINI_API_KEY=your_key_here
```
Get your API key at [aistudio.google.com](https://aistudio.google.com).

**OpenAI:**
```bash
export LLM_PROVIDER=openai
export LLM_MODEL=gpt-4o
export OPENAI_API_KEY=your_key_here
```

### 2. Run the app

```bash
go run ./main.go
```

### 3. Install

Once the project successfully launched, you can execute:

```bash
go install
```

This will install the app. Now you should be able to interact with the project by typing **oda** in your terminal.

## Terminal UI

The TUI has the following keybindings:

| Key | Action |
|---|---|
| `Enter` | Send message |
| `Ctrl+B` | Toggle file sidebar |
| `Ctrl+F` | Focus file sidebar |
| `Ctrl+L` | Clear chat and reset conversation history |
| `Escape` | Return focus to input |
| `Ctrl+C` | Quit |

Clicking a file in the sidebar injects it into the input field so you can ask questions about it directly.

## File Editing

Oda can read and modify files in your project. Simply ask it naturally:

- _"Add error handling to llm/ollama.go"_
- _"Refactor the Ask function in llm/gemini.go"_
- _"Create a new file called tools/parser.go with..."_

Oda will automatically read the relevant file, apply the change, and confirm what it did. The conversation history is maintained across messages so follow-up instructions like _"do it again"_ or _"also add a comment"_ work as expected.

Use `Ctrl+L` to clear the conversation history when starting a new unrelated task — this also reduces token usage and keeps costs low.

## Model Strategy

Oda is model-agnostic. The backend supports multiple AI providers through a flexible architecture — switching models requires only a change in environment variables. 

For local/free usage, Ollama with `mistral` is recommended. For cloud usage, **Claude Haiku** offers the best balance of quality and cost. Larger models like Claude Sonnet or GPT-4o are available when higher reasoning quality is needed.
