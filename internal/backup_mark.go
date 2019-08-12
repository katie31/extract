package internal

import (
	"bytes"
	"encoding/json"
	"github.com/wal-g/wal-g/internal/tracelog"
	"github.com/pkg/errors"

	"github.com/wal-g/wal-g/internal/storages/storage"
	"github.com/wal-g/wal-g/utility"
)

// MarkBackup marks a backup as permanent or impermanent
func MarkBackup(uploader *Uploader, folder storage.Folder, backupName string, toPermanent bool) {
	tracelog.InfoLogger.Printf("Retrieving previous related backups to be marked: toPermanent=%t", toPermanent)
	metadataToUpload, err := GetMarkedBackupMetadataToUpload(folder, backupName, toPermanent)
	if err != nil {
		tracelog.ErrorLogger.Fatalf("Failed to get previous backups: %v", err)
	} else {
		tracelog.InfoLogger.Printf("Retrieved backups to be marked, marking: %v", metadataToUpload)
		err = uploader.UploadMultiple(metadataToUpload)
		if err != nil {
			tracelog.ErrorLogger.Fatalf("Failed to mark previous backups: %v", err)
		}
	}
}

// GetMarkedBackupMetadataToUpload retrieves all previous permanent or
// impermanent backup metas, including itself, any previous delta backups and
// initial full backup, in increasing order beginning from full backup,
// returning modified metadata ready to be uploaded
//
// For example, when marking backups from impermanent to permanent, we retrieve
// all currently impermanent backup metadata, set them to permanent, and return
// the modified metadata as a slice of uploadable objects
func GetMarkedBackupMetadataToUpload(
	folder storage.Folder,
	backupName string,
	toPermanent bool) ([]UploadObject, error) {
	baseBackupFolder := folder.GetSubFolder(utility.BaseBackupPath)

	backup := NewBackup(baseBackupFolder, backupName)
	meta, err := backup.FetchMeta()
	if err != nil {
		return nil, err
	}

	//raise error when backup already has that type
	if toPermanent == meta.IsPermanent {
		permanentType := "permanent"
		if !meta.IsPermanent {
			permanentType = "impermanent"
		}
		return nil, NewBackupAlreadyThisTypePermanentError(backupName, permanentType)
	}

	if toPermanent {
		return getMarkedPermanentBackupMetadata(baseBackupFolder, backupName)
	} else {
		return getMarkedImpermanentBackupMetadata(folder, backupName)
	}
}

func getMarkedPermanentBackupMetadata(baseBackupFolder storage.Folder, backupName string) ([]UploadObject, error){
	backupMetadata := []UploadObject{}

	// retrieve current backup sentinel and meta
	backup := NewBackup(baseBackupFolder, backupName)
	sentinel, err := backup.GetSentinel()
	if err != nil {
		return nil, err
	}

	meta, err := backup.FetchMeta()
	if err != nil {
		return nil, err
	}

	// only return backups that we want to update
	if !meta.IsPermanent {
		meta.IsPermanent = true
		metadataUploadObject, err := getMetadataUploadObject(backup.Name, meta)
		if err != nil {
			return nil, err
		}
		backupMetadata = append(backupMetadata, metadataUploadObject)
	}

	// return when no longer incremental
	if !sentinel.IsIncremental() {
		return backupMetadata, nil
	}

	// mark previous backup
	previousImpermanentBackupMetadata, err := getMarkedPermanentBackupMetadata(
		baseBackupFolder,
		*sentinel.IncrementFrom)
	if err != nil {
		return nil, err
	}

	previousImpermanentBackupMetadata = append(previousImpermanentBackupMetadata, backupMetadata...)
	return previousImpermanentBackupMetadata, nil
}

