package ui

import (
	"fmt"
	"nmod-cleaner/internal/analyzer"
	"nmod-cleaner/internal/scanner"
	"strings"
	"sync"
	"sync/atomic"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
)

// ─── Styles ───────────────────────────────────────────────────────────────────

var (
	SuccessStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))  // Green
	InfoStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("39"))  // Blue
	WarningStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("214")) // Yellow
	ErrorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("196")) // Red
	BoldStyle    = lipgloss.NewStyle().Bold(true)
	DimStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
)

func PrintSuccess(msg string) {
	fmt.Println(SuccessStyle.Render("✔ " + msg))
}

func PrintInfo(msg string) {
	fmt.Println(InfoStyle.Render("ℹ " + msg))
}

func PrintError(msg string) {
	fmt.Println(ErrorStyle.Render("✖ " + msg))
}

func PrintSummary(succeededCount int, failedCount int, savedBytes int64) {
	fmt.Println("\n--- Summary ---")
	fmt.Printf("Directories Cleaned: %s\n", BoldStyle.Render(fmt.Sprintf("%d", succeededCount)))
	if failedCount > 0 {
		fmt.Printf("Directories Failed:  %s\n", BoldStyle.Render(ErrorStyle.Render(fmt.Sprintf("%d", failedCount))))
	}
	fmt.Printf("Disk Space Freed: %s\n", BoldStyle.Render(SuccessStyle.Render(analyzer.FormatBytes(savedBytes))))
}

// ─── CleanableTarget ──────────────────────────────────────────────────────────

type CleanableTarget struct {
	scanner.ScanResult
	Size int64
}

// ─── TUI Prompt ───────────────────────────────────────────────────────────────

// PromptForDirectories returns the subset of targets the user selected
func PromptForDirectories(targets []CleanableTarget) ([]CleanableTarget, error) {
	if len(targets) == 0 {
		return nil, nil
	}

	options := make([]huh.Option[CleanableTarget], len(targets))
	for i, t := range targets {
		label := fmt.Sprintf("%s (%s)  [%s]", t.ProjectPath, analyzer.FormatBytes(t.Size), t.PackageManager)
		options[i] = huh.NewOption(label, t).Selected(true)
	}

	var selected []CleanableTarget

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewMultiSelect[CleanableTarget]().
				Title("Select node_modules directories to clean:").
				Options(options...).
				Height(15).
				Value(&selected),
		),
	)

	err := form.Run()
	if err != nil {
		return nil, err
	}

	return selected, nil
}

// ─── Progress View (Bubbletea) ────────────────────────────────────────────────

type projectStatus int

const (
	statusPending    projectStatus = iota
	statusDeleting                 // deleting node_modules
	statusInstalling               // running package manager install
	statusDone
	statusFailed
)

type progressRow struct {
	path   string
	pm     string
	status projectStatus
	err    error
}

type rowUpdateMsg struct {
	index  int
	status projectStatus
	err    error
}

type doneMsg struct{}

type progressModel struct {
	rows     []progressRow
	done     bool
	quitting bool
}

func (m progressModel) Init() tea.Cmd { return nil }

func (m progressModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case rowUpdateMsg:
		m.rows[msg.index].status = msg.status
		m.rows[msg.index].err = msg.err
	case doneMsg:
		m.done = true
		return m, tea.Quit
	case tea.KeyMsg:
		// allow ctrl+c bailout
		if msg.Type == tea.KeyCtrlC {
			m.quitting = true
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m progressModel) View() string {
	var sb strings.Builder
	sb.WriteString(BoldStyle.Render("\n  Cleaning projects…\n\n"))
	for _, r := range m.rows {
		var icon, label string
		switch r.status {
		case statusPending:
			icon = DimStyle.Render("  ○")
			label = DimStyle.Render(r.path)
		case statusDeleting:
			icon = WarningStyle.Render("  ⠸")
			label = WarningStyle.Render(r.path + "  deleting…")
		case statusInstalling:
			icon = InfoStyle.Render("  ⠸")
			label = InfoStyle.Render(r.path + fmt.Sprintf("  running %s install…", r.pm))
		case statusDone:
			icon = SuccessStyle.Render("  ✔")
			label = SuccessStyle.Render(r.path + "  done")
		case statusFailed:
			icon = ErrorStyle.Render("  ✖")
			label = ErrorStyle.Render(r.path + "  failed")
		}
		sb.WriteString(icon + "  " + label + "\n")
	}
	return sb.String()
}

// ProcessFunc is the work to be done per target. Returns the bytes freed and any error.
type ProcessFunc func(target CleanableTarget) (int64, error)

// RunProgressView runs a live Bubbletea progress UI while processFn is called
// concurrently on each target. Returns aggregate success count, fail count, and
// total bytes freed.
func RunProgressView(targets []CleanableTarget, processFn ProcessFunc) (succeeded, failed int, totalSaved int64) {
	rows := make([]progressRow, len(targets))
	for i, t := range targets {
		rows[i] = progressRow{path: t.ProjectPath, pm: t.PackageManager, status: statusPending}
	}

	m := progressModel{rows: rows}
	p := tea.NewProgram(m)

	var wg sync.WaitGroup
	var mu sync.Mutex
	var succeededAtomic, failedAtomic int32
	var savedAtomic int64

	for i, t := range targets {
		wg.Add(1)
		go func(idx int, target CleanableTarget) {
			defer wg.Done()

			// Phase 1: deleting
			p.Send(rowUpdateMsg{index: idx, status: statusDeleting})

			bytes, err := processFn(target)
			if err != nil {
				p.Send(rowUpdateMsg{index: idx, status: statusFailed, err: err})
				mu.Lock()
				atomic.AddInt32(&failedAtomic, 1)
				mu.Unlock()
				return
			}

			// Phase 2: installing (processFn handles both; we just signal done)
			p.Send(rowUpdateMsg{index: idx, status: statusInstalling})
			// The installing signal is cosmetic here — processFn already ran install.
			// For a future enhancement, split processFn into two phases.

			p.Send(rowUpdateMsg{index: idx, status: statusDone})
			atomic.AddInt64(&savedAtomic, bytes)
			atomic.AddInt32(&succeededAtomic, 1)
		}(i, t)
	}

	// Wait for all goroutines then signal done so the TUI quits cleanly
	go func() {
		wg.Wait()
		p.Send(doneMsg{})
	}()

	if _, err := p.Run(); err != nil {
		// If the TUI fails, fall back gracefully — work was already done
		fmt.Println(ErrorStyle.Render("Progress UI error (work still completed): " + err.Error()))
	}

	return int(atomic.LoadInt32(&succeededAtomic)),
		int(atomic.LoadInt32(&failedAtomic)),
		atomic.LoadInt64(&savedAtomic)
}
