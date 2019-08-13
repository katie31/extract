package mysql

import (
	"github.com/wal-g/wal-g/internal/databases/mysql"
	"github.com/wal-g/wal-g/internal"
	"github.com/wal-g/wal-g/internal/tracelog"

	"github.com/spf13/cobra"
)

const StreamFetchShortDescription = ""

// streamFetchCmd represents the streamFetch command
var streamFetchCmd = &cobra.Command{
	Use:   "stream-fetch backup-name",
	Short: StreamFetchShortDescription,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		folder, err := internal.ConfigureFolder()
		if err != nil {
			tracelog.ErrorLogger.FatalError(err)
		}
		internal.HandleStreamFetch(args[0], folder, mysql.FetchLogs)
	},
}

func init() {
	MySQLCmd.AddCommand(streamFetchCmd)
}