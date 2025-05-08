package main

import (
	"log"

	"github.com/spf13/cobra"
)

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
	},
}

func init() {
	addCmd.Flags().StringP("title", "t", "", "Title of the expense (required)")
	addCmd.Flags().Float64P("amount", "a", 0.0, "Amount of the expense (required)")
	addCmd.Flags().IntP("day", "d", 0, "Day of the month (1-28) for the expense (required)")
	addCmd.Flags().StringP("category", "c", "", "Category of the expense (optional)")

	addCmd.MarkFlagRequired("title")
	addCmd.MarkFlagRequired("amount")
	addCmd.MarkFlagRequired("day")
}
