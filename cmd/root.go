package cmd

import (
	"fmt"
	"os"
	"strings"

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
  gn --dry-run "test"              # Show what would be nudged`,
	Args: cobra.MinimumNArgs(1),
	RunE: runNudge,
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.Flags().StringArrayVarP(&windowFlags, "window", "w", nil, "Target specific window(s) by name (repeatable)")
	rootCmd.Flags().StringVarP(&sessionFlag, "session", "s", "", "Target session (default: current)")
	rootCmd.Flags().StringVarP(&patternFlag, "pattern", "p", "", "Filter windows by name pattern (glob-style)")
	rootCmd.Flags().BoolVarP(&detectFlag, "detect", "d", false, "Only nudge windows running Claude")
	rootCmd.Flags().BoolVarP(&allFlag, "all", "a", false, "Include current window (default: exclude self)")
	rootCmd.Flags().BoolVarP(&dryRunFlag, "dry-run", "n", false, "Show what would be nudged")
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
