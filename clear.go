package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

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
