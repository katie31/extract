package rocksdb

import (
	"github.com/spf13/cobra"
	"github.com/wal-g/tracelog"
	"github.com/wal-g/wal-g/internal"
	"github.com/wal-g/wal-g/tmp/rocksdb"
)

const backupFetchShortDescription = "Restores backup to database"

var backupFetchCmd = &cobra.Command{
	Use:   "backup-fetch database_path backup_name",
	Short: backupFetchShortDescription, // TODO : improve description
	Args:  cobra.ExactValidArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		dbPath := args[0]
		backup_name := args[1]

		if walDirectory == "" {
			walDirectory = dbPath
		}

		uploader, err := internal.ConfigureUploader()
		tracelog.ErrorLogger.FatalOnError(err)

		err = rocksdb.HandleBackupFetch(uploader, rocksdb.NewDatabaseOptions(dbPath, walDirectory), rocksdb.NewRestoreOptions(backup_name, true))
		tracelog.ErrorLogger.FatalOnError(err)
	},
}

func init() {
	backupFetchCmd.Flags().StringVar(&walDirectory, walDirectoryFlag, "", walDirectoryDescription)
	cmd.AddCommand(backupFetchCmd)
}
