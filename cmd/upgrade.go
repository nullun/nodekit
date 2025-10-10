package cmd

import (
	"os"
	"time"

	"github.com/algorandfoundation/nodekit/api"
	"github.com/algorandfoundation/nodekit/cmd/utils/explanations"
	"github.com/algorandfoundation/nodekit/internal/algod"
	"github.com/algorandfoundation/nodekit/internal/system"
	"github.com/algorandfoundation/nodekit/ui/style"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"
)

// UpgradeMsg is a constant string used to indicate the start of the Algod upgrade process.
const UpgradeMsg = "Upgrading Algod"

var upgradeShort = "Upgrade the node daemon"

var upgradeLong = lipgloss.JoinVertical(
	lipgloss.Left,
	style.Purple(style.BANNER),
	"",
	style.Bold(upgradeShort),
	"",
	style.BoldUnderline("Overview:"),
	"Upgrade Algorand packages if it was installed with package manager.",
	"",
	style.Yellow.Render("This requires the daemon to be installed on your system."),
)

// upgradeCmd is a Cobra command used to upgrade Algod, utilizing the OS-specific package manager if applicable.
var upgradeCmd = &cobra.Command{
	Use:          "upgrade",
	Short:        upgradeShort,
	Long:         upgradeLong,
	SilenceUsage: true,
	Run: func(cmd *cobra.Command, args []string) {
		if NeedsUpgrade {
			log.Info(style.Green.Render("Upgrading NodeKit"))
			err := system.Upgrade(new(api.HttpPkg))
			if err != nil {
				log.Fatal(err)
			}
		}

		// TODO: get expected version and check if update is required
		log.Info(style.Green.Render(UpgradeMsg))
		// Warn user for prompt
		log.Warn(style.Yellow.Render(explanations.SudoWarningMsg))
		// TODO: Check Version from S3 against the local binary
		err := algod.Update()
		if err != nil {
			log.Error(err)
		}

		time.Sleep(5 * time.Second)

		// If it's not running, start the daemon (can happen)
		if !algod.IsRunning(algodData) {
			err = algod.Start()
			if err != nil {
				log.Error(err)
				os.Exit(1)
			}
		}
	},
}
