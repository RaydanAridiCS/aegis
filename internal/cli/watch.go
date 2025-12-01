package cli

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/cobra"
)

// fileSnapshot stores the content and metadata of a file
type fileSnapshot struct {
	content []byte
	hash    [32]byte
	lines   []string
	modTime time.Time
}

// fileTracker keeps track of file states for change detection
type fileTracker struct {
	snapshots map[string]*fileSnapshot
	mu        sync.RWMutex
}

func newFileTracker() *fileTracker {
	return &fileTracker{
		snapshots: make(map[string]*fileSnapshot),
	}
}

type changeSummary struct {
	newSize    int
	lineSpec   string
	hasChanges bool
}

var watchCmd = &cobra.Command{
	Use:   "watch [directory]",
	Short: "Watch a directory for changes",
	Long:  `Watch a directory for file changes and log all changes to terminal and two log files (detailed and basic).`,
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		dir := args[0]

		// Verify directory exists
		if info, err := os.Stat(dir); err != nil || !info.IsDir() {
			fmt.Fprintf(os.Stderr, "Error: '%s' is not a valid directory.\n", dir)
			return
		}

		// Create log directory structure
		timestamp := time.Now().Format("2006-01-02_15-04-05")
		logsDir := "logs"
		timestampDir := filepath.Join(logsDir, timestamp)

		// Create logs folder if it doesn't exist
		if err := os.MkdirAll(timestampDir, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to create logs directory: %v\n", err)
			return
		}

		// Create log file paths
		detailedLogName := filepath.Join(timestampDir, fmt.Sprintf("watch_detailed_%s.log", timestamp))
		basicLogName := filepath.Join(timestampDir, fmt.Sprintf("watch_basic_%s.log", timestamp))

		detailedLog, err := os.OpenFile(detailedLogName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to create detailed log file: %v\n", err)
			return
		}
		defer detailedLog.Close()

		basicLog, err := os.OpenFile(basicLogName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to create basic log file: %v\n", err)
			return
		}
		defer basicLog.Close()

		// Write detailed header
		detailedHeader := fmt.Sprintf("╔═══════════════════════════════════════════════════════════════════════╗\n")
		detailedHeader += fmt.Sprintf("║                    AEGIS DIRECTORY WATCH SESSION                      ║\n")
		detailedHeader += fmt.Sprintf("╚═══════════════════════════════════════════════════════════════════════╝\n")
		detailedHeader += fmt.Sprintf("📁 Directory: %s\n", dir)
		detailedHeader += fmt.Sprintf("🕐 Started: %s\n", time.Now().Format("2006-01-02 15:04:05"))
		detailedHeader += fmt.Sprintf("📝 Detailed Log: %s\n", detailedLogName)
		detailedHeader += fmt.Sprintf("📋 Basic Log: %s\n", basicLogName)
		detailedHeader += fmt.Sprintf("═══════════════════════════════════════════════════════════════════════\n\n")
		fmt.Print(detailedHeader)
		detailedLog.WriteString(detailedHeader)

		// Write basic header
		basicHeader := fmt.Sprintf("AEGIS WATCH LOG - %s\n", time.Now().Format("2006-01-02 15:04:05"))
		basicHeader += fmt.Sprintf("Directory: %s\n", dir)
		basicHeader += fmt.Sprintf("Format: [Action] File | Timestamp\n")
		basicHeader += fmt.Sprintf("═══════════════════════════════════════════════════════════════════════\n\n")
		basicLog.WriteString(basicHeader)

		// Initialize file tracker
		tracker := newFileTracker()

		// Create initial snapshots of all files
		initMsg := "📸 Taking initial snapshots of all files...\n"
		fmt.Print(initMsg)
		detailedLog.WriteString(initMsg)
		if err := createInitialSnapshots(tracker, dir); err != nil {
			msg := fmt.Sprintf("⚠️  Warning: Could not create initial snapshots: %v\n", err)
			fmt.Fprint(os.Stderr, msg)
			detailedLog.WriteString(msg)
		}

		// Create file watcher
		watcher, err := fsnotify.NewWatcher()
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ Failed to create watcher: %v\n", err)
			return
		}
		defer watcher.Close()

		// Add directory and all subdirectories to watcher
		if err := addDirRecursive(watcher, dir); err != nil {
			fmt.Fprintf(os.Stderr, "❌ Failed to add directory to watcher: %v\n", err)
			return
		}

		watchMsg := fmt.Sprintf("👀 Watching for changes... (Press Ctrl+C to stop)\n")
		watchMsg += fmt.Sprintf("═══════════════════════════════════════════════════════════════════════\n\n")
		fmt.Print(watchMsg)
		detailedLog.WriteString(watchMsg)

		// Watch for events
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}

				// Filter out events for .aegis files and log files
				if strings.HasSuffix(event.Name, ".aegis") ||
					strings.HasPrefix(filepath.Base(event.Name), "watch_log_") ||
					strings.HasPrefix(filepath.Base(event.Name), "watch_detailed_") ||
					strings.HasPrefix(filepath.Base(event.Name), "watch_basic_") {
					continue
				}

				// Skip directories
				if info, err := os.Stat(event.Name); err == nil && info.IsDir() {
					if event.Has(fsnotify.Create) {
						addDirRecursive(watcher, event.Name)
					}
					continue
				}

				// Get current timestamp
				timestamp := time.Now().Format("2006-01-02 15:04:05")
				relPath, _ := filepath.Rel(dir, event.Name)

				// Handle different event types
				switch {
				case event.Has(fsnotify.Write):
					// Detailed log format
					detailedMsg := fmt.Sprintf("\n┌─── FILE MODIFIED ───────────────────────────────────────────\n")
					detailedMsg += fmt.Sprintf("│ 📝 Time: %s\n", timestamp)
					detailedMsg += fmt.Sprintf("│ 📄 File: %s\n", relPath)
					detailedMsg += fmt.Sprintf("└─────────────────────────────────────────────────────────────\n")
					fmt.Print(detailedMsg)
					detailedLog.WriteString(detailedMsg)

					// Detect changes and gather summary for basic log
					summary := detectAndShowChanges(tracker, event.Name, detailedLog, basicLog)
					// Only write to basic log if there were actual content changes
					if summary.hasChanges {
						lineSpec := summary.lineSpec
						if lineSpec == "" {
							lineSpec = "-"
						}
						basicLog.WriteString(fmt.Sprintf("[Modified] %s | %s | size %d bytes | lines %s\n", relPath, timestamp, summary.newSize, lineSpec))
					}

				case event.Has(fsnotify.Create):
					// Detailed log format
					detailedMsg := fmt.Sprintf("\n┌─── FILE CREATED ────────────────────────────────────────────\n")
					detailedMsg += fmt.Sprintf("│ ➕ Time: %s\n", timestamp)
					detailedMsg += fmt.Sprintf("│ 📄 File: %s\n", relPath)
					detailedMsg += fmt.Sprintf("└─────────────────────────────────────────────────────────────\n")
					fmt.Print(detailedMsg)
					detailedLog.WriteString(detailedMsg)

					// Detailed processing and summary
					summary := showNewFileContent(event.Name, detailedLog, basicLog)
					lineSpec := summary.lineSpec
					if lineSpec == "" {
						lineSpec = "-"
					}
					basicLog.WriteString(fmt.Sprintf("[Created] %s | %s | size %d bytes | lines %s\n", relPath, timestamp, summary.newSize, lineSpec))
					tracker.addSnapshot(event.Name)

				case event.Has(fsnotify.Remove):
					// Detailed log format
					detailedMsg := fmt.Sprintf("\n┌─── FILE REMOVED ────────────────────────────────────────────\n")
					detailedMsg += fmt.Sprintf("│ ➖ Time: %s\n", timestamp)
					detailedMsg += fmt.Sprintf("│ 📄 File: %s\n", relPath)
					detailedMsg += fmt.Sprintf("└─────────────────────────────────────────────────────────────\n\n")
					fmt.Print(detailedMsg)
					detailedLog.WriteString(detailedMsg)

					// Basic log format
					basicLog.WriteString(fmt.Sprintf("[Removed] %s | %s | size 0 bytes | lines -\n", relPath, timestamp))

					tracker.removeSnapshot(event.Name)

				case event.Has(fsnotify.Rename):
					// Detailed log format
					detailedMsg := fmt.Sprintf("\n┌─── FILE RENAMED ────────────────────────────────────────────\n")
					detailedMsg += fmt.Sprintf("│ 🔄 Time: %s\n", timestamp)
					detailedMsg += fmt.Sprintf("│ 📄 File: %s\n", relPath)
					detailedMsg += fmt.Sprintf("└─────────────────────────────────────────────────────────────\n\n")
					fmt.Print(detailedMsg)
					detailedLog.WriteString(detailedMsg)

					// Basic log format
					basicLog.WriteString(fmt.Sprintf("[Renamed] %s | %s\n", relPath, timestamp))
				}

			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				msg := fmt.Sprintf("⚠️  Watcher error: %v\n", err)
				fmt.Fprint(os.Stderr, msg)
				detailedLog.WriteString(msg)
				basicLog.WriteString(msg)
			}
		}
	},
}

