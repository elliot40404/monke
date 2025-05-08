package main

import (
	"database/sql"
	"fmt"
	"log"
	"math"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

const (
	colorReset = "\033[0m"

	colorPast       = "\033[38;5;40m"  // All Past Dates (Dark Green)
	colorToday      = "\033[38;5;226m" // Today (Yellow 1)
	colorFutureNear = "\033[38;5;198m" // Future (1-3 days away) (Hot Pink)
	colorFutureMid  = "\033[38;5;208m" // Future (4-5 days away) (Orange 1)

	statusIndicator = "●"

	lineCharacter = "■"
)

var categoryColors = []string{
	"\033[38;5;21m",  // Deep Sky Blue 3
	"\033[38;5;51m",  // Cyan 1
	"\033[38;5;141m", // Medium Purple 3
	"\033[38;5;43m",  // Dark Sea Green 2
	"\033[38;5;178m", // Gold 1
	"\033[38;5;130m", // Orange 3
	"\033[38;5;108m", // Sea Green 3
	"\033[38;5;97m",  // Violet 1
	"\033[38;5;243m", // Gray 7
	"\033[38;5;65m",  // Medium Spring Green
}

var lsCmd = &cobra.Command{
	Use:   "ls",
	Short: "List all expenses",
	Run: func(_ *cobra.Command, _ []string) {
		rows, err := db.Query("SELECT id, title, amount, day, category FROM expenses ORDER BY day ASC")
		if err != nil {
			if strings.Contains(err.Error(), "no such column: day") {
				log.Fatalf("Error: Database schema mismatch. Clear the database with 'monke clear'.")
			}
			log.Fatalf("Error querying expenses: %v", err)
		}
		defer rows.Close()

		var expenses []Expense
		totalAmount := 0.0
		categoryTotalsMap := make(map[string]float64)
		uniqueCategories := make(map[string]struct{})
		totalLineWidth := 80 // Default width for the colored line

		for rows.Next() {
			var exp Expense
			var category sql.NullString

			err := rows.Scan(&exp.ID, &exp.Title, &exp.Amount, &exp.Day, &category)
			if err != nil {
				log.Printf("Error scanning row: %v", err)
				continue
			}

			displayCategory := "Uncategorized"
			if category.Valid && category.String != "" {
				exp.Category = category.String
				displayCategory = exp.Category
			} else {
				exp.Category = ""
			}
			categoryTotalsMap[displayCategory] += exp.Amount
			uniqueCategories[displayCategory] = struct{}{}

			expenses = append(expenses, exp)
			totalAmount += exp.Amount
		}

		if err = rows.Err(); err != nil {
			log.Fatalf("Error iterating rows: %v", err)
		}

		if len(expenses) == 0 {
			fmt.Println("No expenses found.")
			return
		}

		now := time.Now()
		currentDay := now.Day()
		currentMonthName := now.Format("January")

		categoryColorMap := make(map[string]string)
		var categoryNames []string
		for catName := range categoryTotalsMap {
			categoryNames = append(categoryNames, catName)
		}
		sort.Strings(categoryNames)

		for i, catName := range categoryNames {
			categoryColorMap[catName] = categoryColors[i%len(categoryColors)]
		}

		renderExpenseTable(expenses, totalAmount, categoryTotalsMap, currentDay, currentMonthName, categoryColorMap, totalLineWidth)
	},
}

func renderExpenseTable(expenses []Expense, totalAmount float64, categoryTotalsMap map[string]float64, currentDay int, currentMonthName string, categoryColorMap map[string]string, totalLineWidth int) {
	// Create new table
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Title", "Amount", "Date", "Category", "Status"})
	table.SetAutoWrapText(false)
	table.SetAutoFormatHeaders(true)
	table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)

	// Set different alignments per column
	table.SetColumnAlignment([]int{
		tablewriter.ALIGN_LEFT,   // Title - left aligned
		tablewriter.ALIGN_RIGHT,  // Amount - right aligned for numbers
		tablewriter.ALIGN_LEFT,   // Date - left aligned
		tablewriter.ALIGN_LEFT,   // Category - left aligned
		tablewriter.ALIGN_CENTER, // Status - center aligned
	})

	table.SetCenterSeparator("")
	table.SetColumnSeparator("")
	table.SetRowSeparator("")
	table.SetHeaderLine(true)
	table.SetBorder(false)
	table.SetTablePadding("\t") // pad with tabs
	table.SetNoWhiteSpace(true)

	// Add expense data to table
	for _, exp := range expenses {
		statusOutput := statusIndicator
		expenseDay := exp.Day

		if expenseDay == currentDay {
			statusOutput = colorToday + statusIndicator + colorReset
		} else if expenseDay < currentDay {
			statusOutput = colorPast + statusIndicator + colorReset
		} else {
			dayDiffFuture := expenseDay - currentDay
			if dayDiffFuture > 5 {
				statusOutput = statusIndicator
			} else if dayDiffFuture > 0 && dayDiffFuture <= 3 {
				statusOutput = colorFutureNear + statusIndicator + colorReset
			} else if dayDiffFuture > 3 && dayDiffFuture <= 5 {
				statusOutput = colorFutureMid + statusIndicator + colorReset
			} else {
				statusOutput = statusIndicator
			}
		}

		displayDateStr := fmt.Sprintf("%02d %s", expenseDay, currentMonthName)
		amountStr := fmt.Sprintf("%.2f", exp.Amount)

		displayCategory := exp.Category
		if displayCategory == "" {
			displayCategory = "Uncategorized"
		}

		categoryColor, ok := categoryColorMap[displayCategory]
		if !ok {
			categoryColor = colorReset
		}
		coloredCategory := fmt.Sprintf("%s%s%s", categoryColor, displayCategory, colorReset)

		table.Append([]string{exp.Title, amountStr, displayDateStr, coloredCategory, statusOutput})
	}

	// Render the table
	table.Render()

	// Generate and display category visualization line
	var categories []string
	for cat := range categoryTotalsMap {
		categories = append(categories, cat)
	}
	sort.Slice(categories, func(i, j int) bool {
		return categoryTotalsMap[categories[i]] > categoryTotalsMap[categories[j]]
	})

	coloredLine := generateColoredLine(categories, categoryTotalsMap, totalAmount, categoryColorMap, totalLineWidth)
	fmt.Println(coloredLine)

	printSummaryTotals(totalAmount, categories, categoryTotalsMap, categoryColorMap)
}

