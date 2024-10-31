/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"os"

	conf "github.com/rocky2015aaa/tokenswap-client/cmd/config"
	"github.com/rocky2015aaa/tokenswap-client/cmd/order"
	"github.com/rocky2015aaa/tokenswap-client/config"
	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var (
	rootCmd = &cobra.Command{
		Use:   "tokenswap-client",
		Short: "tokenswap-client application",
		Long:  `tokenswap-client application that trades XELIS token`,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			// Execute logging middleware before every command
			init, _ := cmd.Flags().GetBool("init") // TODO: flag error handling
			if !init {
				err := config.ManageUserTokens()
				if err != nil {
					fmt.Println(err.Error())
					os.Exit(1)
				}
			}
		},
		Run: func(cmd *cobra.Command, args []string) {
			init, _ := cmd.Flags().GetBool("init")
			if init {
				configFile, _ := os.Stat(config.AbsoluteConfigFilePath)
				if configFile != nil {
					fmt.Println("The application initialization has done already")
					return
				}
				err := config.CreateConfig()
				if err != nil {
					if err.Error() == config.UserExistanceMessage {
						fmt.Println("The user with the email already exists")
						return
					}
					fmt.Println("Error while creating a config file")
					return
				}
				fmt.Println("A config file created successfully.")
			} else {
				cmd.Help()
			}
		},
	}
)

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		fmt.Println(err)
		return
	}
}

func init() {
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	// rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.tokenswap-client.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.Root().CompletionOptions.DisableDefaultCmd = true
	rootCmd.SetHelpCommand(&cobra.Command{
		Hidden: true,
	})
	rootCmd.Flags().Bool("init", false, "Initialize the client application configuration")
	rootCmd.AddCommand(conf.ConfigCmd)
	rootCmd.AddCommand(order.OrderCmd)
}
