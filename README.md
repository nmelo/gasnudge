# gn

CLI tool to send nudge messages to Claude agents running in tmux windows.

Nudge protocol and Claude detection logic ripped straight out of [gastown](https://github.com/steveyegge/gastown).

## Installation

```bash
go install github.com/nmelo/gn@latest
```

## Usage

From inside tmux, nudge all other windows in the current session:

```bash
gn "continue"
```

Nudge only windows running Claude:

```bash
gn --detect "continue"
```

Target specific windows:

```bash
gn -w editor -w build "done"
gn -p "worker-*" "update"
```

Preview without sending:

```bash
gn --dry-run "test"
```

## Flags

```
-w, --window NAME      Target specific window(s) by name (repeatable)
-s, --session NAME     Target session (default: current)
-p, --pattern GLOB     Filter windows by name pattern
-d, --detect           Only nudge windows running Claude
-a, --all              Include current window (default: exclude self)
-n, --dry-run          Show what would be nudged
```

## How it works

Uses tmux `send-keys` with a reliable protocol: literal mode for the message, brief delays for paste completion, Escape for vim-mode safety, and Enter with retry logic.

Claude detection checks `pane_current_command` for `node`, `claude`, or version patterns like `2.1.12`, plus child process inspection when the pane shows a shell.

## See also

[gp](https://github.com/nmelo/gaspeek) - the companion tool for reading output from tmux windows
