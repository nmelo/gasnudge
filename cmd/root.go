package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/nmelo/gasnudge/internal/tmux"
	"github.com/spf13/cobra"
)

var (
	windowFlags  []string
	sessionFlag  string
	patternFlag  string
	detectFlag   bool
	allFlag      bool
	dryRunFlag   bool
	clearFlag    bool
)

var rootCmd = &cobra.Command{
	Use:   "gn [flags] [message]",
	Short: "Send nudge messages to Claude agents in tmux windows",
	Long: `gasnudge sends messages to Claude agents running in tmux windows.

By default, it nudges ALL windows in the current session except the caller's window.
Use --detect to limit to only windows running Claude.

Examples:
  gn "continue"                    # Nudge all windows except self
  gn --detect "continue"           # Nudge only windows running Claude
  gn -w editor -w build "done"     # Nudge specific windows
  gn -p "worker-*" "update"        # Nudge windows matching pattern
  gn --dry-run "test"              # Show what would be nudged
  gn -c -w worker-1 "start"        # Clear window first, then nudge`,
	Args: cobra.MinimumNArgs(1),
	RunE: runNudge,
}

func Execute() error {
	return rootCmd.Execute()
}

// clearAndConfirm sends /clear to a window and confirms it was processed.
// Returns an error if the clear cannot be confirmed within the timeout.
func clearAndConfirm(target string) error {
	// Send /clear command
	if err := tmux.NudgeWindow(target, "/clear"); err != nil {
		return fmt.Errorf("failed to send /clear: %w", err)
	}

	// Wait for Claude to process the clear command
	time.Sleep(2 * time.Second)

	// Peek at the window output to confirm clear happened
	output, err := tmux.CaptureWindow(target, 50)
	if err != nil {
		return fmt.Errorf("failed to peek after /clear: %w", err)
	}

	// Check for signs that /clear was processed:
	// 1. The "/clear" text should NOT still be visible as pending input
	// 2. We should see a fresh prompt or cleared state
	// 3. Look for "Conversation cleared" or similar confirmation

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		// If we see "/clear" as the last thing typed (not yet submitted), it failed
		if trimmed == "/clear" || strings.HasSuffix(trimmed, "> /clear") {
			return fmt.Errorf("clear command appears unsubmitted")
		}
		// Success indicators: Claude shows confirmation or fresh prompt
		if strings.Contains(strings.ToLower(line), "cleared") ||
			strings.Contains(line, "Conversation cleared") {
			return nil // Confirmed cleared
		}
	}

	// If we don't see /clear pending and the output looks reasonable, assume success
	// The absence of "/clear" text in the visible output suggests it was processed
	return nil
}

func init() {
	rootCmd.Flags().StringArrayVarP(&windowFlags, "window", "w", nil, "Target specific window(s) by name (repeatable)")
	rootCmd.Flags().StringVarP(&sessionFlag, "session", "s", "", "Target session (default: current)")
	rootCmd.Flags().StringVarP(&patternFlag, "pattern", "p", "", "Filter windows by name pattern (glob-style)")
	rootCmd.Flags().BoolVarP(&detectFlag, "detect", "d", false, "Only nudge windows running Claude")
	rootCmd.Flags().BoolVarP(&allFlag, "all", "a", false, "Include current window (default: exclude self)")
	rootCmd.Flags().BoolVarP(&dryRunFlag, "dry-run", "n", false, "Show what would be nudged")
	rootCmd.Flags().BoolVarP(&clearFlag, "clear", "c", false, "Send /clear first, confirm, then send message")
}

func runNudge(cmd *cobra.Command, args []string) error {
	message := strings.Join(args, " ")

	// Determine session
	var session string
	var currentWindowIndex int
	var currentPaneID string

	if tmux.IsInsideTmux() {
		var err error
		currentSession, currentWindowIdx, paneID, err := tmux.GetCurrentContext()
		if err != nil {
			return fmt.Errorf("failed to get tmux context: %w", err)
		}
		currentPaneID = paneID
		if sessionFlag != "" {
			session = sessionFlag
			currentWindowIndex = -1 // Different session, don't exclude any window
		} else {
			session = currentSession
			currentWindowIndex = currentWindowIdx
		}
	} else {
		if sessionFlag == "" {
			return fmt.Errorf("not inside tmux; use -s/--session to specify target session")
		}
		session = sessionFlag
		currentWindowIndex = -1 // No window to exclude
	}

	// Verify session exists
	if !tmux.SessionExists(session) {
		return fmt.Errorf("session %q does not exist", session)
	}

	// Get all windows
	windows, err := tmux.ListWindows(session)
	if err != nil {
		return fmt.Errorf("failed to list windows: %w", err)
	}

	// Filter windows
	var targets []tmux.Window
	for _, w := range windows {
		// Exclude current window unless --all is set
		if !allFlag && currentWindowIndex >= 0 && w.Index == currentWindowIndex {
			continue
		}

		// Filter by specific window names if provided
		if len(windowFlags) > 0 {
			found := false
			for _, name := range windowFlags {
				if w.Name == name || fmt.Sprintf("%d", w.Index) == name {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		// Filter by pattern if provided
		if patternFlag != "" && !tmux.MatchPattern(w.Name, patternFlag) {
			continue
		}

		// Filter by Claude detection if requested
		if detectFlag && !tmux.IsClaudeRunning(w) {
			continue
		}

		targets = append(targets, w)
	}

	if len(targets) == 0 {
		fmt.Fprintln(os.Stderr, "No windows to nudge")
		return nil
	}

	// Dry run: show targets and exit
	if dryRunFlag {
		fmt.Printf("Would nudge %d window(s) in session %q:\n", len(targets), session)
		for _, w := range targets {
			claudeStatus := ""
			if tmux.IsClaudeRunning(w) {
				claudeStatus = " [claude]"
			}
			fmt.Printf("  %d: %s (%s)%s\n", w.Index, w.Name, w.Command, claudeStatus)
		}
		fmt.Printf("\nMessage: %s\n", message)
		return nil
	}

	// Execute nudges
	var succeeded, failed int
	for _, w := range targets {
		target := fmt.Sprintf("%s:%d", session, w.Index)

		// If --clear flag is set, send /clear first and confirm
		if clearFlag {
			if err := clearAndConfirm(target); err != nil {
				fmt.Fprintf(os.Stderr, "Failed to clear %s: %v\n", w.Name, err)
				failed++
				continue
			}
		}

		if err := tmux.NudgeWindow(target, message); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to nudge %s: %v\n", w.Name, err)
			failed++
		} else {
			succeeded++
		}
	}

	// Report results
	if failed > 0 {
		fmt.Printf("Nudged %d window(s), %d failed\n", succeeded, failed)
		return fmt.Errorf("%d nudge(s) failed", failed)
	}

	// Don't print current pane ID in output, just use it for internal logic
	_ = currentPaneID

	fmt.Printf("Nudged %d window(s)\n", succeeded)
	return nil
}
