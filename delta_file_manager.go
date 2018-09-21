package walg

import (
	"bytes"
	"fmt"
	"github.com/wal-g/wal-g/walparser"
	"sync"
)

type DeltaFileWriterNotFoundError struct {
	filename string
}

func (err DeltaFileWriterNotFoundError) Error() string {
	return fmt.Sprintf("can't file delta file writer for file: '%s'", err.filename)
}

type DeltaFileManager struct {
	dataFolder            DataFolder
	PartFiles             *LazyCache
	DeltaFileWriters      *LazyCache
	deltaFileWriterWaiter sync.WaitGroup
	canceledWalRecordings chan string
	CanceledDeltaFiles    map[string]bool
	canceledWaiter        sync.WaitGroup
}

func NewDeltaFileManager(dataFolder DataFolder) *DeltaFileManager {
	manager := &DeltaFileManager{
		dataFolder,
		nil,
		nil,
		sync.WaitGroup{},
		make(chan string),
		make(map[string]bool),
		sync.WaitGroup{},
	}
	manager.PartFiles = NewLazyCache(func(partFilenameInterface interface{}) (partFile interface{}, err error) {
		partFilename, ok := partFilenameInterface.(string)
		if !ok {
			return nil, WrongTypeError{"string"}
		}
		return manager.LoadPartFile(partFilename)
	})
	manager.DeltaFileWriters = NewLazyCache(func(deltaFilenameInterface interface{}) (deltaFileWriter interface{}, err error) {
		deltaFilename, ok := deltaFilenameInterface.(string)
		if !ok {
			return nil, WrongTypeError{"string"}
		}
		return manager.LoadDeltaFileWriter(deltaFilename)
	})
	manager.canceledWaiter.Add(1)
	go manager.collectCanceledDeltaFiles()
	return manager
}

func (manager *DeltaFileManager) GetBlockLocationConsumer(deltaFilename string) (chan walparser.BlockLocation, error) {
	deltaFileWriter, _, err := manager.DeltaFileWriters.Load(deltaFilename)
	if err != nil {
		return nil, err
	}
	return deltaFileWriter.(*DeltaFileChanWriter).BlockLocationConsumer, nil
}

// TODO : unit tests
func (manager *DeltaFileManager) LoadDeltaFileWriter(deltaFilename string) (deltaFileWriter *DeltaFileChanWriter, err error) {
	physicalDeltaFile, err := manager.dataFolder.OpenReadonlyFile(deltaFilename)
	var deltaFile *DeltaFile
	if err != nil {
		if _, ok := err.(*NoSuchFileError); !ok {
			return nil, err
		}
		deltaFile, err = NewDeltaFile(walparser.NewWalParser())
		if err != nil {
			return nil, err
		}
	} else {
		defer physicalDeltaFile.Close()
		deltaFile, err = LoadDeltaFile(physicalDeltaFile)
		if err != nil {
			return nil, err
		}
	}
	deltaFileWriter = NewDeltaFileChanWriter(deltaFile)
	manager.deltaFileWriterWaiter.Add(1)
	go deltaFileWriter.Consume(&manager.deltaFileWriterWaiter)
	return deltaFileWriter, nil
}

func (manager *DeltaFileManager) GetPartFile(deltaFilename string) (*WalPartFile, error) {
	partFilename := ToPartFilename(deltaFilename)
	partFile, _, err := manager.PartFiles.Load(partFilename)
	if err != nil {
		return nil, err
	}
	return partFile.(*WalPartFile), nil
}

// TODO : unit tests
func (manager *DeltaFileManager) LoadPartFile(partFilename string) (*WalPartFile, error) {
	physicalPartFile, err := manager.dataFolder.OpenReadonlyFile(partFilename)
	var partFile *WalPartFile
	if err != nil {
		if _, ok := err.(*NoSuchFileError); !ok {
			return nil, err
		}
		partFile = NewWalPartFile()
	} else {
		defer physicalPartFile.Close()
		partFile, err = LoadPartFile(physicalPartFile)
		if err != nil {
			return nil, err
		}
	}
	return partFile, nil
}