// addDirRecursive adds a directory and all its subdirectories to the watcher
func addDirRecursive(watcher *fsnotify.Watcher, dir string) error {
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			// Skip common directories that shouldn't be watched
			if shouldExcludeDir(info.Name()) && path != dir {
				return filepath.SkipDir
			}
			if err := watcher.Add(path); err != nil {
				return fmt.Errorf("failed to watch %s: %v", path, err)
			}
		}
		return nil
	})
}

// shouldExcludeDir checks if a directory should be excluded from watching
func shouldExcludeDir(name string) bool {
	excludeList := []string{".git", "vendor", "node_modules", "target", ".idea", ".vscode"}
	for _, excluded := range excludeList {
		if name == excluded {
			return true
		}
	}
	return false
}

// addSnapshot adds or updates a file snapshot
func (ft *fileTracker) addSnapshot(path string) error {
	ft.mu.Lock()
	defer ft.mu.Unlock()

	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	info, err := os.Stat(path)
	if err != nil {
		return err
	}

	lines := strings.Split(string(content), "\n")
	hash := sha256.Sum256(content)

	ft.snapshots[path] = &fileSnapshot{
		content: content,
		hash:    hash,
		lines:   lines,
		modTime: info.ModTime(),
	}

	return nil
}

// removeSnapshot removes a file snapshot
func (ft *fileTracker) removeSnapshot(path string) {
	ft.mu.Lock()
	defer ft.mu.Unlock()
	delete(ft.snapshots, path)
}

