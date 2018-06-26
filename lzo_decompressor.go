package walg

import (
	"encoding/binary"
	"github.com/pkg/errors"
	"github.com/rasky/go-lzo"
	"io"
)

// RaskyReader handles cases when the Rasky lzo package crashes.
// Occurs if byte size is too small (1-5).
type RaskyReader struct {
	Reader io.Reader
}

// Read ensures all bytes are get read for Rasky package.
func (reader *RaskyReader) Read(buffer []byte) (int, error) {
	return io.ReadFull(reader.Reader, buffer)
}

// Uncompressed is used to log compression ratio.
var Uncompressed uint32

// Compressed is used to log compression ratio.
var Compressed uint32

type LzoDecompressor struct{}

func readSkipped(src io.Reader) error {
	const toSkip = 33
	skipped := make([]byte, toSkip)

	n, err := io.ReadFull(src, skipped)
	if n != len(skipped) {
		return errors.New("DecompressLzo: did not fill skip")
	}
	if err != nil {
		return errors.Wrap(err, "DecompressLzo: read failed")
	}
	return nil
}

func readFileName(src io.Reader) error {
	var fileNameLen uint8
	binary.Read(src, binary.BigEndian, &fileNameLen)
	fileName := make([]byte, fileNameLen)
	n, err := io.ReadFull(src, fileName)
	if n != len(fileName) {
		return errors.New("DecompressLzo: did not fill filename")
	}
	if err != nil {
		return errors.Wrap(err, "DecompressLzo: read failed")
	}
	return nil
}

func readFileComment(src io.Reader) error {
	fileComment := make([]byte, 4)
	n, err := io.ReadFull(src, fileComment)
	if n != len(fileComment) {
		return errors.New("DecompressLzo: did not fill fileComment")
	}
	if err != nil {
		return errors.Wrap(err, "DecompressLzo: read failed")
	}
	return nil
}

func readUselessData(src io.Reader) error {
	uselessDataReaders := []func(reader io.Reader) error{
		readSkipped,
		readFileName,
		readFileComment,
	}
	for _, reader := range uselessDataReaders {
		err := reader(src)
		if err != nil {
			return err
		}
	}
	return nil
}

func (decompressor LzoDecompressor) Decompress(dst io.Writer, src io.Reader) error {
	err := readUselessData(src)
	if err != nil {
		return err
	}

	var uncom uint32
	var com uint32
	var check uint32

	for {
		err := binary.Read(src, binary.BigEndian, &uncom)
		if uncom == 0 {
			break
		}
		if err != nil {
			return errors.Wrap(err, "DecompressLzo: read failed")
		}

		err = binary.Read(src, binary.BigEndian, &com)
		if err != nil {
			return errors.Wrap(err, "DecompressLzo: read failed")
		}

		Uncompressed += uncom
		Compressed += com

		err = binary.Read(src, binary.BigEndian, &check)
		if err != nil {
			return errors.Wrap(err, "DecompressLzo: read failed")
		}

		if uncom <= com {
			n, err := io.CopyN(dst, src, int64(com))
			if n != int64(com) {
				return errors.New("DecompressLzo: copy failed")
			}
			if err != nil {
				return errors.Wrap(err, "DecompressLzo: copy failed")
			}

		} else {
			raskyReader := &RaskyReader{
				Reader: src,
			}

			out, err := lzo.Decompress1X(raskyReader, int(com), int(uncom))
			if err != nil {
				return errors.Wrap(err, "DecompressLzo: decompress lzo failed")
			}

			if len(out) != int(uncom) {
				return errors.New("DecompressLzo: out bytes do not equal uncompressed")
			}

			n, err := dst.Write(out)
			if n != len(out) {
				return errors.New("DecompressLzo: write to pipe failed")
			}
			if err != nil {
				return errors.Wrap(err, "DecompressLzo: write to pipe failed")
			}
		}
	}
	return nil
}

func (decompressor LzoDecompressor) FileExtension() string {
	return LzoFileExtension
}
