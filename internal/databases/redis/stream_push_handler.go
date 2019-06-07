package redis

import (
	"github.com/wal-g/wal-g/internal"
	"github.com/wal-g/wal-g/internal/tracelog"
	"github.com/wal-g/wal-g/utility"
	"io"
	"os"
	"strings"
	"time"
)

func HandleStreamPush(uploader *Uploader) {
	// Configure folder
	uploader.UploadingFolder = uploader.UploadingFolder.GetSubFolder(utility.BaseBackupPath)

	// Init backup process
	var stream io.Reader = os.Stdin
	stat, _ := os.Stdin.Stat()
	if (stat.Mode() & os.ModeCharDevice) == 0 {
		tracelog.InfoLogger.Println("Data is piped from stdin")
	} else {
		tracelog.ErrorLogger.Println("WARNING: stdin is terminal: operating in test mode!")
		stream = strings.NewReader("testtesttest")
	}
	backupName := "dump_" + time.Now().Format(time.RFC3339)
	err := uploader.UploadStream(backupName, stream)
	if err != nil {
		tracelog.ErrorLogger.Fatalf("%+v\n", err)
	}
}

func (uploader *Uploader) UploadStream(backupName string, stream io.Reader) error {
	compressed := internal.CompressAndEncrypt(stream, uploader.Compressor, internal.ConfigureCrypter())

	err := uploader.Upload(backupName, compressed)

	return err
}