func generateColoredLine(categories []string, categoryTotalsMap map[string]float64, totalAmount float64, categoryColorMap map[string]string, totalLineWidth int) string {
	var coloredLine strings.Builder
	remainingWidth := totalLineWidth

	for i, cat := range categories {
		categoryTotal := categoryTotalsMap[cat]
		percentage := 0.0
		if totalAmount > 0 {
			percentage = (categoryTotal / totalAmount) * 100
		}

		segmentLength := min(int(math.Round(percentage/100.0*float64(totalLineWidth))), remainingWidth)

		categoryColor, ok := categoryColorMap[cat]
		if !ok {
			categoryColor = colorReset
		}

		coloredLine.WriteString(categoryColor)
		coloredLine.WriteString(strings.Repeat(lineCharacter, segmentLength))
		coloredLine.WriteString(colorReset)
		remainingWidth -= segmentLength

		if i == len(categories)-1 && remainingWidth > 0 {
			coloredLine.WriteString(categoryColor)
			coloredLine.WriteString(strings.Repeat(lineCharacter, remainingWidth))
			coloredLine.WriteString(colorReset)
		}
	}
	return coloredLine.String()
}

func printSummaryTotals(totalAmount float64, categories []string, categoryTotalsMap map[string]float64, categoryColorMap map[string]string) {
	fmt.Printf("\nTotal Amount: %.2f\n", totalAmount)

	if len(categoryTotalsMap) > 0 {
		fmt.Println("Category Totals:")

		for _, cat := range categories {
			categoryTotal := categoryTotalsMap[cat]
			percentage := 0.0
			if totalAmount > 0 {
				percentage = (categoryTotal / totalAmount) * 100
			}

			categoryColor, ok := categoryColorMap[cat]
			if !ok {
				categoryColor = colorReset
			}
			coloredCatName := fmt.Sprintf("%s%s%s", categoryColor, cat, colorReset)

			fmt.Printf("  - %s: %.2f (%.1f%%)\n", coloredCatName, categoryTotal, percentage)
		}
	}
}
