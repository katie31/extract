package postgres

import (
	"fmt"
	"io"
	"os"
	"text/tabwriter"
	"time"

	"github.com/jedib0t/go-pretty/table"
	"github.com/wal-g/storages/storage"
	"github.com/wal-g/tracelog"
	"github.com/wal-g/wal-g/internal"
)

// TODO : unit tests
func HandleBackupListWithFlags(folder storage.Folder, pretty bool, json bool, detail bool) {
	backups, err := internal.GetBackups(folder)
	if len(backups) == 0 {
		tracelog.InfoLogger.Println("No backups found")
		return
	}
	tracelog.ErrorLogger.FatalOnError(err)
	// if details are requested we append content of metadata.json to each line
	if detail {
		backupDetails, err := GetBackupsDetails(folder, backups)
		tracelog.ErrorLogger.FatalOnError(err)
		if json {
			err = internal.WriteAsJSON(backupDetails, os.Stdout, pretty)
			tracelog.ErrorLogger.FatalOnError(err)
		} else if pretty {
			writePrettyBackupListDetails(backupDetails, os.Stdout)
		} else {
			err = writeBackupListDetails(backupDetails, os.Stdout)
			tracelog.ErrorLogger.FatalOnError(err)
		}
	} else {
		if json {
			err = internal.WriteAsJSON(backups, os.Stdout, pretty)
			tracelog.ErrorLogger.FatalOnError(err)
		} else if pretty {
			internal.WritePrettyBackupList(backups, os.Stdout)
		} else {
			internal.WriteBackupList(backups, os.Stdout)
		}
	}
}

func GetBackupsDetails(folder storage.Folder, backups []internal.BackupTime) ([]BackupDetail, error) {
	backupsDetails := make([]BackupDetail, 0, len(backups))
	for i := len(backups) - 1; i >= 0; i-- {
		details, err := GetBackupDetails(folder, backups[i])
		if err != nil {
			return nil, err
		}
		backupsDetails = append(backupsDetails, details)
	}
	return backupsDetails, nil
}

func GetBackupDetails(folder storage.Folder, backupTime internal.BackupTime) (BackupDetail, error) {
	backup := NewBackup(folder, backupTime.BackupName)

	metaData, err := backup.FetchMeta()
	if err != nil {
		return BackupDetail{}, err
	}
	return BackupDetail{backupTime, metaData}, nil
}

// TODO : unit tests
func writeBackupListDetails(backupDetails []BackupDetail, output io.Writer) error {
	writer := tabwriter.NewWriter(output, 0, 0, 1, ' ', 0)
	defer writer.Flush()
	_, err := fmt.Fprintln(writer, "name\tlast_modified\twal_segment_backup_start\tstart_time\tfinish_time\thostname\tdata_dir\tpg_version\tstart_lsn\tfinish_lsn\tis_permanent") //nolint:lll
	if err != nil {
		return err
	}
	for i := len(backupDetails) - 1; i >= 0; i-- {
		b := backupDetails[i]
		_, err = fmt.Fprintf(writer, "%v\t%v\t%v\t%v\t%v\t%v\t%v\t%v\t%v\t%v\t%v",
			b.BackupName, b.Time.Format(time.RFC3339), b.WalFileName, b.StartTime.Format(time.RFC850), b.FinishTime.Format(time.RFC850), b.Hostname, b.DataDir, b.PgVersion, b.StartLsn, b.FinishLsn, b.IsPermanent) //nolint:lll
		if err != nil {
			return err
		}
	}
	return nil
}

// TODO : unit tests
func writePrettyBackupListDetails(backupDetails []BackupDetail, output io.Writer) {
	writer := table.NewWriter()
	writer.SetOutputMirror(output)
	defer writer.Render()
	writer.AppendHeader(table.Row{"#", "Name", "Last modified", "WAL segment backup start", "Start time", "Finish time", "Hostname", "Datadir", "PG Version", "Start LSN", "Finish LSN", "Permanent"}) //nolint:lll
	for idx := range backupDetails {
		b := &backupDetails[idx]
		writer.AppendRow(table.Row{idx, b.BackupName, b.Time.Format(time.RFC850), b.WalFileName, b.StartTime.Format(time.RFC850), b.FinishTime.Format(time.RFC850), b.Hostname, b.DataDir, b.PgVersion, b.StartLsn, b.FinishLsn, b.IsPermanent}) //nolint:lll
	}
}
