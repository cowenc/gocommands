package subcmd

import (
	"strconv"

	irodsclient_fs "github.com/cyverse/go-irodsclient/fs"
	"github.com/cyverse/gocommands/commons"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"golang.org/x/xerrors"
)

var rmCmd = &cobra.Command{
	Use:   "rm [data-object1] [data-object2] [collection1] ...",
	Short: "Remove iRODS data-objects or collections",
	Long:  `This removes iRODS data-objects or collections.`,
	RunE:  processRmCommand,
}

func AddRmCommand(rootCmd *cobra.Command) {
	// attach common flags
	commons.SetCommonFlags(rmCmd)
	rmCmd.Flags().BoolP("recurse", "r", false, "Remove non-empty collections")
	rmCmd.Flags().BoolP("force", "f", false, "Remove forcefully")

	rootCmd.AddCommand(rmCmd)
}

func processRmCommand(command *cobra.Command, args []string) error {
	cont, err := commons.ProcessCommonFlags(command)
	if err != nil {
		return xerrors.Errorf("failed to process common flags: %w", err)
	}

	if !cont {
		return nil
	}

	// handle local flags
	_, err = commons.InputMissingFields()
	if err != nil {
		return xerrors.Errorf("failed to input missing fields: %w", err)
	}

	recurse := false
	recurseFlag := command.Flags().Lookup("recurse")
	if recurseFlag != nil {
		recurse, err = strconv.ParseBool(recurseFlag.Value.String())
		if err != nil {
			recurse = false
		}
	}

	force := false
	forceFlag := command.Flags().Lookup("force")
	if forceFlag != nil {
		force, err = strconv.ParseBool(forceFlag.Value.String())
		if err != nil {
			force = false
		}
	}

	// Create a file system
	account := commons.GetAccount()
	filesystem, err := commons.GetIRODSFSClient(account)
	if err != nil {
		return xerrors.Errorf("failed to get iRODS FS Client: %w", err)
	}

	defer filesystem.Release()

	if len(args) == 0 {
		return xerrors.Errorf("not enough input arguments")
	}

	for _, sourcePath := range args {
		err = removeOne(filesystem, sourcePath, force, recurse)
		if err != nil {
			return xerrors.Errorf("failed to perform rm %s: %w", sourcePath, err)
		}
	}
	return nil
}

func removeOne(filesystem *irodsclient_fs.FileSystem, targetPath string, force bool, recurse bool) error {
	logger := log.WithFields(log.Fields{
		"package":  "main",
		"function": "removeOne",
	})

	cwd := commons.GetCWD()
	home := commons.GetHomeDir()
	zone := commons.GetZone()
	targetPath = commons.MakeIRODSPath(cwd, home, zone, targetPath)

	targetEntry, err := commons.StatIRODSPath(filesystem, targetPath)
	if err != nil {
		return xerrors.Errorf("failed to stat %s: %w", targetPath, err)
	}

	if targetEntry.Type == irodsclient_fs.FileEntry {
		// file
		logger.Debugf("removing a data object %s", targetPath)
		err = filesystem.RemoveFile(targetPath, force)
		if err != nil {
			return xerrors.Errorf("failed to remove %s: %w", targetPath, err)
		}
	} else {
		// dir
		if !recurse {
			return xerrors.Errorf("cannot remove a collection, recurse is not set")
		}

		logger.Debugf("removing a collection %s", targetPath)
		err = filesystem.RemoveDir(targetPath, recurse, force)
		if err != nil {
			return xerrors.Errorf("failed to remove dir %s: %w", targetPath, err)
		}
	}
	return nil
}
