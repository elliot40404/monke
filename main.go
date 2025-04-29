package main

import (
	"bufio"
	"database/sql"
	"fmt"
	"log"
	"os"
	"os/user"
	"path/filepath"
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
	colorReset      = "\033[0m"
	statusIndicator = "●"
)

type Expense struct {
	ID       int
	Title    string
	Amount   float64
	Day      int
	Category string
}

func initDB() {
	currentUser, err := user.Current()
	if err != nil {
		log.Fatalf("Error getting current user: %v", err)
	}
	configDir := filepath.Join(currentUser.HomeDir, ".config", "monke")
	dbPath = filepath.Join(configDir, "monke.db")
	err = os.MkdirAll(configDir, 0o755)
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
	Short: "return to monke",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		initDB()
	},
	PersistentPostRun: func(cmd *cobra.Command, args []string) {
		if db != nil {
			db.Close()
		}
	},
}

var addCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a new expense",
	Run: func(cmd *cobra.Command, args []string) {
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
	Run: func(cmd *cobra.Command, args []string) {
		rows, err := db.Query("SELECT id, title, amount, day, category FROM expenses ORDER BY day ASC")
		if err != nil {
			if strings.Contains(err.Error(), "no such column: day") {
				log.Fatalf("Error: Database schema mismatch. The 'day' column is missing or incorrect. If you have existing data from an older version, please migrate it using the SQL statements provided separately or clear the database with './monke clear'.")
			}
			log.Fatalf("Error querying expenses: %v", err)
		}
		defer rows.Close()

		var expenses []Expense
		totalAmount := 0.0
		categoryTotals := make(map[string]float64)

		for rows.Next() {
			var exp Expense
			var category sql.NullString

			err := rows.Scan(&exp.ID, &exp.Title, &exp.Amount, &exp.Day, &category)
			if err != nil {
				log.Printf("Error scanning row: %v", err)
				continue
			}

			if category.Valid {
				exp.Category = category.String
			} else {
				exp.Category = ""
			}

			expenses = append(expenses, exp)
			totalAmount += exp.Amount
			if exp.Category != "" {
				categoryTotals[exp.Category] += exp.Amount
			}
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
			}

			displayDateStr := fmt.Sprintf("%02d %s", expenseDay, currentMonthName)
			amountStr := fmt.Sprintf("%.2f", exp.Amount)
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", exp.Title, amountStr, displayDateStr, exp.Category, statusOutput)
		}

		w.Flush()

		fmt.Printf("\nTotal Amount: %.2f\n", totalAmount)
		if len(categoryTotals) > 0 {
			fmt.Println("Category Totals:")
			var categories []string
			for cat := range categoryTotals {
				categories = append(categories, cat)
			}
			for i := range categories {
				for j := i + 1; j < len(categories); j++ {
					if strings.ToLower(categories[i]) > strings.ToLower(categories[j]) {
						categories[i], categories[j] = categories[j], categories[i]
					}
				}
			}
			for _, cat := range categories {
				categoryTotal := categoryTotals[cat]
				percentage := 0.0
				if totalAmount > 0 {
					percentage = (categoryTotal / totalAmount) * 100
				}
				fmt.Printf("  - %s: %.2f (%.1f%%)\n", cat, categoryTotal, percentage)
			}
		}
	},
}

var clearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Delete all expenses from the database",
	Run: func(cmd *cobra.Command, args []string) {
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
	rootCmd.AddCommand(clearCmd)
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
