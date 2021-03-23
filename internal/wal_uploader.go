package internal

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/viper"
	"github.com/wal-g/storages/fs"

	"github.com/wal-g/storages/storage"
	"github.com/wal-g/tracelog"
	"github.com/wal-g/wal-g/internal/compression"
	"github.com/wal-g/wal-g/internal/ioextensions"
	"github.com/wal-g/wal-g/utility"
)

// WalUploader extends uploader with wal specific functionality.
type WalUploader struct {
	*Uploader
	*DeltaFileManager
}

const (
	WalBulkMetadataLevel       = "BULK"
	WalIndividualMetadataLevel = "INDIVIDUAL"
	WalNoMetadataLevel         = "NOMETADATA"
)

var WalMetadataLevels = []string{WalBulkMetadataLevel, WalIndividualMetadataLevel, WalNoMetadataLevel}

type WalMetadataDescription struct {
	CreatedTime    time.Time `json:"created_time"`
	DatetimeFormat string    `json:"date_fmt"`
}

func checkWalMetadataLevel(walMetadataLevel string) error {
	isCorrect := false
	for _, level := range WalMetadataLevels {
		if walMetadataLevel == level {
			isCorrect = true
		}
	}
	if !isCorrect {
		return errors.Errorf("got incorrect Wal metadata  level: '%s', expected one of: '%v'", walMetadataLevel, WalMetadataLevels)
	}
	return nil
}

func (walUploader *WalUploader) getUseWalDelta() (useWalDelta bool) {
	return walUploader.DeltaFileManager != nil
}

func NewWalUploader(
	compressor compression.Compressor,
	uploadingLocation storage.Folder,
	deltaFileManager *DeltaFileManager,
) *WalUploader {
	uploader := NewUploader(compressor, uploadingLocation)

	return &WalUploader{
		uploader,
		deltaFileManager,
	}
}

// Clone creates similar WalUploader with new WaitGroup
func (walUploader *WalUploader) clone() *WalUploader {
	return &WalUploader{
		walUploader.Uploader.clone(),
		walUploader.DeltaFileManager,
	}
}

// TODO : unit tests
func (walUploader *WalUploader) UploadWalFile(file ioextensions.NamedReader) error {
	var walFileReader io.Reader

	filename := path.Base(file.Name())
	if walUploader.getUseWalDelta() && isWalFilename(filename) {
		recordingReader, err := NewWalDeltaRecordingReader(file, filename, walUploader.DeltaFileManager)
		if err != nil {
			walFileReader = file
		} else {
			walFileReader = recordingReader
			defer utility.LoggedClose(recordingReader, "")
		}
	} else {
		walFileReader = file
	}

	err := walUploader.UploadFile(ioextensions.NewNamedReaderImpl(walFileReader, file.Name()))
	if err == nil {
		if err := walUploader.ArchiveStatusManager.MarkWalUploaded(filename); err != nil {
			tracelog.ErrorLogger.Printf("Error marking wal file %s as uploaded: %v", filename, err)
		}
		if viper.GetString(UploadWalMetadata) != WalNoMetadataLevel {
			return uploadWALMetadataFile(walUploader, file)
		}
	}
	return err
}

func (walUploader *WalUploader) FlushFiles() {
	walUploader.DeltaFileManager.FlushFiles(walUploader.Uploader)
}

