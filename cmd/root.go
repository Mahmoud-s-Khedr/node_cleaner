package cmd

import (
	"fmt"
	"nmod-cleaner/internal/analyzer"
	"nmod-cleaner/internal/cleaner"
	"nmod-cleaner/internal/config"
	"nmod-cleaner/internal/history"
	"nmod-cleaner/internal/installer"
	"nmod-cleaner/internal/scanner"
	"nmod-cleaner/internal/ui"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/briandowns/spinner"
	"github.com/spf13/cobra"
)

var (
	isDryRun    bool
	scanPath    string
	configPath  string
	showHistory bool
)

func NewSpinner(msg string) *spinner.Spinner {
	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	s.Suffix = " " + msg
	return s
}

var rootCmd = &cobra.Command{
	Use:   "nmod-cleaner",
	Short: "A CLI to safely remove bloated node_modules and reinstall dependencies.",
	RunE: func(cmd *cobra.Command, args []string) error {
		// ── Feature 5: --history flag ─────────────────────────────────
		if showHistory {
			records, err := history.Load()
			if err != nil {
				ui.PrintError(fmt.Sprintf("Failed to load history: %v", err))
				return nil
			}
			history.PrintSummary(records)
			return nil
		}

		absPath, err := filepath.Abs(scanPath)
		if err != nil {
			return err
		}

		// ── Feature 4: load config ────────────────────────────────────
		cfg, err := config.Load(configPath)
		if err != nil {
			ui.PrintError(fmt.Sprintf("Failed to load config: %v", err))
			return nil
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

		// Filter: skip pnpm-managed and config skip-listed projects.
		var unoptimizedResults []scanner.ScanResult
		for _, r := range scanResults {
			if r.IsPnpm {
				continue
			}
			if cfg.ShouldSkip(r.ProjectPath) {
				ui.PrintInfo(fmt.Sprintf("Skipping (config): %s", r.ProjectPath))
				continue
			}
			unoptimizedResults = append(unoptimizedResults, r)
		}

		if len(unoptimizedResults) == 0 {
			ui.PrintInfo("No cleanup needed — all node_modules are either pnpm-managed or on your skip-list.")
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

		// ── Dry-run mode ──────────────────────────────────────────────
		if isDryRun {
			ui.PrintInfo(ui.BoldStyle.Render("\n--- DRY RUN MODE ---"))
			var totalSaved int64
			var count int32
			for _, target := range selectedTargets {
				ui.PrintInfo(fmt.Sprintf("Would delete: %s", target.NodeModulesPath))
				ui.PrintInfo(fmt.Sprintf("Would run '%s install' in: %s", target.PackageManager, target.ProjectPath))
				totalSaved += target.Size
				count++
			}
			ui.PrintSummary(int(count), 0, totalSaved)
			return nil
		}

		// ── Live cleanup with progress TUI (Feature 2) ────────────────
		fmt.Println("")

		processFn := func(target ui.CleanableTarget) (int64, error) {
			if err := cleaner.DeleteNodeModules(target.NodeModulesPath); err != nil {
				return 0, fmt.Errorf("delete failed for %s: %w", target.NodeModulesPath, err)
			}
			if err := installer.RunInstall(target.ProjectPath, target.PackageManager); err != nil {
				return 0, fmt.Errorf("install failed for %s: %w", target.ProjectPath, err)
			}
			return target.Size, nil
		}

		succeeded, failed, savedBytes := ui.RunProgressView(selectedTargets, processFn)
		ui.PrintSummary(succeeded, failed, savedBytes)
		ui.PrintSuccess("Cleanup and reinstallation complete!")

		// ── Feature 5: persist run to history ─────────────────────────
		_ = history.Append(history.Record{
			Timestamp:  time.Now(),
			DirsClean:  succeeded,
			DirsFailed: failed,
			BytesFreed: savedBytes,
		})

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
	rootCmd.Flags().StringVar(&configPath, "config", "", "path to config file (default: ~/.nmodcleanerrc)")
	rootCmd.Flags().BoolVar(&showHistory, "history", false, "print cumulative stats history and exit")
}
