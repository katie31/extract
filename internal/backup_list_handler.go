package internal

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"text/tabwriter"
	"time"

	"github.com/jedib0t/go-pretty/table"
	"github.com/wal-g/storages/storage"
	"github.com/wal-g/tracelog"
	"github.com/wal-g/wal-g/utility"
)

type InfoLogger interface {
	Println(v ...interface{})
}

type ErrorLogger interface {
	FatalOnError(err error)
}

type Logging struct {
	InfoLogger  InfoLogger
	ErrorLogger ErrorLogger
}

func DefaultHandleBackupList(folder storage.Folder) {
	DefaultHandleBackupListWithTarget(folder, utility.BaseBackupPath)
}

func DefaultHandleBackupListWithTarget(folder storage.Folder, targetPath string) {
	getBackupsFunc := func() ([]BackupTime, error) {
		return GetBackupsWithTarget(folder, targetPath)
	}
	writeBackupListFunc := func(backups []BackupTime) {
		WriteBackupList(backups, os.Stdout)
	}
	logging := Logging{
		InfoLogger:  tracelog.InfoLogger,
		ErrorLogger: tracelog.ErrorLogger,
	}

	HandleBackupList(getBackupsFunc, writeBackupListFunc, logging)
}

func HandleBackupList(
	getBackupsFunc func() ([]BackupTime, error),
	writeBackupListFunc func([]BackupTime),
	logging Logging,
) {
	backups, err := getBackupsFunc()
	if len(backups) == 0 {
		logging.InfoLogger.Println("No backups found")
		return
	}
	logging.ErrorLogger.FatalOnError(err)

	writeBackupListFunc(backups)
}

func HandleBackupListWithFlags(folder storage.Folder, pretty bool, json bool, detail bool) {
	HandleBackupListWithFlagsAndTarget(folder, pretty, json, detail, utility.BaseBackupPath)
}

// TODO : unit tests
func HandleBackupListWithFlagsAndTarget(folder storage.Folder, pretty bool, json bool, detail bool, targetPath string) {
	backups, err := GetBackupsWithTarget(folder, targetPath)
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
			err = WriteAsJson(backupDetails, os.Stdout, pretty)
			tracelog.ErrorLogger.FatalOnError(err)
		} else if pretty {
			writePrettyBackupListDetails(backupDetails, os.Stdout)
		} else {
			writeBackupListDetails(backupDetails, os.Stdout)
		}
	} else {
		if json {
			err = WriteAsJson(backups, os.Stdout, pretty)
			tracelog.ErrorLogger.FatalOnError(err)
		} else if pretty {
			WritePrettyBackupList(backups, os.Stdout)
		} else {
			WriteBackupList(backups, os.Stdout)
		}
	}
}

func GetBackupsDetails(folder storage.Folder, backups []BackupTime) ([]BackupDetail, error) {
	return GetBackupsDetailsWithTarget(folder, backups, utility.BaseBackupPath)
}

func GetBackupsDetailsWithTarget(folder storage.Folder, backups []BackupTime, targetPath string) ([]BackupDetail, error) {
	backupsDetails := make([]BackupDetail, 0, len(backups))
	for i := len(backups) - 1; i >= 0; i-- {
		details, err := GetBackupDetailsWithTarget(folder, backups[i], targetPath)
		if err != nil {
			return nil, err
		}
		backupsDetails = append(backupsDetails, details)
	}
	return backupsDetails, nil
}

func GetBackupDetails(folder storage.Folder, backupTime BackupTime) (BackupDetail, error) {
	return GetBackupDetailsWithTarget(folder, backupTime, utility.BaseBackupPath)
}

func GetBackupMetaData(folder storage.Folder, backupName string, targetPath string) (ExtendedMetadataDto, error) {
	backup, err := GetBackupByName(backupName, targetPath, folder)
	if err != nil {
		return ExtendedMetadataDto{}, err
	}

	metaData, err := backup.FetchMeta()
	if err != nil {
		return ExtendedMetadataDto{}, err
	}
	return metaData, nil
}