// getSnapshot retrieves a file snapshot
func (ft *fileTracker) getSnapshot(path string) (*fileSnapshot, bool) {
	ft.mu.RLock()
	defer ft.mu.RUnlock()
	snapshot, exists := ft.snapshots[path]
	return snapshot, exists
}

// createInitialSnapshots creates snapshots of all files in directory
func createInitialSnapshots(tracker *fileTracker, dir string) error {
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		if info.IsDir() {
			if shouldExcludeDir(info.Name()) && path != dir {
				return filepath.SkipDir
			}
			return nil
		}

		// Skip symlinks and .aegis files
		if (info.Mode()&os.ModeSymlink) != 0 || strings.HasSuffix(path, ".aegis") {
			return nil
		}

		tracker.addSnapshot(path)
		return nil
	})
}

// detectAndShowChanges detects and displays line-by-line changes in a file
func detectAndShowChanges(tracker *fileTracker, path string, detailedLog *os.File, basicLog *os.File) changeSummary {
	oldSnapshot, exists := tracker.getSnapshot(path)

	content, err := os.ReadFile(path)
	if err != nil {
		msg := fmt.Sprintf("│ ⚠️  Could not read file: %v\n", err)
		fmt.Print(msg)
		detailedLog.WriteString(msg)
		basicLog.WriteString(msg)
		return changeSummary{newSize: 0, lineSpec: "-", hasChanges: false}
	}

	newLines := strings.Split(string(content), "\n")
	newSize := len(content)

	if !exists {
		msg := fmt.Sprintf("│ 📄 New file with %d lines\n\n", len(newLines))
		fmt.Print(msg)
		detailedLog.WriteString(msg)
		basicLog.WriteString(msg)
		tracker.addSnapshot(path)
		return changeSummary{newSize: newSize, lineSpec: formatLineRangeFromCount(len(newLines)), hasChanges: len(newLines) > 0}
	}

	newHash := sha256.Sum256(content)
	if bytes.Equal(oldSnapshot.hash[:], newHash[:]) {
		msg := "│ ℹ️  File metadata changed but content is identical\n\n"
		fmt.Print(msg)
		detailedLog.WriteString(msg)
		return changeSummary{newSize: newSize, lineSpec: "-", hasChanges: false}
	}

	oldLines := oldSnapshot.lines
	changedLines := []int{}
	addedLines := []int{}
	removedLines := []int{}

	for i := 0; i < len(oldLines); i++ {
		if i >= len(newLines) {
			removedLines = append(removedLines, i+1)
		} else if oldLines[i] != newLines[i] {
			changedLines = append(changedLines, i+1)
		}
	}

	if len(newLines) > len(oldLines) {
		for i := len(oldLines); i < len(newLines); i++ {
			addedLines = append(addedLines, i+1)
		}
	}

	oldSize := len(oldSnapshot.content)
	sizeDiff := newSize - oldSize

	summaryMsg := fmt.Sprintf("│ 📊 Summary: ")
	if len(changedLines) > 0 {
		summaryMsg += fmt.Sprintf("%d line(s) modified, ", len(changedLines))
	}
	if len(addedLines) > 0 {
		summaryMsg += fmt.Sprintf("%d line(s) added, ", len(addedLines))
	}
	if len(removedLines) > 0 {
		summaryMsg += fmt.Sprintf("%d line(s) removed, ", len(removedLines))
	}
	if sizeDiff > 0 {
		summaryMsg += fmt.Sprintf("size +%d bytes", sizeDiff)
	} else if sizeDiff < 0 {
		summaryMsg += fmt.Sprintf("size %d bytes", sizeDiff)
	} else {
		summaryMsg += "no size change"
	}
	summaryMsg += "\n"

	fmt.Print(summaryMsg)
	detailedLog.WriteString(summaryMsg)

	lineSet := make(map[int]struct{})
	for _, lineList := range [][]int{changedLines, addedLines, removedLines} {
		for _, line := range lineList {
			if line <= 0 {
				continue
			}
			lineSet[line] = struct{}{}
		}
	}

	lineIndices := make([]int, 0, len(lineSet))
	for line := range lineSet {
		lineIndices = append(lineIndices, line)
	}
	sort.Ints(lineIndices)
	lineSpec := formatLineRanges(lineIndices)
	if lineSpec == "" {
		lineSpec = "-"
	}

	if len(changedLines) > 0 {
		detailedMsg := fmt.Sprintf("│\n│ ✏️  Modified Lines: %v\n", changedLines)
		fmt.Print(detailedMsg)
		detailedLog.WriteString(detailedMsg)

		for _, lineNum := range changedLines {
			idx := lineNum - 1
			if idx < len(oldLines) && idx < len(newLines) {
				detailedMsg = fmt.Sprintf("│   • Line %d:\n", lineNum)
				fmt.Print(detailedMsg)
				detailedLog.WriteString(detailedMsg)

				detailedMsg = fmt.Sprintf("│     [-] %s\n", truncate(oldLines[idx], 70))
				fmt.Print(detailedMsg)
				detailedLog.WriteString(detailedMsg)

				detailedMsg = fmt.Sprintf("│     [+] %s\n", truncate(newLines[idx], 70))
				fmt.Print(detailedMsg)
				detailedLog.WriteString(detailedMsg)

				charChanges := detectCharacterChanges(oldLines[idx], newLines[idx])
				if charChanges != "" {
					detailedMsg = fmt.Sprintf("│     🔤  %s\n", charChanges)
					fmt.Print(detailedMsg)
					detailedLog.WriteString(detailedMsg)
				}
			}
		}
	}

	if len(addedLines) > 0 {
		detailedMsg := fmt.Sprintf("│\n│ ➕ Added Lines: %v\n", addedLines)
		fmt.Print(detailedMsg)
		detailedLog.WriteString(detailedMsg)

		for _, lineNum := range addedLines {
			idx := lineNum - 1
			if idx < len(newLines) {
				detailedMsg = fmt.Sprintf("│   • Line %d: %s\n", lineNum, truncate(newLines[idx], 70))
				fmt.Print(detailedMsg)
				detailedLog.WriteString(detailedMsg)
			}
		}
	}

	if len(removedLines) > 0 {
		detailedMsg := fmt.Sprintf("│\n│ ➖ Removed Lines: %v\n", removedLines)
		fmt.Print(detailedMsg)
		detailedLog.WriteString(detailedMsg)
	}

	closingMsg := "└─────────────────────────────────────────────────────────────\n\n"
	fmt.Print(closingMsg)
	detailedLog.WriteString(closingMsg)

	tracker.addSnapshot(path)

	return changeSummary{
		newSize:    newSize,
		lineSpec:   lineSpec,
		hasChanges: len(lineIndices) > 0,
	}
}

