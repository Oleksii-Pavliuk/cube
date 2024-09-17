/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"cube/worker"
	"log"

	"github.com/spf13/cobra"
)

// workerCmd represents the worker command
var workerCmd = &cobra.Command{
	Use:   "worker",
	Short: "Worker command to operate a Cube worker node.",
	Long: `cube worker command.
	The worker runs tasks and responds to the manager's requests about task state.`,
	Run: func(cmd *cobra.Command, args []string) {
    host, _ := cmd.Flags().GetString("host")
    port, _ := cmd.Flags().GetInt("port")
    name, _ := cmd.Flags().GetString("name")
    dbType, _ := cmd.Flags().GetString("dbtype")

		log.Println("Starting worker.")
    w := worker.New(name, dbType)
    api := worker.Api{Address: host, Port: port, Worker: w}
    go w.RunTasks()
    go w.CollectStats()
    go w.UpdateTasks()
    log.Printf("Starting worker API on http://%s:%d", host, port)
    api.Start()
	},
}

func init() {
	rootCmd.AddCommand(workerCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// workerCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// workerCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
