package subcmd

import (
	"fmt"
	"path"
	"sort"
	"strconv"

	irodsclient_fs "github.com/cyverse/go-irodsclient/fs"
	irodsclient_irodsfs "github.com/cyverse/go-irodsclient/irods/fs"
	irodsclient_types "github.com/cyverse/go-irodsclient/irods/types"
	"github.com/cyverse/gocommands/commons"
	"github.com/spf13/cobra"
	"golang.org/x/xerrors"
)

var lsCmd = &cobra.Command{
	Use:   "ls [collection1] [collection2] ...",
	Short: "List entries in iRODS collections",
	Long:  `This lists data objects and collections in iRODS collections.`,
	RunE:  processLsCommand,
}

func AddLsCommand(rootCmd *cobra.Command) {
	// attach common flags
	commons.SetCommonFlags(lsCmd)

	lsCmd.Flags().BoolP("long", "l", false, "List data objects in a long format")
	lsCmd.Flags().BoolP("verylong", "L", false, "List data objects in a very long format")

	rootCmd.AddCommand(lsCmd)
}

func processLsCommand(command *cobra.Command, args []string) error {
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

	longFormat := false
	longFlag := command.Flags().Lookup("long")
	if longFlag != nil {
		longFormat, err = strconv.ParseBool(longFlag.Value.String())
		if err != nil {
			longFormat = false
		}
	}

	veryLongFormat := false
	veryLongFlag := command.Flags().Lookup("verylong")
	if veryLongFlag != nil {
		veryLongFormat, err = strconv.ParseBool(veryLongFlag.Value.String())
		if err != nil {
			veryLongFormat = false
		}
	}

	// Create a file system
	account := commons.GetAccount()
	filesystem, err := commons.GetIRODSFSClient(account)
	if err != nil {
		return xerrors.Errorf("failed to get iRODS FS Client: %w", err)
	}

	defer filesystem.Release()

	sourcePaths := args[:]

	if len(args) == 0 {
		sourcePaths = []string{"."}
	}

	for _, sourcePath := range sourcePaths {
		err = listOne(filesystem, sourcePath, longFormat, veryLongFormat)
		if err != nil {
			return xerrors.Errorf("failed to perform ls %s: %w", sourcePath, err)
		}
	}

	return nil
}

func listOne(fs *irodsclient_fs.FileSystem, sourcePath string, longFormat bool, veryLongFormat bool) error {
	cwd := commons.GetCWD()
	home := commons.GetHomeDir()
	zone := commons.GetZone()
	sourcePath = commons.MakeIRODSPath(cwd, home, zone, sourcePath)

	connection, err := fs.GetMetadataConnection()
	if err != nil {
		return xerrors.Errorf("failed to get connection: %w", err)
	}
	defer fs.ReturnMetadataConnection(connection)

	collection, err := irodsclient_irodsfs.GetCollection(connection, sourcePath)
	if err != nil {
		if !irodsclient_types.IsFileNotFoundError(err) {
			return xerrors.Errorf("failed to get collection %s: %w", sourcePath, err)
		}
	}

	if err == nil {
		colls, err := irodsclient_irodsfs.ListSubCollections(connection, sourcePath)
		if err != nil {
			return xerrors.Errorf("failed to list sub-collections in %s: %w", sourcePath, err)
		}

		objs, err := irodsclient_irodsfs.ListDataObjects(connection, collection)
		if err != nil {
			return xerrors.Errorf("failed to list data-objects in %s: %w", sourcePath, err)
		}

		printDataObjects(objs, veryLongFormat, longFormat)
		printCollections(colls)
		return nil
	}

	// data object
	parentSourcePath := path.Dir(sourcePath)

	parentCollection, err := irodsclient_irodsfs.GetCollection(connection, parentSourcePath)
	if err != nil {
		return xerrors.Errorf("failed to get collection %s: %w", parentSourcePath, err)
	}

	entry, err := irodsclient_irodsfs.GetDataObject(connection, parentCollection, path.Base(sourcePath))
	if err != nil {
		return xerrors.Errorf("failed to get data-object %s: %w", sourcePath, err)
	}

	printDataObject(entry, veryLongFormat, longFormat)
	return nil
}

func printDataObjects(entries []*irodsclient_types.IRODSDataObject, veryLongFormat bool, longFormat bool) {
	// sort by name
	sort.SliceStable(entries, func(i int, j int) bool {
		return entries[i].Name < entries[j].Name
	})

	for _, entry := range entries {
		printDataObject(entry, veryLongFormat, longFormat)
	}
}

func printDataObject(entry *irodsclient_types.IRODSDataObject, veryLongFormat bool, longFormat bool) {
	if veryLongFormat {
		for _, replica := range entry.Replicas {
			modTime := commons.MakeDateTimeString(replica.ModifyTime)
			fmt.Printf("  %s\t%d\t%s\t%d\t%s\t%s\t%s\n", replica.Owner, replica.Number, replica.ResourceHierarchy, entry.Size, modTime, getStatusMark(replica.Status), entry.Name)
			fmt.Printf("    %s\t%s\n", replica.CheckSum, replica.Path)
		}
	} else if longFormat {
		for _, replica := range entry.Replicas {
			modTime := commons.MakeDateTimeString(replica.ModifyTime)
			fmt.Printf("  %s\t%d\t%s\t%d\t%s\t%s\t%s\n", replica.Owner, replica.Number, replica.ResourceHierarchy, entry.Size, modTime, getStatusMark(replica.Status), entry.Name)
		}
	} else {
		fmt.Printf("  %s\n", entry.Name)
	}
}

func printCollections(entries []*irodsclient_types.IRODSCollection) {
	// sort by name
	sort.SliceStable(entries, func(i int, j int) bool {
		return entries[i].Name < entries[j].Name
	})

	for _, entry := range entries {
		fmt.Printf("  C- %s\n", entry.Path)
	}
}

func getStatusMark(status string) string {
	switch status {
	case "0":
		return "X" // stale
	case "1":
		return "&" // good
	default:
		return "?"
	}
}