// detectCharacterChanges detects and describes character-level changes between two strings
func detectCharacterChanges(oldStr, newStr string) string {
	if oldStr == newStr {
		return ""
	}

	oldRunes := []rune(oldStr)
	newRunes := []rune(newStr)

	// Simple character addition detection
	if len(newRunes) > len(oldRunes) {
		// Check if characters were added
		added := len(newRunes) - len(oldRunes)

		// Try to find where characters were added
		if len(oldRunes) == 0 {
			return fmt.Sprintf("Added %d character(s): '%s'", added, string(newRunes))
		}

		// Check if added at the end
		if len(oldRunes) <= len(newRunes) && string(oldRunes) == string(newRunes[:len(oldRunes)]) {
			addedChars := newRunes[len(oldRunes):]
			return fmt.Sprintf("Added %d character(s) at end: '%s'", added, string(addedChars))
		}

		// Check if added at the beginning
		if len(oldRunes) <= len(newRunes) && string(oldRunes) == string(newRunes[len(newRunes)-len(oldRunes):]) {
			addedChars := newRunes[:added]
			return fmt.Sprintf("Added %d character(s) at beginning: '%s'", added, string(addedChars))
		}

		// Added somewhere in the middle
		return fmt.Sprintf("Added %d character(s)", added)
	}

	// Character deletion detection
	if len(newRunes) < len(oldRunes) {
		deleted := len(oldRunes) - len(newRunes)

		// Check if deleted from the end
		if len(newRunes) <= len(oldRunes) && string(newRunes) == string(oldRunes[:len(newRunes)]) {
			deletedChars := oldRunes[len(newRunes):]
			return fmt.Sprintf("Deleted %d character(s) from end: '%s'", deleted, string(deletedChars))
		}

		// Check if deleted from the beginning
		if len(newRunes) <= len(oldRunes) && string(newRunes) == string(oldRunes[deleted:]) {
			deletedChars := oldRunes[:deleted]
			return fmt.Sprintf("Deleted %d character(s) from beginning: '%s'", deleted, string(deletedChars))
		}

		return fmt.Sprintf("Deleted %d character(s)", deleted)
	}

	// Same length but different content - character replacement
	diffCount := 0
	var changedPositions []int
	for i := 0; i < len(oldRunes) && i < len(newRunes); i++ {
		if oldRunes[i] != newRunes[i] {
			diffCount++
			if len(changedPositions) < 3 {
				changedPositions = append(changedPositions, i)
			}
		}
	}

	if diffCount == 1 && len(changedPositions) > 0 {
		pos := changedPositions[0]
		return fmt.Sprintf("Changed character at position %d: '%c' → '%c'", pos+1, oldRunes[pos], newRunes[pos])
	}

	return fmt.Sprintf("Changed %d character(s)", diffCount)
}

