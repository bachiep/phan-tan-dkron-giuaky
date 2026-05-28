package cmd

import (
	dkplugin "github.com/distribworks/dkron/v4/plugin"
	"github.com/distribworks/dkron/v4/plugin/webhook"
	"github.com/hashicorp/go-plugin"
	"github.com/spf13/cobra"
)

var webhookCmd = &cobra.Command{
	Hidden: true,
	Use:    "webhook",
	Short:  "Webhook processor plugin for dkron",
	Long:   ``,
	Run: func(cmd *cobra.Command, args []string) {
		plugin.Serve(&plugin.ServeConfig{
			HandshakeConfig: dkplugin.Handshake,
			Plugins: map[string]plugin.Plugin{
				"processor": &dkplugin.ProcessorPlugin{Processor: &webhook.Webhook{}},
			},
			GRPCServer: plugin.DefaultGRPCServer,
		})
	},
}

func init() {
	dkronCmd.AddCommand(webhookCmd)
}
