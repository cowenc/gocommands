package subcmd

import (
	"fmt"

	"github.com/cyverse/gocommands/commons"
	"github.com/spf13/cobra"
	"golang.org/x/xerrors"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize gocommands",
	Long: `This sets up iRODS Host and access account for other gocommands tools. 
	Once the configuration is set, configuration files are created under ~/.irods directory.
	The configuration is fully-compatible to that of icommands`,
	RunE: processInitCommand,
}

func AddInitCommand(rootCmd *cobra.Command) {
	// attach common flags
	commons.SetCommonFlags(initCmd)

	rootCmd.AddCommand(initCmd)
}

func processInitCommand(command *cobra.Command, args []string) error {
	cont, err := commons.ProcessCommonFlags(command)
	if err != nil {
		return xerrors.Errorf("failed to process common flags: %w", err)
	}

	if !cont {
		return nil
	}

	// handle local flags
	updated, err := commons.InputMissingFields()
	if err != nil {
		return xerrors.Errorf("failed to input missing fields: %w", err)
	}

	account, err := commons.GetEnvironmentManager().ToIRODSAccount()
	if err != nil {
		return xerrors.Errorf("failed to get iRODS account info from iCommands Environment: %w", err)
	}

	err = commons.TestConnect(account)
	if err != nil {
		return xerrors.Errorf("failed to connect to iRODS server: %w", err)
	}

	if updated {
		// save
		err := commons.GetEnvironmentManager().SaveEnvironment()
		if err != nil {
			return xerrors.Errorf("failed to save iCommands Environment: %w", err)
		}
	} else {
		fmt.Println("gocommands is already configured for following account:")
		err := commons.PrintAccount()
		if err != nil {
			return xerrors.Errorf("failed to print account info: %w", err)
		}
	}
	return nil
}