// showNewFileContent displays information about a newly created file
func showNewFileContent(path string, detailedLog *os.File, basicLog *os.File) changeSummary {
	// Retry logic for Windows file locking issues
	var content []byte
	var err error
	maxRetries := 5
	retryDelay := time.Millisecond * 100

	for attempt := 0; attempt < maxRetries; attempt++ {
		content, err = os.ReadFile(path)
		if err == nil {
			break
		}

		// Only retry on file access/locking errors
		if attempt < maxRetries-1 {
			time.Sleep(retryDelay)
			retryDelay *= 2 // Exponential backoff: 100ms, 200ms, 400ms, 800ms
		}
	}

	if err != nil {
		msg := fmt.Sprintf("│ ⚠️  Could not read file: %v\n└─────────────────────────────────────────────────────────────\n\n", err)
		fmt.Print(msg)
		detailedLog.WriteString(msg)
		basicLog.WriteString(msg)
		return changeSummary{newSize: 0, lineSpec: "-", hasChanges: false}
	}

	lines := strings.Split(string(content), "\n")

	// Basic info for detailed log and terminal
	basicMsg := fmt.Sprintf("│ 📊 Size: %d bytes, %d line(s)\n", len(content), len(lines))
	fmt.Print(basicMsg)
	detailedLog.WriteString(basicMsg)
	// Don't write size info to basic log

	// Show first few lines if it's a text file (detailed only)
	if isTextFile(content) && len(lines) > 0 {
		detailedMsg := "│\n│ 📝 Content Preview:\n"
		fmt.Print(detailedMsg)
		detailedLog.WriteString(detailedMsg)

		previewLines := 5
		if len(lines) < previewLines {
			previewLines = len(lines)
		}
		for i := 0; i < previewLines; i++ {
			if lines[i] != "" {
				detailedMsg = fmt.Sprintf("│   %d: %s\n", i+1, truncate(lines[i], 70))
				fmt.Print(detailedMsg)
				detailedLog.WriteString(detailedMsg)
			}
		}
		if len(lines) > previewLines {
			detailedMsg = fmt.Sprintf("│   ... (%d more lines)\n", len(lines)-previewLines)
			fmt.Print(detailedMsg)
			detailedLog.WriteString(detailedMsg)
		}
	}

	closingMsg := "└─────────────────────────────────────────────────────────────\n\n"
	fmt.Print(closingMsg)
	detailedLog.WriteString(closingMsg)
	// Don't write closing box to basic log

	return changeSummary{
		newSize:    len(content),
		lineSpec:   formatLineRangeFromCount(len(lines)),
		hasChanges: len(lines) > 0,
	}
}

