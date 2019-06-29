package internal_test

import (
	"bytes"
	"github.com/stretchr/testify/assert"
	"github.com/wal-g/wal-g/internal"
	"github.com/wal-g/wal-g/internal/walparser"
	"github.com/wal-g/wal-g/testtools"
	"io/ioutil"
	"os"
	"path"
	"testing"
)

var ParserFilePath = path.Join(WalgTestDataFolderPath, internal.RecordPartFilename)
var WalFilePath = path.Join(WalgTestDataFolderPath, WalFilename)
var DeltaFilePath = path.Join(WalgTestDataFolderPath, DeltaFilename)
var RealLocation = *walparser.NewBlockLocation(internal.DefaultSpcNode, 16384, 16397, 2062)

func createWalParser() (*walparser.WalParser, error) {
	data := testtools.CreateWalPageWithContinuation()

	walParser := walparser.NewWalParser()
	_, _, err := walParser.ParseRecordsFromPage(bytes.NewReader(data)) // initializing parsing
	if err != nil {
		return nil, err
	}
	return walParser, nil
}

func TestRecordBlockLocationsFromPage(t *testing.T) {
	walParser := walparser.NewWalParser()
	walFile, err := os.Open(WalFilePath)
	assert.NoError(t, err)
	pageReader := walparser.NewWalPageReader(walFile)
	page1, err := pageReader.ReadPageData()
	assert.NoError(t, err)
	page2, err := pageReader.ReadPageData()
	assert.NoError(t, err)

	_, _, err = walParser.ParseRecordsFromPage(bytes.NewReader(page1)) // initializing
	assert.NoError(t, err)

	blockLocationConsumer := make(chan walparser.BlockLocation)
	recordingReader := internal.WalDeltaRecordingReader{
		WalParser:        *walParser,
		PageDataLeftover: page2,
		Recorder:         internal.NewWalDeltaRecorder(blockLocationConsumer),
	}
	go func() {
		err = recordingReader.RecordBlockLocationsFromPage()
		assert.NoError(t, err)

		close(blockLocationConsumer)
	}()
	locations := make([]walparser.BlockLocation, 0)
	for location := range blockLocationConsumer {
		locations = append(locations, location)
	}
	assert.Len(t, locations, 1)
	assert.Equal(t, RealLocation, locations[0])
}

func TestRead_CorrectData(t *testing.T) {
	data, err := ioutil.ReadFile(WalFilePath)
	assert.NoError(t, err)
	reader := internal.WalDeltaRecordingReader{
		PageReader: *walparser.NewWalPageReader(bytes.NewReader(data)),
	}
	actualData := make([]byte, len(data), cap(data))
	n, err := reader.Read(actualData)
	assert.NoError(t, err)
	assert.Equal(t, len(data), n)
	assert.Equal(t, data, actualData)
}

func TestRead_CorrectRecording(t *testing.T) {
	walFile, err := os.Open(WalFilePath)
	assert.NoError(t, err)
	defer walFile.Close()

	dataFolder := testtools.NewMockDataFolder()
	manager := internal.NewDeltaFileManager(dataFolder)
	recordingReader, err := internal.NewWalDeltaRecordingReader(walFile, WalFilename, manager)
	assert.NoError(t, err)

	_, err = ioutil.ReadAll(recordingReader)
	assert.NoError(t, err)
	manager.FlushFiles(nil)

	locations, err := internal.ReadLocationsFrom((*dataFolder)[DeltaFilename])
	assert.NoError(t, err)
	assert.Len(t, locations, 1)
	assert.Equal(t, RealLocation, locations[0])
}

func TestRead_RecordingFail(t *testing.T) {
	walData := make([]byte, walparser.WalPageSize*3)
	for i := range walData {
		walData[i] = 1
	}

	dataFolder := testtools.NewMockDataFolder()
	manager := internal.NewDeltaFileManager(dataFolder)
	recordingReader, err := internal.NewWalDeltaRecordingReader(bytes.NewReader(walData), WalFilename, manager)
	assert.NoError(t, err)

	actualData, err := ioutil.ReadAll(recordingReader)
	assert.NoError(t, err)
	manager.FlushFiles(nil)

	assert.Equal(t, walData, actualData)
	assert.True(t, dataFolder.IsEmpty())
}
