package walg_test

import (
	"github.com/stretchr/testify/assert"
	"github.com/wal-g/wal-g"
	"github.com/wal-g/wal-g/testtools"
	"github.com/wal-g/wal-g/walparser"
	"testing"
)

const (
	WalgTestDataFolderPath = "./testdata"
	WalFilename            = "00000001000000000000007C"
	DeltaFilename          = "000000010000000000000070_delta"
)

var TestLocation = *walparser.NewBlockLocation(1, 2, 3, 4)

func TestGetDeltaFileNameFor(t *testing.T) {
	deltaFilename, err := walg.GetDeltaFilenameFor(WalFilename)
	assert.NoError(t, err)
	assert.Equal(t, DeltaFilename, deltaFilename)
}

func assertContainsTestLocation(t *testing.T, storage testtools.MockStorage) {
	storageDeltaFilePath := "bucket/server/wal_005/000000010000000000000070_delta.mock"
	locationBuffer := storage[storageDeltaFilePath]
	reader := walg.NewBlockLocationReader(&locationBuffer)
	location, err := reader.ReadNextLocation()
	assert.NoError(t, err)
	assert.NotNil(t, location)
	assert.Equal(t, TestLocation, *location)
}
