package cmd

import (
	"fmt"
	"sort"

	"llmsh/pkg/tracker"

	"github.com/spf13/cobra"
)

var statsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show token usage statistics",
	Long:  `Display token usage statistics aggregated by day and method.`,
	RunE:  runStats,
}

func runStats(cmd *cobra.Command, args []string) error {
	// Load token records
	storage, err := tracker.LoadRecords()
	if err != nil {
		fmt.Printf("Error loading records: %v\n", err)
		return err
	}

	if len(storage.Records) == 0 {
		fmt.Println("No usage records found.")
		return nil
	}

	// Aggregate by day
	dayStats := tracker.AggregateByDay(storage.Records)

	// Aggregate by method
	methodStats := tracker.AggregateByMethod(storage.Records)

	// Aggregate by provider/model
	providerModelStats := tracker.AggregateByProviderModel(storage.Records)

	// Print statistics
	fmt.Println("Token Usage Statistics")
	fmt.Println("======================")
	fmt.Println()

	// Print by provider and model
	fmt.Println("Usage by Provider/Model:")
	fmt.Println("------------------------")

	for _, stat := range providerModelStats {
		fmt.Printf("%s / %s:\n", stat.Provider, stat.Model)
		fmt.Printf("  Requests:      %d\n", stat.Count)
		fmt.Printf("  Input Tokens:  %d\n", stat.InputTokens)
		fmt.Printf("  Output Tokens: %d\n", stat.OutputTokens)
		if stat.CacheReadTokens > 0 {
			fmt.Printf("  Cache Read:    %d\n", stat.CacheReadTokens)
		}
		fmt.Println()
	}

	// Print by day
	fmt.Println("Usage by Day:")
	fmt.Println("-------------")

	// Sort days
	days := make([]string, 0, len(dayStats))
	for day := range dayStats {
		days = append(days, day)
	}
	sort.Strings(days)

	totalRequests := 0
	totalInput := 0
	totalOutput := 0
	totalCacheRead := 0

	for _, day := range days {
		stat := dayStats[day]
		fmt.Printf("%s:\n", day)
		fmt.Printf("  Requests:      %d\n", stat.Count)
		fmt.Printf("  Input Tokens:  %d\n", stat.InputTokens)
		fmt.Printf("  Output Tokens: %d\n", stat.OutputTokens)
		if stat.CacheReadTokens > 0 {
			fmt.Printf("  Cache Read:    %d\n", stat.CacheReadTokens)
		}
		fmt.Println()

		totalRequests += stat.Count
		totalInput += stat.InputTokens
		totalOutput += stat.OutputTokens
		totalCacheRead += stat.CacheReadTokens
	}

	// Print by method
	fmt.Println("Usage by Method:")
	fmt.Println("----------------")

	for _, stat := range methodStats {
		fmt.Printf("%s:\n", stat.Method)
		fmt.Printf("  Requests:      %d\n", stat.Count)
		fmt.Printf("  Input Tokens:  %d\n", stat.InputTokens)
		fmt.Printf("  Output Tokens: %d\n", stat.OutputTokens)
		fmt.Println()
	}

	// Print totals
	fmt.Println("Total Summary:")
	fmt.Println("--------------")
	fmt.Printf("  Total Requests:      %d\n", totalRequests)
	fmt.Printf("  Total Input Tokens:  %d\n", totalInput)
	fmt.Printf("  Total Output Tokens: %d\n", totalOutput)
	if totalCacheRead > 0 {
		fmt.Printf("  Total Cache Read:    %d\n", totalCacheRead)
		// Calculate potential savings from cache
		savingsPercent := float64(totalCacheRead) / float64(totalInput+totalCacheRead) * 100
		fmt.Printf("  Cache Savings:       %.1f%%\n", savingsPercent)
	}

	return nil
}
