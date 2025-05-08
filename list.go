package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
)

const (
	colorGreen      = "\033[32m"
	colorYellow     = "\033[93m"
	effectBlink     = "\033[5m"
	colorReset      = "\033[0m"
	statusIndicator = "●"
)

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

		for rows.Next() {
			var exp Expense
			var category sql.NullString

			err := rows.Scan(&exp.ID, &exp.Title, &exp.Amount, &exp.Day, &category)
			if err != nil {
				log.Printf("Error scanning row: %v", err)
				continue
			}

			if category.Valid && category.String != "" {
				exp.Category = category.String
				categoryTotalsMap[exp.Category] += exp.Amount
			} else {
				exp.Category = ""
				categoryTotalsMap["Uncategorized"] += exp.Amount
			}

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

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "Title\tAmount\tDate\tCategory\t")
		fmt.Fprintln(w, "-----\t------\t----\t--------\t")

		now := time.Now()
		currentDay := now.Day()
		currentMonthName := now.Format("January")

		for _, exp := range expenses {
			statusOutput := statusIndicator
			expenseDay := exp.Day

			if currentDay > expenseDay {
				statusOutput = colorGreen + statusIndicator + colorReset
			} else if currentDay == expenseDay {
				// Use Bright Yellow + Blink
				statusOutput = effectBlink + colorYellow + statusIndicator + colorReset
			}

			displayDateStr := fmt.Sprintf("%02d %s", expenseDay, currentMonthName)
			amountStr := fmt.Sprintf("%.2f", exp.Amount)
			displayCategory := exp.Category
			if displayCategory == "" {
				displayCategory = "Uncategorized"
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", exp.Title, amountStr, displayDateStr, displayCategory, statusOutput)
		}

		w.Flush()

		fmt.Printf("\nTotal Amount: %.2f\n", totalAmount)
		if len(categoryTotalsMap) > 0 {
			fmt.Println("Category Totals:")
			var categories []string
			for cat := range categoryTotalsMap {
				categories = append(categories, cat)
			}
			sort.Slice(categories, func(i, j int) bool {
				if categories[i] == "Uncategorized" {
					return false
				}
				if categories[j] == "Uncategorized" {
					return true
				}
				return strings.ToLower(categories[i]) < strings.ToLower(categories[j])
			})

			for _, cat := range categories {
				categoryTotal := categoryTotalsMap[cat]
				percentage := 0.0
				if totalAmount > 0 {
					percentage = (categoryTotal / totalAmount) * 100
				}
				fmt.Printf("  - %s: %.2f (%.1f%%)\n", cat, categoryTotal, percentage)
			}
		}
	},
}
