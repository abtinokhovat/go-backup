package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "backup-agent",
	Short: "A backup agent for various databases with encryption support",
	Long: `A backup agent that supports backing up various databases (MySQL, PostgreSQL, InfluxDB)
with optional encryption and S3 upload capabilities.`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	// Add global flags here if needed
	rootCmd.PersistentFlags().StringP("config", "c", "config.yaml", "path to config file")
} 