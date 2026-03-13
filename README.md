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

This project supports running models locally via [Ollama](https://ollama.ai).

### 1. Install Ollama
Follow the instructions on the [Ollama website](https://ollama.ai/download) to install it on your system.  
After installation, you can verify it’s working by running:

```bash
ollama run llama2 "Hello!"
```

### 2. Set environment variables
The app reads configuration from environment variables to decide which LLM provider to use.

For Ollama / LLaMA, set:

```bash
export LLM_PROVIDER=ollama
export LLM_MODEL=llama2
```

- `LLM_PROVIDER` → tells the app which backend to use (`ollama` in this case).  
- `LLM_MODEL` → which model to run (e.g., `llama2`, `llama3`, `mistral`, etc.).  

If you don’t set `LLM_MODEL`, it defaults to `llama2`.

### 3. Run the app
Once the environment variables are set and Ollama is running in the background:
```
go run ./main.go
````

### 4. Install
Once the project successfully launched, you can execute:
```
go install
````
This will install the app. Now you should be able to interact with the project by typing <b>oda</b> in your terminal

## Project Plan

**Phase 1 — Core Q&A**  
The first step is building the core question-answering engine. This involves integrating an AI model through a modular backend service that processes user queries. The system will return plain text answers, allowing testing for accuracy and reliability while keeping the AI model interchangeable.

**Phase 2 — Terminal UI**  
Once the core Q&A engine is functional, a terminal interface will be implemented. This interface will handle input and output streams for a smooth interactive experience. Libraries such as `rich`, `textual`, or `blessed` will be considered. A history log will be added to allow users to review past interactions.

**Phase 3 — Code Editing**  
The final phase focuses on adding AI-driven code editing capabilities. This will involve enabling secure file access and integrating structured editing tools. The AI will be able to both explain suggested changes and apply them safely, with version control to track modifications.

## Model Strategy

Oda is designed to be model-agnostic. The backend will support multiple AI models through a flexible architecture so models can be swapped without major changes. For early testing and development, free and open-source models (such as LLaMA 3, GPT-NeoX, or StarCoder) will be used. This allows rapid iteration without cost while keeping future expansion open to more advanced or specialized models. This flexibility ensures the project is not limited to a single provider or technology.
