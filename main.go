package main

import (
	"bufio"
	"database/sql"
	"fmt"
	"log"
	"math"
	"os"
	"os/user"
	"path/filepath"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/spf13/cobra"
)

var (
	db     *sql.DB
	dbPath string
)

const (
	colorGreen      = "\033[32m"
	colorYellow     = "\033[93m"
	effectBlink     = "\033[5m"
	colorReset      = "\033[0m"
	statusIndicator = "●"
	barChar         = "■"
	maxBarWidth     = 40
)

type Expense struct {
	ID       int
	Title    string
	Amount   float64
	Day      int
	Category string
}

type CategoryTotal struct {
	Name   string
	Amount float64
}

func initDB() {
	currentUser, err := user.Current()
	if err != nil {
		log.Fatalf("Error getting current user: %v", err)
	}
	configDir := filepath.Join(currentUser.HomeDir, ".config", "monke")
	dbPath = filepath.Join(configDir, "monke.db")
	err = os.MkdirAll(configDir, 0755)
	if err != nil {
		log.Fatalf("Error creating config directory: %v", err)
	}
	db, err = sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatalf("Error opening database: %v", err)
	}

	createTableSQL := `CREATE TABLE IF NOT EXISTS expenses (
		"id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
		"title" TEXT,
		"amount" REAL,
		"day" INTEGER,
		"category" TEXT
	);`

	_, err = db.Exec(createTableSQL)
	if err != nil {
		log.Fatalf("Error creating table: %v", err)
	}
}

var rootCmd = &cobra.Command{
	Use:   "monke",
	Short: "Monke is a simple expense tracker CLI",
	PersistentPreRun: func(_ *cobra.Command, _ []string) {
		initDB()
	},
	PersistentPostRun: func(_ *cobra.Command, _ []string) {
		if db != nil {
			db.Close()
		}
	},
}

var addCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a new expense",
	Run: func(cmd *cobra.Command, _ []string) {
		title, _ := cmd.Flags().GetString("title")
		amount, _ := cmd.Flags().GetFloat64("amount")
		day, _ := cmd.Flags().GetInt("day")
		category, _ := cmd.Flags().GetString("category")

		if title == "" {
			log.Fatal("Error: title flag is required.")
		}

		if day < 1 || day > 28 {
			log.Fatalf("Error: Invalid day '%d'. Please provide a day between 1 and 28.", day)
		}

		insertSQL := `INSERT INTO expenses(title, amount, day, category) VALUES (?, ?, ?, ?)`
		statement, err := db.Prepare(insertSQL)
		if err != nil {
			log.Fatalf("Error preparing insert statement: %v", err)
		}
		defer statement.Close()

		_, err = statement.Exec(title, amount, day, category)
		if err != nil {
			log.Fatalf("Error executing insert statement: %v", err)
		}

		fmt.Printf("Expense added successfully for day %d!\n", day)
	},
}

var lsCmd = &cobra.Command{
	Use:   "ls",
	Short: "List all expenses",
	Run: func(_ *cobra.Command, _ []string) {
		rows, err := db.Query("SELECT id, title, amount, day, category FROM expenses ORDER BY day ASC")
		if err != nil {
			if strings.Contains(err.Error(), "no such column: day") {
				log.Fatalf("Error: Database schema mismatch. Clear the database with './monke clear'.")
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

var chartCmd = &cobra.Command{
	Use:   "chart",
	Short: "Display category expenses as a simple chart",
	Run: func(cmd *cobra.Command, _ []string) {
		query := `
			SELECT
				COALESCE(category, 'Uncategorized') as category_name,
				SUM(amount) as total
			FROM expenses
			GROUP BY category_name
			HAVING total > 0
			ORDER BY total DESC;
		`
		rows, err := db.Query(query)
		if err != nil {
			log.Fatalf("Error querying category totals: %v", err)
		}
		defer rows.Close()

		var categoryTotals []CategoryTotal
		grandTotal := 0.0

		for rows.Next() {
			var ct CategoryTotal
			err := rows.Scan(&ct.Name, &ct.Amount)
			if err != nil {
				log.Printf("Error scanning category total row: %v", err)
				continue
			}
			categoryTotals = append(categoryTotals, ct)
			grandTotal += ct.Amount
		}
		if err = rows.Err(); err != nil {
			log.Fatalf("Error iterating category totals: %v", err)
		}

		if len(categoryTotals) == 0 {
			fmt.Println("No categorized expenses found to chart.")
			return
		}

		fmt.Println("Category Expense Chart:")
		fmt.Printf("Total: %.2f\n\n", grandTotal)

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "Category\tAmount\tPercent\tChart")
		fmt.Fprintln(w, "--------\t------\t-------\t-----")

		for _, ct := range categoryTotals {
			percentage := 0.0
			if grandTotal > 0 {
				percentage = (ct.Amount / grandTotal) * 100
			}
			barWidth := int(math.Round((percentage / 100) * float64(maxBarWidth)))
			bar := strings.Repeat(barChar, barWidth)

			fmt.Fprintf(w, "%s\t%.2f\t%.1f%%\t%s\n", ct.Name, ct.Amount, percentage, bar)
		}

		w.Flush()
	},
}

var clearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Delete all expenses from the database",
	Run: func(_ *cobra.Command, _ []string) {
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("Are you sure you want to delete ALL expenses? This cannot be undone. [y/N]: ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(strings.ToLower(input))
		if input == "y" || input == "yes" {
			deleteSQL := `DELETE FROM expenses;`
			resetSeqSQL := `DELETE FROM sqlite_sequence WHERE name='expenses';`
			_, err := db.Exec(deleteSQL)
			if err != nil {
				log.Fatalf("Error deleting expenses: %v", err)
			}

			_, err = db.Exec(resetSeqSQL)
			if err != nil {
				log.Printf("Warning: Could not reset sequence counter: %v", err)
			}

			fmt.Println("All expenses have been deleted.")
		} else {
			fmt.Println("Operation cancelled.")
		}
	},
}

func main() {
	addCmd.Flags().StringP("title", "t", "", "Title of the expense (required)")
	addCmd.Flags().Float64P("amount", "a", 0.0, "Amount of the expense (required)")
	addCmd.Flags().IntP("day", "d", 0, "Day of the month (1-28) for the expense (required)")
	addCmd.Flags().StringP("category", "c", "", "Category of the expense (optional)")

	addCmd.MarkFlagRequired("title")
	addCmd.MarkFlagRequired("amount")
	addCmd.MarkFlagRequired("day")

	rootCmd.AddCommand(addCmd)
	rootCmd.AddCommand(lsCmd)
	rootCmd.AddCommand(chartCmd)
	rootCmd.AddCommand(clearCmd)
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
