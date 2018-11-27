package walg

import (
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"github.com/wal-g/wal-g/tracelog"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const TarPartitionFolderName = "/tar_partitions/"

type NoBackupsFoundError struct {
	error
}

func NewNoBackupsFoundError() NoBackupsFoundError {
	return NoBackupsFoundError{errors.New("No backups found")}
}

func (err NoBackupsFoundError) Error() string {
	return fmt.Sprintf(tracelog.GetErrorFormatter(), err.error)
}

// Backup contains information about a valid backup
// generated and uploaded by WAL-G.
type Backup struct {
	BaseBackupFolder StorageFolder
	Name             string
}

func NewBackup(folder StorageFolder, name string) *Backup {
	return &Backup{folder, name}
}

func (backup *Backup) getStopSentinelPath() string {
	return backup.Name + SentinelSuffix
}

func (backup *Backup) getTarPartitionFolder() StorageFolder {
	return backup.BaseBackupFolder.GetSubFolder(backup.Name + TarPartitionFolderName)
}

// CheckExistence checks that the specified backup exists.
func (backup *Backup) CheckExistence() (bool, error) {
	return backup.BaseBackupFolder.Exists(backup.getStopSentinelPath())
}

// TODO : unit tests
func (backup *Backup) getTarNames() ([]string, error) {
	objects, _, err := backup.getTarPartitionFolder().ListFolder()
	if err != nil {
		return nil, errors.Wrapf(err, "unable to list backup '%s' for deletion", backup.Name)
	}
	result := make([]string, len(objects))
	for id, object := range objects {
		result[id] = strings.TrimPrefix(object.GetAbsPath(), backup.getTarPartitionFolder().GetPath())
	}
	return result, nil
}

// TODO : unit tests
func (backup *Backup) fetchSentinel() (BackupSentinelDto, error) {
	sentinelDto := BackupSentinelDto{}
	backupReaderMaker := NewStorageReaderMaker(backup.BaseBackupFolder, backup.getStopSentinelPath())
	backupReader, err := backupReaderMaker.Reader()
	if err != nil {
		return sentinelDto, err
	}
	sentinelDtoData, err := ioutil.ReadAll(backupReader)
	if err != nil {
		return sentinelDto, errors.Wrap(err, "failed to fetch sentinel")
	}

	err = json.Unmarshal(sentinelDtoData, &sentinelDto)
	return sentinelDto, errors.Wrap(err, "failed to unmarshal sentinel")
}

func checkDbDirectoryForUnwrap(dbDataDirectory string, sentinelDto BackupSentinelDto) error {
	if !sentinelDto.isIncremental() {
		isEmpty, err := IsDirectoryEmpty(dbDataDirectory)
		if err != nil {
			return err
		}
		if !isEmpty {
			return NewNonEmptyDbDataDirectoryError(dbDataDirectory)
		}
	} else {
		tracelog.DebugLogger.Println("DB data directory before increment:")
		filepath.Walk(dbDataDirectory,
			func(path string, info os.FileInfo, err error) error {
				if !info.IsDir() {
					tracelog.DebugLogger.Println(path)
				}
				return nil
			})

		for fileName, fileDescription := range sentinelDto.Files {
			if fileDescription.IsSkipped {
				tracelog.DebugLogger.Printf("Skipped file %v\n", fileName)
			}
		}
	}
	return nil
}

// TODO : unit tests
// Do the job of unpacking Backup object
func (backup *Backup) unwrap(dbDataDirectory string, sentinelDto BackupSentinelDto, filesToUnwrap map[string]bool) error {
	err := checkDbDirectoryForUnwrap(dbDataDirectory, sentinelDto)
	if err != nil {
		return err
	}

	tarInterpreter := NewFileTarInterpreter(dbDataDirectory, sentinelDto, filesToUnwrap)
	tarsToExtract, pgControlKey, err := backup.getTarsToExtract()
	if err != nil {
		return err
	}
	err = ExtractAll(tarInterpreter, tarsToExtract)
	if err != nil {
		return err
	}
	// Check name for backwards compatibility. Will check for `pg_control` if WALG version of backup.
	re := regexp.MustCompile(`^([^_]+._{1}[^_]+._{1})`)
	match := re.FindString(backup.Name)
	if match == "" || sentinelDto.isIncremental() {
		err = ExtractAll(tarInterpreter, []ReaderMaker{NewStorageReaderMaker(backup.getTarPartitionFolder(), pgControlKey)})
		if err != nil {
			return errors.Wrap(err, "failed to extract pg_control")
		}
	}

	tracelog.InfoLogger.Print("\nBackup extraction complete.\n")
	return nil
}

// TODO : unit tests
func IsDirectoryEmpty(directoryPath string) (bool, error) {
	var isEmpty = true
	searchLambda := func(path string, info os.FileInfo, err error) error {
		if path != directoryPath {
			isEmpty = false
			tracelog.InfoLogger.Printf("found file '%s' in directory: '%s'\n", path, directoryPath)
		}
		return nil
	}
	err := filepath.Walk(directoryPath, searchLambda)
	return isEmpty, errors.Wrapf(err, "can't check, that directory: '%s' is empty", directoryPath)
}

// TODO : init tests
func (backup *Backup) getTarsToExtract() (tarsToExtract []ReaderMaker, pgControlKey string, err error) {
	keys, err := backup.getTarNames()
	if err != nil {
		return nil, "", err
	}
	tracelog.DebugLogger.Printf("Tars to extract: '%+v'\n", keys)
	tarsToExtract = make([]ReaderMaker, 0, len(keys))

	pgControlRe := regexp.MustCompile(`^.*?pg_control\.tar(\..+$|$)`)
	for _, key := range keys {
		// Separate the pg_control key from the others to
		// extract it at the end, as to prevent server startup
		// with incomplete backup restoration.  But only if it
		// exists: it won't in the case of WAL-E backup
		// backwards compatibility.
		if pgControlRe.MatchString(key) {
			if pgControlKey != "" {
				panic("expect only one pg_control key match")
			}
			pgControlKey = key
			continue
		}
		tarToExtract := NewStorageReaderMaker(backup.getTarPartitionFolder(), key)
		tarsToExtract = append(tarsToExtract, tarToExtract)
	}
	if pgControlKey == "" {
		return nil, "", NewPgControlNotFoundError()
	}
	return
}
