package ui

import (
	"fmt"
	"nmod-cleaner/internal/analyzer"
	"nmod-cleaner/internal/scanner"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
)

var (
	SuccessStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))  // Green
	InfoStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("39"))  // Blue
	WarningStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("214")) // Yellow
	ErrorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("196")) // Red
	BoldStyle    = lipgloss.NewStyle().Bold(true)
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

type CleanableTarget struct {
	scanner.ScanResult
	Size int64
}

// PromptForDirectories returns the subset of targets the user selected
func PromptForDirectories(targets []CleanableTarget) ([]CleanableTarget, error) {
	if len(targets) == 0 {
		return nil, nil
	}

	options := make([]huh.Option[CleanableTarget], len(targets))
	for i, t := range targets {
		label := fmt.Sprintf("%s (%s)", t.ProjectPath, analyzer.FormatBytes(t.Size))
		options[i] = huh.NewOption[CleanableTarget](label, t).Selected(true)
	}

	var selected []CleanableTarget

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewMultiSelect[CleanableTarget]().
				Title("Select node_modules directories to clean and migrate to pnpm:").
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