func formatLineRanges(lines []int) string {
	if len(lines) == 0 {
		return "-"
	}

	b := &strings.Builder{}
	start := lines[0]
	prev := start

	appendRange := func(s, e int) {
		if b.Len() > 0 {
			b.WriteString(",")
		}
		if s == e {
			fmt.Fprintf(b, "%d", s)
		} else {
			fmt.Fprintf(b, "%d-%d", s, e)
		}
	}

	for i := 1; i < len(lines); i++ {
		curr := lines[i]
		if curr == prev {
			continue
		}
		if curr == prev+1 {
			prev = curr
			continue
		}
		appendRange(start, prev)
		start = curr
		prev = curr
	}
	appendRange(start, prev)

	return b.String()
}

func formatLineRangeFromCount(count int) string {
	if count <= 0 {
		return "-"
	}
	if count == 1 {
		return "1"
	}
	return fmt.Sprintf("1-%d", count)
}

// truncate truncates a string to a maximum length
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// isTextFile checks if content appears to be text
func isTextFile(content []byte) bool {
	if len(content) == 0 {
		return true
	}

	// Check for null bytes (binary indicator)
	for i := 0; i < len(content) && i < 512; i++ {
		if content[i] == 0 {
			return false
		}
	}

	// Check if most bytes are printable
	printable := 0
	for i := 0; i < len(content) && i < 512; i++ {
		if content[i] >= 32 && content[i] <= 126 || content[i] == '\n' || content[i] == '\r' || content[i] == '\t' {
			printable++
		}
	}

	checkLen := len(content)
	if checkLen > 512 {
		checkLen = 512
	}

	return float64(printable)/float64(checkLen) > 0.85
}

func init() {
	RootCmd.AddCommand(watchCmd)
}
