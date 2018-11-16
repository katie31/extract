package walg

import (
	"bytes"
	"fmt"
	"github.com/wal-g/wal-g/walparser"
	"io"
)

const (
	RecordPartFilename = "currentRecord.part"
)

// In case of recording error WalDeltaRecordingReader stops recording, but continues reading data correctly
type WalDeltaRecordingReader struct {
	PageReader       walparser.WalPageReader
	WalParser        walparser.WalParser
	PageDataLeftover []byte
	Recorder         *WalDeltaRecorder
	partRecorder     *WalPartRecorder
	firstRead        bool
}

func NewWalDeltaRecordingReader(walFileReader io.Reader, walFilename string, manager *DeltaFileManager) (*WalDeltaRecordingReader, error) {
	walParser, recorder, partRecorder, err := tryOpenParserAndRecorders(walFilename, manager)
	if err != nil {
		return nil, err
	}
	return &WalDeltaRecordingReader{
		*walparser.NewWalPageReader(walFileReader),
		*walParser,
		nil,
		recorder,
		partRecorder,
		true,
	}, nil
}

func (reader *WalDeltaRecordingReader) Close() error {
	err := reader.partRecorder.SaveNextWalHead(reader.WalParser.GetCurrentRecordData())
	if err != nil {
		fmt.Printf("Failed to save next wal file prefix after end of recording because of: %v", err)
	}
	return err
}

func (reader *WalDeltaRecordingReader) Read(p []byte) (n int, err error) {
	dataExpected := len(p)
	for {
		if len(p) <= len(reader.PageDataLeftover) {
			copy(p, reader.PageDataLeftover[:len(p)])
			reader.PageDataLeftover = reader.PageDataLeftover[len(p):]
			return dataExpected, nil
		}
		copy(p, reader.PageDataLeftover)
		p = p[len(reader.PageDataLeftover):]
		reader.PageDataLeftover, err = reader.PageReader.ReadPageData()
		if err != nil {
			if err != io.EOF {
				reader.partRecorder.cancelRecordingWithErr(err)
			}
			return dataExpected - len(p), err
		}
		recordingErr := reader.RecordBlockLocationsFromPage()
		if recordingErr != nil {
			reader.partRecorder.cancelRecordingWithErr(recordingErr)
		}
	}
}

func (reader *WalDeltaRecordingReader) RecordBlockLocationsFromPage() error {
	defer func() { reader.firstRead = false }()
	if reader.Recorder == nil {
		return nil
	}
	discardedRecordTail, records, err := reader.WalParser.ParseRecordsFromPage(bytes.NewReader(reader.PageDataLeftover))
	if reader.firstRead {
		err = reader.partRecorder.SavePreviousWalTail(discardedRecordTail)
		if err != nil {
			return err
		}
	}
	if _, ok := err.(walparser.PartialPageError); !ok && err != nil {
		if _, ok := err.(walparser.ZeroPageError); ok {
			return nil
		}
		reader.WalParser.Invalidate()
		return err
	}
	reader.Recorder.recordWalDelta(records)
	return nil
}

func tryOpenParserAndRecorders(walFilename string, manager *DeltaFileManager) (*walparser.WalParser, *WalDeltaRecorder, *WalPartRecorder, error) {
	walParser := walparser.NewWalParser()
	deltaFilename, err := GetDeltaFilenameFor(walFilename)
	if err != nil {
		return nil, nil, nil, err
	}
	blockLocationConsumer, err := manager.GetBlockLocationConsumer(deltaFilename)
	if err != nil {
		return nil, nil, nil, err
	}
	recorder := NewWalDeltaRecorder(blockLocationConsumer)
	partRecorder, err := NewWalPartRecorder(walFilename, manager)
	if err != nil {
		return nil, nil, nil, err
	}
	return walParser, recorder, partRecorder, nil
}
