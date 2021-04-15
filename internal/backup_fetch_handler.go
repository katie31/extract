package internal

import (
	"bytes"
		"fmt"
	"os/exec"

	"github.com/wal-g/wal-g/utility"

	"github.com/pkg/errors"
	"github.com/wal-g/storages/storage"
	"github.com/wal-g/tracelog"
)

type BackupNonExistenceError struct {
	error
}

func NewBackupNonExistenceError(backupName string) BackupNonExistenceError {
	return BackupNonExistenceError{errors.Errorf("Backup '%s' does not exist.", backupName)}
}

func (err BackupNonExistenceError) Error() string {
	return fmt.Sprintf(tracelog.GetErrorFormatter(), err.error)
}

func GetCommandStreamFetcher(cmd *exec.Cmd) func(folder storage.Folder, backup Backup) {
	return func(folder storage.Folder, backup Backup) {
		stdin, err := cmd.StdinPipe()
		tracelog.ErrorLogger.FatalfOnError("Failed to fetch backup: %v\n", err)
		stderr := &bytes.Buffer{}
		cmd.Stderr = stderr
		err = cmd.Start()
		tracelog.ErrorLogger.FatalfOnError("Failed to start restore command: %v\n", err)
		err = downloadAndDecompressStream(&backup, stdin)
		cmdErr := cmd.Wait()
		if cmdErr != nil {
			tracelog.ErrorLogger.Printf("Restore command output:\n%s", stderr.String())
			err = cmdErr
		}
		tracelog.ErrorLogger.FatalfOnError("Failed to fetch backup: %v\n", err)
	}
}

// StreamBackupToCommandStdin downloads and decompresses backup stream to cmd stdin.
func StreamBackupToCommandStdin(cmd *exec.Cmd, backup *Backup) error {
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to fetch backup: %v", err)
	}
	err = cmd.Start()
	if err != nil {
		return fmt.Errorf("failed to start command: %v", err)
	}
	err = downloadAndDecompressStream(backup, stdin)
	if err != nil {
		return fmt.Errorf("failed to download and decompress stream: %v", err)
	}
	return cmd.Wait()
}

// TODO : unit tests
// HandleBackupFetch is invoked to perform wal-g backup-fetch
func HandleBackupFetch(folder storage.Folder,
	targetBackupSelector BackupSelector,
	fetcher func(folder storage.Folder, backup Backup)) {
	backupName, err := targetBackupSelector.Select(folder)
	tracelog.ErrorLogger.FatalOnError(err)
	tracelog.DebugLogger.Printf("HandleBackupFetch(%s, folder,)\n", backupName)
	backup, err := GetBackupByName(backupName, utility.BaseBackupPath, folder)
	tracelog.ErrorLogger.FatalfOnError("Failed to fetch backup: %v\n", err)

	fetcher(folder, *backup)
}

func GetBackupByName(backupName, subfolder string, folder storage.Folder) (*Backup, error) {
	baseBackupFolder := folder.GetSubFolder(subfolder)

	var backup *Backup
	if backupName == LatestString {
		latest, err := getLatestBackupName(folder)
		if err != nil {
			return nil, err
		}
		tracelog.InfoLogger.Printf("LATEST backup is: '%s'\n", latest)

		backup = NewBackup(baseBackupFolder, latest)
	} else {
		backup = NewBackup(baseBackupFolder, backupName)

		exists, err := backup.CheckExistence()
		if err != nil {
			return nil, err
		}
		if !exists {
			return nil, NewBackupNonExistenceError(backupName)
		}
	}
	return backup, nil
}

// If specified - choose specified, else choose from latest sentinelDto
func chooseTablespaceSpecification(sentinelDtoSpec, spec *TablespaceSpec) *TablespaceSpec {
	// spec is preferred over sentinelDtoSpec.TablespaceSpec if it is non-nil
	if spec != nil {
		return spec
	} else if sentinelDtoSpec == nil {
		return &TablespaceSpec{}
	}
	return sentinelDtoSpec
}

// TODO : unit tests
// deltaFetchRecursion function composes Backup object and recursively searches for necessary base backup
func deltaFetchRecursionOld(backupName string, folder storage.Folder, dbDataDirectory string,
	tablespaceSpec *TablespaceSpec, filesToUnwrap map[string]bool) error {
	backup, err := GetBackupByName(backupName, utility.BaseBackupPath, folder)
	if err != nil {
		return err
	}
	sentinelDto, err := backup.GetSentinel()
	if err != nil {
		return err
	}
	tablespaceSpec = chooseTablespaceSpecification(sentinelDto.TablespaceSpec, tablespaceSpec)
	sentinelDto.TablespaceSpec = tablespaceSpec

	if sentinelDto.IsIncremental() {
		tracelog.InfoLogger.Printf("Delta from %v at LSN %x \n",
			*(sentinelDto.IncrementFrom),
			*(sentinelDto.IncrementFromLSN))
		baseFilesToUnwrap, err := GetBaseFilesToUnwrap(sentinelDto.Files, filesToUnwrap)
		if err != nil {
			return err
		}
		err = deltaFetchRecursionOld(*sentinelDto.IncrementFrom, folder, dbDataDirectory, tablespaceSpec, baseFilesToUnwrap)
		if err != nil {
			return err
		}
		tracelog.InfoLogger.Printf("%v fetched. Upgrading from LSN %x to LSN %x \n",
			*(sentinelDto.IncrementFrom), *(sentinelDto.IncrementFromLSN),
			*(sentinelDto.BackupStartLSN))
	}

	return backup.unwrapToEmptyDirectory(dbDataDirectory, sentinelDto, filesToUnwrap, false)
}

func GetBaseFilesToUnwrap(backupFileStates BackupFileList,
	currentFilesToUnwrap map[string]bool) (map[string]bool, error) {
	baseFilesToUnwrap := make(map[string]bool)
	for file := range currentFilesToUnwrap {
		fileDescription, hasDescription := backupFileStates[file]
		if !hasDescription {
			if _, ok := UtilityFilePaths[file]; !ok {
				tracelog.ErrorLogger.Panicf("Wanted to fetch increment for file: '%s', but didn't find one in base", file)
			}
			continue
		}
		if fileDescription.IsSkipped || fileDescription.IsIncremented {
			baseFilesToUnwrap[file] = true
		}
	}
	return baseFilesToUnwrap, nil
}
