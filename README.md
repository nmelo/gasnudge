# gn

Send nudge messages to Claude agents in tmux windows with interruption.

Part of the [gas-tools](https://github.com/nmelo) family for multi-agent coordination in tmux.

## Installation

**Homebrew:**
```bash
brew tap nmelo/tap
brew install gn
```

**Go:**
```bash
go install github.com/nmelo/gasnudge@latest
```

## Quick Start

```bash
# Nudge all windows in current session
gn "continue"

# Nudge only windows running Claude
gn --detect "urgent: check logs"

# Preview targets without sending
gn --dry-run "test"
```

## Usage

```
gasnudge (gn) sends messages to Claude agents running in tmux windows with interruption.

WHEN TO USE gn vs ga:
  gn  - Urgent: "stop now" (sends Escape to interrupt current work)
  ga  - Non-urgent: "when you're done, run tests" (queues without interrupting)

BEHAVIOR:
  - Targets ALL windows in the current session (not Claude-only by default)
  - Excludes the caller's own window (prevents self-messaging)
  - Sends Escape key to interrupt vim-mode or pending input
  - Use --detect to limit to only windows running Claude

NUDGE PROTOCOL:
  1. Send message text in literal mode
  2. Wait 500ms for paste completion
  3. Send Escape (interrupts any pending input or vim-mode)
  4. Send Enter with retry logic

CLAUDE DETECTION (--detect flag):
  Identifies Claude by pane_current_command matching:
  - "claude" or "node" (direct process)
  - Version pattern like "2.1.25"
  - Child processes of shells (inspects via pgrep)

USE CASES FOR AGENT COORDINATION:
  - Urgently interrupt an agent stuck in a loop
  - Force immediate attention to a critical issue
  - Clear and restart an agent's context
  - Broadcast urgent messages across a swarm

EXAMPLES:
  gn "stop and check the error"      # Nudge all windows in session
  gn --detect "urgent: check logs"   # Only windows running Claude
  gn -w worker-1 "stop now"          # Target specific window
  gn -w worker-1 -w worker-2 "halt"  # Multiple windows
  gn -p "worker-*" "checkpoint"      # Glob pattern matching
  gn -s swarm "broadcast"            # Different tmux session
  gn -a "note to self"               # Include own window
  gn -n "test"                       # Dry-run: show targets
  gn -c -w worker-1 "fresh start"    # Clear context first, then nudge

RELATED TOOLS:
  ga (gasadd)   - Queue messages without interrupting (no Escape)
  gp (gaspeek)  - Read output from agent windows
  gm (gasmail)  - Persistent messaging via beads database

Usage:
  gn [flags] [message]

Flags:
  -a, --all                  Include current window (default: exclude self)
  -c, --clear                Send /clear first, confirm, then send message
  -d, --detect               Only nudge windows running Claude
  -n, --dry-run              Show what would be nudged
  -h, --help                 help for gn
  -p, --pattern string       Filter windows by name pattern (glob-style)
  -s, --session string       Target session (default: current)
  -w, --window stringArray   Target specific window(s) by name (repeatable)
```

## Related Tools

- [ga](https://github.com/nmelo/gasadd) - Queue messages without interrupting
- [gp](https://github.com/nmelo/gaspeek) - Read output from agent windows
- [gm](https://github.com/nmelo/gasmail) - Persistent messaging via beads

## Credits

Nudge protocol and Claude detection logic adapted from [gastown](https://github.com/steveyegge/gastown).
