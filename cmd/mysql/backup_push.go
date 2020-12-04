package mysql

import (
	"github.com/spf13/cobra"
	"github.com/wal-g/tracelog"
	"github.com/wal-g/wal-g/internal"
	"github.com/wal-g/wal-g/internal/databases/mysql"
)

const backupPushShortDescription = "Creates new backup and pushes it to storage"

// backupPushCmd represents the streamPush command
var backupPushCmd = &cobra.Command{
	Use:   "backup-push",
	Short: backupPushShortDescription,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		internal.RequiredSettings[internal.NameStreamCreateCmd] = true
		internal.RequiredSettings[internal.MysqlDatasourceNameSetting] = true
		err := internal.AssertRequiredSettingsSet()
		tracelog.ErrorLogger.FatalOnError(err)
	},
	Run: func(cmd *cobra.Command, args []string) {
		uploader, err := internal.ConfigureUploader()
		tracelog.ErrorLogger.FatalOnError(err)
		backupCmd, err := internal.GetCommandSetting(internal.NameStreamCreateCmd)
		tracelog.ErrorLogger.FatalOnError(err)
		mysql.HandleBackupPush(uploader, backupCmd)
	},
}

func init() {
	cmd.AddCommand(backupPushCmd)
}