func uploadWALMetadataFile(uploader *WalUploader, walFile ioextensions.NamedReader) error {
	err := checkWalMetadataLevel(viper.GetString(UploadWalMetadata))
	if err != nil {
		return errors.Wrapf(err, "Incorrect value for parameter WALG_UPLOAD_WAL_METADATA")
	}
	walFilePath := walFile.Name()
	walFileName := path.Base(walFile.Name())
	var walMetadata WalMetadataDescription
	walMetadataMap := make(map[string]WalMetadataDescription)
	walMetadataName := walFileName + ".json"
	var walMetadataBulkUploadPath string
	walMetadata.DatetimeFormat = "%Y-%m-%dT%H:%M:%S.%fZ"
	isSourceWalPush := true
	if walFilePath == walFileName {
		isSourceWalPush = false
	}

	/* Identifying if the WAL files are generated by wal-push(archive_command) or from the wal-receive command.
	   Identifying timestamp of the WAL file generated will be bit different as wal-receive can run from any remote machine and may not have access to the pg_wal/pg_xlog folder
	   on the postgres cluster machine.
	*/

	if isSourceWalPush {
		// WAL files generated from the wal-push command
		fileStat, err := os.Stat(walFilePath)
		if err != nil {
			return errors.Wrapf(err, "upload: could not stat wal file'%s'\n", walFilePath)
		}
		walMetadata.CreatedTime = fileStat.ModTime().UTC()

	} else {
		// WAL files generated from the wal-receive command
		walMetadata.CreatedTime = time.Now().UTC()
	}
	walMetadataMap[walFileName] = walMetadata

	dtoBody, err := json.Marshal(walMetadataMap)
	if err != nil {
		return errors.Wrapf(err, "Unable to marshal walmetadata")
	}
	if viper.GetString(UploadWalMetadata) == WalBulkMetadataLevel {
		if isSourceWalPush {
			walMetadataBulkUploadPath = getRelativeArchiveDataFolderPath()
		} else {
			if viper.IsSet(WalReceiveMetadataPath) {
				walMetadataBulkUploadPath = viper.GetString(WalReceiveMetadataPath)
			} else {
				return errors.Wrapf(errors.New("WALG_RECEIVE_BULKMETADATA_PATH is not set for Bulk Wal Metadata upload"), "")
			}
		}
		walMetadataFolder := fs.NewFolder(walMetadataBulkUploadPath, "")
		err = walMetadataFolder.PutObject(walMetadataName, bytes.NewReader(dtoBody))
		if err != nil {
			return errors.Wrapf(err, "upload: could not Upload metadata'%s'\n", walFilePath)
		}
		// Calling uploadBulkMetadata only for the wal-receive. For wal-push it will be called from wal_push_handler itself.
		// wal-receive is synchronous command and each wal-file will be generated in sequence. But wal-push can run in parallel
		// based on the WALG_UPLOAD_CONCURRENCY parameter and may lead to missing consolidated metadata file.
		if !isSourceWalPush {
			err = uploadBulkMetadata(uploader, filepath.Join(walMetadataBulkUploadPath, walFileName))
		}
	} else {
		err = uploader.Upload(walMetadataName, bytes.NewReader(dtoBody))
	}
	return errors.Wrapf(err, "upload: could not Upload metadata'%s'\n", walFilePath)
}

func uploadBulkMetadata(uploader *WalUploader, walFilePath string) error {

	// Creating consolidated wal metadata only for bulk option
	// Checking if the walfile name ends with "F" (last file in the series) and consolidating all the metadata together.
	// For example, All the metadata for the files in the series 000000030000000800000010, 000000030000000800000011 to 00000003000000080000001F
	// will be consolidated together and single  file 00000003000000080000001.json will be created.
	// Parameter isSourceWalPush will identify if the source of the file is from wal-push or from wal-receive.

	if viper.GetString(UploadWalMetadata) != WalBulkMetadataLevel || walFilePath[len(walFilePath)-1:] != "F" {
		return nil
	}
	walMetadataBulkFolderPath := filepath.Dir(walFilePath)
	walFileName := filepath.Base(walFilePath)
	walMetadataBulkFolder := fs.NewFolder(walMetadataBulkFolderPath, "")
	walSearchString := walFileName[0 : len(walFileName)-1]
	walMetadataFiles, _ := filepath.Glob(walMetadataBulkFolder.GetFilePath("") + "/" + walSearchString + "*.json")

	walMetadata := make(map[string]WalMetadataDescription)
	walMetadataArray := make(map[string]WalMetadataDescription)

	for _, walMetadataFile := range walMetadataFiles {
		file, _ := ioutil.ReadFile(walMetadataFile)
		err := json.Unmarshal(file, &walMetadata)
		if err == nil {
			for k := range walMetadata {
				walMetadataArray[k] = walMetadata[k]
			}
		}
	}
	dtoBody, _ := json.Marshal(walMetadataArray)
	err := uploader.Upload(walSearchString+".json", bytes.NewReader(dtoBody))
	//Deleting the temporary metadata files created
	for _, walMetadataFile := range walMetadataFiles {
		err := os.Remove(walMetadataFile)
		if err != nil {
			tracelog.InfoLogger.Printf("Unable to remove walmetadata file %s", walMetadataFile)
		}
	}
	return errors.Wrapf(err, "Unable to upload bulk wal metadata %s", walFilePath)
}
