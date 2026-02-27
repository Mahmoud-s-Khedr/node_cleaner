package cmd

import (
	"fmt"
	"nmod-cleaner/internal/analyzer"
	"nmod-cleaner/internal/cleaner"
	"nmod-cleaner/internal/installer"
	"nmod-cleaner/internal/scanner"
	"nmod-cleaner/internal/ui"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/briandowns/spinner"
	"github.com/spf13/cobra"
)

var (
	isDryRun bool
	scanPath string
)

func NewSpinner(msg string) *spinner.Spinner {
	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	s.Suffix = " " + msg
	return s
}

var rootCmd = &cobra.Command{
	Use:   "nmod-cleaner",
	Short: "A CLI to safely remove bloated node_modules and reinstall dependencies using pnpm.",
	RunE: func(cmd *cobra.Command, args []string) error {
		if _, err := exec.LookPath("pnpm"); err != nil {
			ui.PrintError("pnpm is not installed or not in your PATH.")
			ui.PrintInfo("Install it from https://pnpm.io/installation and try again.")
			return nil
		}

		absPath, err := filepath.Abs(scanPath)
		if err != nil {
			return err
		}

		s := NewSpinner(fmt.Sprintf("Scanning for node_modules in %s...", absPath))
		s.Start()

		scanResults, err := scanner.FindNodeModules(absPath)
		s.Stop()

		if err != nil {
			ui.PrintError(fmt.Sprintf("Scan failed: %v", err))
			return nil
		}

		ui.PrintSuccess(fmt.Sprintf("Found %d node_modules directories.", len(scanResults)))

		if len(scanResults) == 0 {
			ui.PrintInfo("No node_modules directories found. Clean and tidy!")
			return nil
		}

		var unoptimizedResults []scanner.ScanResult
		for _, r := range scanResults {
			if !r.IsPnpm {
				unoptimizedResults = append(unoptimizedResults, r)
			}
		}

		if len(unoptimizedResults) == 0 {
			ui.PrintInfo("All found node_modules directories are already optimally managed by pnpm. No cleanup needed!")
			return nil
		}

		s = NewSpinner("Analyzing disk usage...")
		s.Start()

		var targetPaths []string
		for _, r := range unoptimizedResults {
			targetPaths = append(targetPaths, r.NodeModulesPath)
		}

		sizeMap := analyzer.CalculateSizesConcurrently(targetPaths)
		s.Stop()
		ui.PrintSuccess("Analysis complete.")

		var targets []ui.CleanableTarget
		for _, r := range unoptimizedResults {
			size := sizeMap[r.NodeModulesPath]
			targets = append(targets, ui.CleanableTarget{ScanResult: r, Size: size})
		}

		// Sort targets by size descending
		sort.Slice(targets, func(i, j int) bool {
			return targets[i].Size > targets[j].Size
		})

		selectedTargets, err := ui.PromptForDirectories(targets)
		if err != nil {
			return fmt.Errorf("prompt failed: %w", err)
		}

		if len(selectedTargets) == 0 {
			ui.PrintInfo("Operation cancelled or no directories selected.")
			return nil
		}

		var totalSaved int64
		var deletedCount int32
		var failedCount int32

		if isDryRun {
			ui.PrintInfo(ui.BoldStyle.Render("\n--- DRY RUN MODE ---"))
			for _, target := range selectedTargets {
				ui.PrintInfo(fmt.Sprintf("Would delete: %s", target.NodeModulesPath))
				ui.PrintInfo(fmt.Sprintf("Would run pnpm install in: %s", target.ProjectPath))
				totalSaved += target.Size
				deletedCount++
			}
			ui.PrintSummary(int(deletedCount), 0, totalSaved)
			return nil
		}

		fmt.Println("")
		ui.PrintInfo(fmt.Sprintf("Starting concurrent processing for %d projects...", len(selectedTargets)))

		var wg sync.WaitGroup
		var mu sync.Mutex

		for _, t := range selectedTargets {
			wg.Add(1)
			go func(target ui.CleanableTarget) {
				defer wg.Done()

				err := cleaner.DeleteNodeModules(target.NodeModulesPath)
				if err != nil {
					mu.Lock()
					ui.PrintError(fmt.Sprintf("Delete failed for %s: %v", target.NodeModulesPath, err))
					mu.Unlock()
					atomic.AddInt32(&failedCount, 1)
					return
				}

				err = installer.RunPnpmInstall(target.ProjectPath)
				if err != nil {
					mu.Lock()
					ui.PrintError(fmt.Sprintf("Install failed for %s: %v", target.ProjectPath, err))
					mu.Unlock()
					atomic.AddInt32(&failedCount, 1)
					return
				}

				mu.Lock()
				ui.PrintSuccess(fmt.Sprintf("Cleaned and reinstalled: %s", target.ProjectPath))
				mu.Unlock()

				atomic.AddInt64(&totalSaved, target.Size)
				atomic.AddInt32(&deletedCount, 1)
			}(t)
		}

		wg.Wait()

		ui.PrintSummary(int(deletedCount), int(failedCount), totalSaved)
		ui.PrintSuccess("Cleanup and reinstallation complete!")

		return nil
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	cwd, _ := os.Getwd()
	rootCmd.Flags().BoolVarP(&isDryRun, "dry-run", "d", false, "simulate deletion without modifying the file system")
	rootCmd.Flags().StringVarP(&scanPath, "path", "p", cwd, "directory to scan for node_modules")
}