func GetBackupDetailsWithTarget(folder storage.Folder, backupTime BackupTime, targetPath string) (BackupDetail, error) {
	metaData, err := GetBackupMetaData(folder, backupTime.BackupName, targetPath)
	if err != nil {
		return BackupDetail{}, err
	}
	return BackupDetail{backupTime, metaData}, nil
}

func FormatTimeInner(backupTime time.Time, timeFormat string) string {
	if backupTime.IsZero() {
		return "-"
	}
	return backupTime.Format(timeFormat)
}

func FormatTime(backupTime time.Time) string {
	return FormatTimeInner(backupTime, time.RFC3339)
}

func PrettyFormatTime(backupTime time.Time) string {
	return FormatTimeInner(backupTime, time.RFC850)
}

// TODO : unit tests
func WriteBackupList(backups []BackupTime, output io.Writer) {
	writer := tabwriter.NewWriter(output, 0, 0, 1, ' ', 0)
	defer writer.Flush()
	fmt.Fprintln(writer, "name\tcreated\tmodified\twal_segment_backup_start")
	for i := len(backups) - 1; i >= 0; i-- {
		b := backups[i]
		fmt.Fprintln(writer, fmt.Sprintf("%v\t%v\t%v\t%v", b.BackupName, FormatTime(b.CreationTime), FormatTime(b.ModificationTime), b.WalFileName))
	}
}

// TODO : unit tests
func writeBackupListDetails(backupDetails []BackupDetail, output io.Writer) {
	writer := tabwriter.NewWriter(output, 0, 0, 1, ' ', 0)
	defer writer.Flush()
	fmt.Fprintln(writer, "name\tcreated\tmodified\twal_segment_backup_start\tstart_time\tfinish_time\thostname\tdata_dir\tpg_version\tstart_lsn\tfinish_lsn\tis_permanent")
	for i := len(backupDetails) - 1; i >= 0; i-- {
		b := backupDetails[i]
		fmt.Fprintln(writer, fmt.Sprintf("%v\t%v\t%v\t%v\t%v\t%v\t%v\t%v\t%v\t%v\t%v\t%v", b.BackupName, FormatTime(b.CreationTime), FormatTime(b.ModificationTime), b.WalFileName, FormatTime(b.StartTime), FormatTime(b.FinishTime), b.Hostname, b.DataDir, b.PgVersion, b.StartLsn, b.FinishLsn, b.IsPermanent))
	}
}

func WritePrettyBackupList(backups []BackupTime, output io.Writer) {
	writer := table.NewWriter()
	writer.SetOutputMirror(output)
	defer writer.Render()
	writer.AppendHeader(table.Row{"#", "Name", "Created", "Modified", "WAL segment backup start"})
	for i, b := range backups {
		writer.AppendRow(table.Row{i, b.BackupName, PrettyFormatTime(b.CreationTime), PrettyFormatTime(b.ModificationTime), b.WalFileName})
	}
}

// TODO : unit tests
func writePrettyBackupListDetails(backupDetails []BackupDetail, output io.Writer) {
	writer := table.NewWriter()
	writer.SetOutputMirror(output)
	defer writer.Render()
	writer.AppendHeader(table.Row{"#", "Name", "Created", "Modified", "WAL segment backup start", "Start time", "Finish time", "Hostname", "Datadir", "PG Version", "Start LSN", "Finish LSN", "Permanent"})
	for idx := range backupDetails {
		b := &backupDetails[idx]
		writer.AppendRow(table.Row{idx, b.BackupName, PrettyFormatTime(b.CreationTime), PrettyFormatTime(b.ModificationTime), b.WalFileName, PrettyFormatTime(b.StartTime), PrettyFormatTime(b.FinishTime), b.Hostname, b.DataDir, b.PgVersion, b.StartLsn, b.FinishLsn, b.IsPermanent})
	}
}

func WriteAsJson(data interface{}, output io.Writer, pretty bool) error {
	var bytes []byte
	var err error
	if pretty {
		bytes, err = json.MarshalIndent(data, "", "    ")
	} else {
		bytes, err = json.Marshal(data)
	}
	if err != nil {
		return err
	}
	_, err = output.Write(bytes)
	return err
}
