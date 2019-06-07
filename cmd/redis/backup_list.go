package redis

import (
	"github.com/spf13/cobra"
	"github.com/wal-g/wal-g/internal"
	"github.com/wal-g/wal-g/internal/tracelog"
)

const BackupListShortDescription = "Print available backups"

// backupListCmd represents the backupList command
var backupListCmd = &cobra.Command{
	Use:   "backup-list",
	Short: BackupListShortDescription,
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		folder, err := internal.ConfigureFolder()
		if err != nil {
			tracelog.ErrorLogger.FatalError(err)
		}
		internal.HandleBackupList(folder)
	},
}

func init() {
	RedisCmd.AddCommand(backupListCmd)
}