func getMarkedImpermanentBackupMetadata(folder storage.Folder, backupName string) ([]UploadObject, error){
	backupMetadata := []UploadObject{}
	baseBackupFolder := folder.GetSubFolder(utility.BaseBackupPath)

	// retrieve current backup meta
	backup := NewBackup(baseBackupFolder, backupName)

	meta, err := backup.FetchMeta()
	if err != nil {
		return nil, err
	}

	permanentBackups, _ := getPermanentObjects(folder)
	//  del current backup from
	delete(permanentBackups, backupName[len(utility.BackupNamePrefix):len(utility.BackupNamePrefix)+24])

	reverseLinks, err := getGraphFromBaseToIncrement(folder)
	if err != nil {
		return nil, err
	}

	if backupHasPermanentInFuture(&reverseLinks, backupName, &permanentBackups){
		return nil, NewBackupHasPermanentBackupInFutureError(backupName)
	}

	meta.IsPermanent = false
	metadataUploadObject, err := getMetadataUploadObject(backup.Name, meta)
	if err != nil {
		return nil, err
	}
	backupMetadata = append(backupMetadata, metadataUploadObject)

	return backupMetadata, nil

}

func backupHasPermanentInFuture(reverseLinks *map[string][]string, backupName string, permanentBackups *map[string]bool) (bool) {
	if _, ok := (*permanentBackups)[backupName[len(utility.BackupNamePrefix):len(utility.BackupNamePrefix)+24]]; ok {
		return true
	}

	if _, ok := (*reverseLinks)[backupName]; !ok {
		return false
	}

	for _, b := range (*reverseLinks)[backupName]{
		if backupHasPermanentInFuture(reverseLinks, b, permanentBackups) {
			return true
		}
	}

	return false
}

//return graph where nodes - backup names, edges - links from base backups to increment backups
func getGraphFromBaseToIncrement(folder storage.Folder) (map[string][]string, error){
	baseBackupFolder := folder.GetSubFolder(utility.BaseBackupPath)

	backups, err := getBackups(folder)
	if err != nil {
		return nil, err
	}

	reverseLinks := make(map[string][]string)
	for i := len(backups) - 1; i >= 0; i-- {
		b := backups[i]
		incrementFrom, isIncrement, err := getMetadataFromBackup(baseBackupFolder, b.BackupName)
		if err != nil {
			return nil, err
		}

		if isIncrement{
			if _, ok := reverseLinks[incrementFrom]; !ok{
				var list []string
				reverseLinks[incrementFrom] = list
			}
			reverseLinks[incrementFrom] = append(reverseLinks[incrementFrom], b.BackupName)
		}
	}

	return reverseLinks, nil
}

func getMetadataFromBackup(baseBackupFolder storage.Folder, backupName string) (incrementFrom string, isIncrement bool, err error){
	backup := NewBackup(baseBackupFolder, backupName)
	sentinel, err := backup.GetSentinel()
	if err != nil {
		return "", false, err
	}
	if !sentinel.IsIncremental() {
		return "", sentinel.IsIncremental(),nil
	}
	return *sentinel.IncrementFrom, sentinel.IsIncremental(),nil
}

func getMetadataUploadObject(backupName string, meta ExtendedMetadataDto) (UploadObject, error) {
	metaFilePath := storage.JoinPath(backupName, utility.MetadataFileName)
	dtoBody, err := json.Marshal(meta)
	if err != nil {
		return UploadObject{}, err
	}
	return UploadObject{metaFilePath, bytes.NewReader(dtoBody)}, nil
}

type BackupAlreadyThisTypePermanentError struct {
	error
}

//raise when user try make permanent/impermanent already permanent/impermanent backup,
func NewBackupAlreadyThisTypePermanentError(backupName string, permanentType string) BackupAlreadyThisTypePermanentError {
	return BackupAlreadyThisTypePermanentError{errors.Errorf("Backup '%s' is already %s.", backupName, permanentType)}
}


type BackupHasPermanentBackupInFutureError struct {
	error
}

func NewBackupHasPermanentBackupInFutureError(backupName string) BackupHasPermanentBackupInFutureError {
	return BackupHasPermanentBackupInFutureError{errors.Errorf("Can't mark backup '%s' as impermanent. There is permanent increment backup.", backupName)}
}

