package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

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

func main() {
	rootCmd.AddCommand(addCmd)
	rootCmd.AddCommand(lsCmd)
	rootCmd.AddCommand(clearCmd)
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