func (manager *DeltaFileManager) FlushPartFiles() (completedPartFiles map[string]bool) {
	close(manager.canceledWalRecordings)
	manager.canceledWaiter.Wait()
	completedPartFiles = make(map[string]bool)
	manager.PartFiles.Range(func(key, value interface{}) bool {
		partFilename := key.(string)
		partFile := value.(*WalPartFile)
		deltaFilename := partFilenameToDelta(partFilename)
		if _, ok := manager.CanceledDeltaFiles[deltaFilename]; ok {
			return true
		}
		if partFile.IsComplete() {
			completedPartFiles[partFilename] = true
			err := manager.CombinePartFile(deltaFilename, partFile)
			if err != nil {
				manager.CanceledDeltaFiles[deltaFilename] = true
				fmt.Printf("Canceled delta file writing because of error: %v\n", err)
			}
		} else {
			err := saveToDataFolder(partFile, partFilename, manager.dataFolder)
			if err != nil {
				manager.CanceledDeltaFiles[deltaFilename] = true
				fmt.Printf("Failed to save part file: '%s' because of error: '%v'\n", partFilename, err)
			}
		}
		return true
	})
	return
}

func (manager *DeltaFileManager) FlushDeltaFiles(uploader *Uploader, completedPartFiles map[string]bool) {
	manager.DeltaFileWriters.Range(func(key, value interface{}) bool {
		deltaFileWriter := value.(*DeltaFileChanWriter)
		deltaFileWriter.close()
		return true
	})
	manager.deltaFileWriterWaiter.Wait()
	manager.DeltaFileWriters.Range(func(key, value interface{}) bool {
		deltaFilename := key.(string)
		deltaFileWriter := value.(*DeltaFileChanWriter)
		if _, ok := manager.CanceledDeltaFiles[deltaFilename]; ok {
			return true
		}
		partFilename := ToPartFilename(deltaFilename)
		if _, ok := completedPartFiles[partFilename]; ok {
			var deltaFileData bytes.Buffer
			err := deltaFileWriter.DeltaFile.Save(&deltaFileData)
			if err != nil {
				fmt.Printf("Failed to upload delta file: '%s' because of saving error: '%v'\n", deltaFilename, err)
			} else {
				err = uploader.UploadFile(&NamedReaderImpl{&deltaFileData, deltaFilename})
				if err != nil {
					fmt.Printf("Failed to upload delta file: '%s' because of uploading error: '%v'\n", deltaFilename, err)
				}
			}
		} else {
			err := saveToDataFolder(deltaFileWriter.DeltaFile, deltaFilename, manager.dataFolder)
			if err != nil {
				fmt.Printf("Failed to save delta file: '%s' because of error: '%v'\n", deltaFilename, err)
			}
		}
		return true
	})
}

func (manager *DeltaFileManager) FlushFiles(uploader *Uploader) {
	err := manager.dataFolder.CleanFolder()
	if err != nil {
		fmt.Printf("Failed to clean delta folder because of error: '%v'\n", err)
	}
	completedPartFiles := manager.FlushPartFiles()
	manager.FlushDeltaFiles(uploader, completedPartFiles)
}

func (manager *DeltaFileManager) CancelRecording(walFilename string) {
	manager.canceledWalRecordings <- walFilename
}

// TODO : unit tests
func (manager *DeltaFileManager) collectCanceledDeltaFiles() {
	for walFilename := range manager.canceledWalRecordings {
		deltaFilename, err := GetDeltaFilenameFor(walFilename)
		if err != nil {
			continue
		}
		manager.CanceledDeltaFiles[deltaFilename] = true
		nextWalFilename, _ := GetNextWalFilename(walFilename)
		deltaFilename, _ = GetDeltaFilenameFor(nextWalFilename)
		manager.CanceledDeltaFiles[deltaFilename] = true
	}
	manager.canceledWaiter.Done()
}

func (manager *DeltaFileManager) CombinePartFile(deltaFilename string, partFile *WalPartFile) error {
	deltaFileWriterInterface, exists := manager.DeltaFileWriters.LoadExisting(deltaFilename)
	if !exists {
		return DeltaFileWriterNotFoundError{deltaFilename}
	}
	deltaFileWriter := deltaFileWriterInterface.(*DeltaFileChanWriter)
	deltaFileWriter.DeltaFile.WalParser = walparser.LoadWalParserFromCurrentRecordData(partFile.WalHeads[WalFileInDelta-1])
	records, err := partFile.CombineRecords()
	if err != nil {
		return err
	}
	locations := ExtractBlockLocations(records)
	for _, location := range locations {
		deltaFileWriter.BlockLocationConsumer <- location
	}
	return nil
}
