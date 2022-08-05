package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/KKKKjl/tinykit/internal/server"
)

var (
	rootCmd = &cobra.Command{
		Use: "tinykit",
		Long: `
	_______  _                _  __ _  _   
	|__   __|(_)              | |/ /(_)| |  
	   | |    _  _ __   _   _ | ' /  _ | |_ 
	   | |   | || '_ \ | | | ||  <  | || __|
	   | |   | || | | || |_| || . \ | || |_ 
	   |_|   |_||_| |_| \__, ||_|\_\|_| \__|
						 __/ |              
						|___/               	   
	`,
	}

	startCmd = &cobra.Command{
		Use:   "start",
		Short: "start the gateway server",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Print(args)
			server.Start()
		},
	}
)

func init() {
	rootCmd.AddCommand(startCmd)
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
