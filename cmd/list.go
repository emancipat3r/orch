package cmd

import (
	"fmt"

	"github.com/emancipat3r/orch/logger"
	"github.com/emancipat3r/orch/providers"
	"github.com/emancipat3r/orch/ui"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List running VPS instances",
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()

		providerName := ui.ChoiceProvider()
		if providerName == "" {
			logger.Info("Operation cancelled by user.")
			return
		}

		prov, err := providers.GetProvider(providerName, configFile)
		if err != nil {
			logger.Error(err.Error())
			return
		}

		balance, err := prov.Balance(ctx)
		if err != nil {
			logger.Error("Failed to get " + providerName + " account balance: " + err.Error())
			return
		}
		logger.Info(providerName + " account balance: " + logger.Highlight("$"+balance))

		instances, err := prov.List(ctx)
		if err != nil {
			logger.Error("Failed to list " + providerName + " instances: " + err.Error())
			return
		}

		rows := make([][]string, 0, len(instances))
		for _, inst := range instances {
			rows = append(rows, []string{
				inst.ID, inst.IP, inst.Region, inst.Image, inst.Type, inst.Created, inst.Status,
			})
		}
		fmt.Println(ui.InstanceTable(rows))
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}
