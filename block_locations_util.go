package walg

import (
	"bytes"
	"github.com/wal-g/wal-g/walparser"
	"io"
)

func ExtractBlockLocations(records []walparser.XLogRecord) []walparser.BlockLocation {
	locations := make([]walparser.BlockLocation, 0)
	for _, record := range records {
		if record.IsZero() {
			continue
		}
		for _, block := range record.Blocks {
			locations = append(locations, block.Header.BlockLocation)
		}
	}
	return locations
}

// TODO : unit tests
func extractLocationsFromWalFile(parser *walparser.WalParser, walFile io.ReadCloser) ([]walparser.BlockLocation, error) {
	pageReader := walparser.NewWalPageReader(walFile)
	locations := make([]walparser.BlockLocation, 0)
	for {
		data, err := pageReader.ReadPageData()
		if err != nil {
			if err == io.EOF {
				return locations, nil
			}
			return nil, err
		}
		_, records, err := parser.ParseRecordsFromPage(bytes.NewReader(data))
		switch err.(type) {
		case nil:
		case walparser.PartialPageError:
		case walparser.ZeroPageError:
		default:
			return nil, err
		}
		locations = append(locations, ExtractBlockLocations(records)...)
	}
}
