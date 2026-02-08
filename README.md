# SAA (Single Action Agent)

SAA (Single Action Agent) is a simple autonomous agent which has only one tool: BASH.

## Why?

- Because if the model is smart enough, BASH is all you need.
- Because Claude Code makes my terminal lag like it's the 70s.
- Because I find it weird when coding tools are tightly coupled with UI or API frameworks. Keep it simple.

This repository needs no stars, no support, nothing. If you find it interesting, go ahead and build your own "better version".

## Features

- Session Management: Maintains conversation history in `.saa/session` directory.
- Configuration: Project specific and user level configuration support.

## Prerequisites

- Go 1.23 or later
- OpenAI API Key (or compatible API)

## Installation

```bash
go install github.com/moravy-mochi/saa@latest
```

## Example

with llama-server and GLM-4.7-Flash(UDQ4):
````
$ saa config --show-tool-call
$ saa n
$ saa x add main.c with hello world
[MESSAGE]
I'll create a main.c file with a simple Hello World program.
[TOOL] cat > main.c << 'EOF'
#include <stdio.h>

int main() {
    printf("Hello, World!\n");
    return 0;
}
EOF
[TOOL] cat main.c
[MESSAGE]
Done! I've created `main.c` with a simple Hello World program:

```c
#include <stdio.h>

int main() {
    printf("Hello, World!\n");
    return 0;
}
```

The program includes the standard input/output header, defines the main function, prints "Hello, World!" to the console, and returns 0 to indicate successful execution.
$ ls
main.c
````

For more practical examples of how the agent thinks and acts, check out the recorded session logs in [examples/simple/.saa/session/](examples/simple/.saa/session/).

## Usage

### Initialize a project

`saa init` creats `.saa` directory.

### Configure settings

llama-server from [llama.cpp](https://github.com/ggml-org/llama.cpp):

```bash
saa config --api-key dummy-or-your-key --api-url https://localhost:8080/v1 --model your-model-name
```

### Execute a task

```bash
saa exec "List the files in the current directory and explain what they are."
```

or

```bash
saa x list files and explain
```

### New session

`saa new` or `saa n`

## Q&A

### Why no sandbox? Isn't it dangerous?

True Unix users sandbox themselves.
Use bubblewrap, Docker, or whatever you prefer.
It's annoying to have bubblewrap calls built-in if you're already running inside Docker, right?
Use the project root returned by `saa where` as an argument for bubblewrap.

Check out [examples/simple/saa-wrapper](examples/simple/saa-wrapper) for a concrete example using bubblewrap.

### Typing commands every time is a pain.

```bash
alias sx='your-cool-saa-wrapper.sh exec'
alias sn='your-cool-saa-wrapper.sh new'
```

Or build a chat UI wrapper and come show it off.

### I want to be notified when a session is created, etc.

Build a nice UI using `saa session current` and other commands.
How about adding it to your shell prompt?

### AGENTS.md is not working?

`cat $(saa where)/AGENTS.md | saa x your prompt`

### How to use MCP or Skills?

Just write a script to use them and describe the usage in your prompt. It's a win-win because you'll be able to use them yourself too.

### What about plan mode? Sub-agents? Teams?

You can do it all.
Assemble your own ultimate autonomous agent army using prompts and scripts, and show it off.

### What about foo-claw-bot? I want to know what's missing in my fridge via a messaging app.

You can do anything.
Create your own ultimate chatbot using prompts and scripts to streamline your shopping.
Just be careful not to leave your credit card in the fridge.

## License

MIT
