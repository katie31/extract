package sqlserver

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/wal-g/tracelog"
	"github.com/wal-g/wal-g/internal"
	"github.com/wal-g/wal-g/internal/databases/sqlserver/blob"
	"github.com/wal-g/wal-g/utility"
	"net/url"
	"os"
	"syscall"
)

func HandleBackupRestore(backupName string, dbnames []string, noRecovery bool) {
	ctx, cancel := context.WithCancel(context.Background())
	signalHandler := utility.NewSignalHandler(ctx, cancel, []os.Signal{syscall.SIGINT, syscall.SIGTERM})
	defer func() { _ = signalHandler.Close() }()

	folder, err := internal.ConfigureFolder()
	tracelog.ErrorLogger.FatalOnError(err)

	backup, err := internal.GetBackupByName(backupName, utility.BaseBackupPath, folder)
	tracelog.ErrorLogger.FatalOnError(err)

	sentinel := new(SentinelDto)
	err = internal.FetchStreamSentinel(backup, &sentinel)
	tracelog.ErrorLogger.FatalOnError(err)

	db, err := getSQLServerConnection()
	tracelog.ErrorLogger.FatalfOnError("failed to connect to SQLServer: %v", err)

	dbnames, err = getDatabasesToRestore(sentinel, dbnames)
	tracelog.ErrorLogger.FatalfOnError("failed to list databases to restore: %v", err)

	bs, err := blob.NewServer(folder)
	tracelog.ErrorLogger.FatalfOnError("proxy create error: %v", err)

	err = bs.RunBackground(ctx, cancel)
	tracelog.ErrorLogger.FatalfOnError("proxy run error: %v", err)

	backupName = backup.Name
	baseUrl := getBackupUrl(backupName)

	err = runParallel(func(dbname string) error {
		return restoreSingleDatabase(ctx, db, baseUrl, dbname, noRecovery)
	}, dbnames)
	tracelog.ErrorLogger.FatalfOnError("overall restore failed: %v", err)

	tracelog.InfoLogger.Printf("restore finished")
}

func restoreSingleDatabase(ctx context.Context, db *sql.DB, baseUrl string, dbname string, noRecovery bool) error {
	backupUrl := fmt.Sprintf("%s/%s", baseUrl, url.QueryEscape(dbname))
	sql := fmt.Sprintf("RESTORE DATABASE %s FROM URL = '%s' WITH REPLACE", quoteName(dbname), backupUrl)
	if noRecovery {
		sql += ", NORECOVERY"
	}
	tracelog.InfoLogger.Printf("staring restore database [%s] from %s", dbname, backupUrl)
	tracelog.DebugLogger.Printf("SQL: %s", sql)
	_, err := db.ExecContext(ctx, sql)
	if err != nil {
		tracelog.ErrorLogger.Printf("database [%s] restore failed: %v", dbname, err)
	} else {
		tracelog.InfoLogger.Printf("database [%s] restore succefully finished", dbname)
	}
	return err
}
