package internal

import (
	"github.com/wal-g/wal-g/internal/storages/storage"
	"github.com/wal-g/wal-g/internal/tracelog"
	"io"
	"path"
	"path/filepath"
	"sync"
)

// Uploader contains fields associated with uploading tarballs.
// Multiple tarballs can share one uploader.
type Uploader struct {
	uploadingFolder     storage.Folder
	compressor          Compressor
	waitGroup           *sync.WaitGroup
	deltaFileManager    *DeltaFileManager
	Success             bool
	useWalDelta         bool
	preventWalOverwrite bool
}

func NewUploader(
	compressor Compressor,
	uploadingLocation storage.Folder,
	deltaDataFolder DataFolder,
	useWalDelta, preventWalOverwrite bool,
) *Uploader {
	var deltaFileManager *DeltaFileManager = nil
	if useWalDelta {
		deltaFileManager = NewDeltaFileManager(deltaDataFolder)
	}
	return &Uploader{
		uploadingFolder:     uploadingLocation,
		compressor:          compressor,
		useWalDelta:         useWalDelta,
		waitGroup:           &sync.WaitGroup{},
		deltaFileManager:    deltaFileManager,
		preventWalOverwrite: preventWalOverwrite,
	}
}

// finish waits for all waiting parts to be uploaded. If an error occurs,
// prints alert to stderr.
func (uploader *Uploader) finish() {
	uploader.waitGroup.Wait()
	if !uploader.Success {
		tracelog.ErrorLogger.Printf("WAL-G could not complete upload.\n")
	}
}

// Clone creates similar Uploader with new WaitGroup
func (uploader *Uploader) Clone() *Uploader {
	return &Uploader{
		uploader.uploadingFolder,
		uploader.compressor,
		&sync.WaitGroup{},
		uploader.deltaFileManager,
		uploader.Success,
		uploader.useWalDelta,
		uploader.preventWalOverwrite,
	}
}

// TODO : unit tests
func (uploader *Uploader) UploadWalFile(file NamedReader) error {
	var walFileReader io.Reader

	filename := path.Base(file.Name())
	if uploader.useWalDelta && isWalFilename(filename) {
		recordingReader, err := NewWalDeltaRecordingReader(file, filename, uploader.deltaFileManager)
		if err != nil {
			walFileReader = file
		} else {
			walFileReader = recordingReader
			defer recordingReader.Close()
		}
	} else {
		walFileReader = file
	}

	return uploader.UploadFile(&NamedReaderImpl{walFileReader, file.Name()})
}

// TODO : unit tests
// UploadFile compresses a file and uploads it.
func (uploader *Uploader) UploadFile(file NamedReader) error {
	pipeWriter := &CompressingPipeWriter{
		Input:                file,
		NewCompressingWriter: uploader.compressor.NewWriter,
	}

	pipeWriter.Compress(&OpenPGPCrypter{})

	dstPath := sanitizePath(filepath.Base(file.Name()) + "." + uploader.compressor.FileExtension())
	reader := pipeWriter.Output

	err := uploader.upload(dstPath, reader)
	tracelog.InfoLogger.Println("FILE PATH:", dstPath)
	return err
}

// TODO : unit tests
func (uploader *Uploader) upload(path string, content io.Reader) error {
	err := uploader.uploadingFolder.PutObject(path, content)
	if err == nil {
		uploader.Success = true
		return nil
	}
	tracelog.ErrorLogger.Printf(tracelog.GetErrorFormatter()+"\n", err)
	return err
}
