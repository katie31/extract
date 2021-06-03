package internal_test

import (
	"bufio"
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"strconv"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/wal-g/wal-g/internal"
	"github.com/wal-g/wal-g/internal/crypto/openpgp"
	"github.com/wal-g/wal-g/testtools"
	"github.com/wal-g/wal-g/utility"
)

const (
	PrivateKeyFilePath = "../test/testdata/waleGpgKey"
	randomBytesAmount = 1024
	seed = 4
	minBufferSize = 1024
)

func TestExtractAll_noFilesProvided(t *testing.T) {
	buf := &testtools.NOPTarInterpreter{}
	err := internal.ExtractAll(buf, []internal.ReaderMaker{})
	assert.IsType(t, err, internal.NoFilesToExtractError{})
}

func TestExtractAll_fileDoesntExist(t *testing.T) {
	readerMaker := &testtools.FileReaderMaker{Key: "testdata/booba.tar"}
	err := internal.ExtractAll(&testtools.NOPTarInterpreter{}, []internal.ReaderMaker {readerMaker})
	assert.Error(t, err)
}

func generateRandomBytes() []byte {
	sb := testtools.NewStrideByteReader(seed)
	lr := &io.LimitedReader{
		R: sb,
		N: int64(randomBytesAmount),
	}
	b, _ := ioutil.ReadAll(lr)
	return b
}

func makeTar() (BufferReaderMaker, []byte) {
	b := generateRandomBytes()
	bCopy := make([]byte, len(b))
	copy(bCopy, b)

	r, w := io.Pipe()
	go func() {
		bw := bufio.NewWriterSize(w, minBufferSize)

		defer utility.LoggedClose(w, "")
		defer func() {
			if err := bw.Flush(); err != nil {
				panic(err)
			}
		}()

		testtools.CreateTar(bw, &io.LimitedReader{
			R: bytes.NewBuffer(b),
			N: int64(len(b)),
		})

	}()
	tarContents := &bytes.Buffer{}
	io.Copy(tarContents, r)

	return BufferReaderMaker{tarContents, "/usr/local.tar"}, bCopy
}

func TestExtractAll_simpleTar(t *testing.T){
	os.Setenv("WALG_DOWNLOAD_CONCURRENCY", "1")
	defer os.Unsetenv("WALG_DOWNLOAD_CONCURRENCY")

	brm, b := makeTar()

	buf := &testtools.BufferTarInterpreter{}
	files := []internal.ReaderMaker{&brm}

	err := internal.ExtractAll(buf, files)
	if err != nil {
		t.Log(err)
	}

	assert.Equalf(t, b, buf.Out, "ExtractAll: Output does not match input.")
}

func TestExtractAll_multipleTars(t *testing.T) {
	os.Setenv("WALG_DOWNLOAD_CONCURRENCY", "1")
	defer os.Unsetenv("WALG_DOWNLOAD_CONCURRENCY")

	fileAmount := 3
	bufs := [][]byte {}
	brms := []internal.ReaderMaker{}

	for i := 0; i < fileAmount; i++{
		brm, b := makeTar()
		bufs = append(bufs, b)
		brms = append(brms, &brm)
	}

	buf := testtools.NewConcurrentConcatBufferTarInterpreter()

	err := internal.ExtractAll(buf, brms)
	if err != nil {
		t.Log(err)
	}

	for i := 0; i < fileAmount; i++ {
		assert.Equal(t, bufs[i], buf.Out[strconv.Itoa(i + 1)], "Some of outputs do not match input")
	}
}

func TestExtractAll_multipleConcurrentTars(t *testing.T) {
	os.Setenv("WALG_DOWNLOAD_CONCURRENCY", "4")
	defer os.Unsetenv("WALG_DOWNLOAD_CONCURRENCY")

	fileAmount := 24
	bufs := [][]byte {}
	brms := []internal.ReaderMaker{}

	for i := 0; i < fileAmount; i++{
		brm, b := makeTar()
		bufs = append(bufs, b)
		brms = append(brms, &brm)
	}

	buf := testtools.NewConcurrentConcatBufferTarInterpreter()

	err := internal.ExtractAll(buf, brms)
	if err != nil {
		t.Log(err)
	}

	for i := 0; i < fileAmount; i++ {
		assert.Equal(t, bufs[i], buf.Out[strconv.Itoa(i + 1)], "Some of outputs do not match input")
	}
}

func noPassphrase() (string, bool) {
	return "", false
}

func TestDecryptAndDecompressTar_unencrypted(t *testing.T) {
	b := generateRandomBytes()
	bCopy := make([]byte, len(b))
	copy(bCopy, b)

	compressor := GetLz4Compressor()
	compressed := internal.CompressAndEncrypt(bytes.NewReader(b), compressor, nil)

	compressedBuffer := &bytes.Buffer{}
	_, _ = compressedBuffer.ReadFrom(compressed)
	brm := &BufferReaderMaker{compressedBuffer, "/usr/local/test.tar.lz4"}

	decompressed := &bytes.Buffer{}
	err := internal.DecryptAndDecompressTar(decompressed, brm, nil)

	if err != nil {
		t.Logf("%+v\n", err)
	}
	assert.Equalf(t, bCopy, decompressed.Bytes(), "decompressed tar does not match the input")
}

func TestDecryptAndDecompressTar_encrypted(t *testing.T) {
	b := generateRandomBytes()

	// Copy generated bytes to another slice to make the test more robust against modifications of "b".
	bCopy := make([]byte, len(b))
	copy(bCopy, b)

	crypter := openpgp.CrypterFromKeyPath(PrivateKeyFilePath, noPassphrase)

	compressor := GetLz4Compressor()
	compressed := internal.CompressAndEncrypt(bytes.NewReader(b), compressor, crypter)

	compressedBuffer, _ := ioutil.ReadAll(compressed)
	brm := &BufferReaderMaker{bytes.NewBuffer(compressedBuffer), "/usr/local/test.tar.lz4"}

	decompressed := &bytes.Buffer{}
	err := internal.DecryptAndDecompressTar(decompressed, brm, crypter)

	if err != nil {
		t.Logf("%+v\n", err)
	}

	assert.Equalf(t, bCopy, decompressed.Bytes(), "decompressed tar does not match the input")
}

func TestDecryptAndDecompressTar_noCrypter(t *testing.T) {
	b := generateRandomBytes()

	// Copy generated bytes to another slice to make the test more robust against modifications of "b".
	bCopy := make([]byte, len(b))
	copy(bCopy, b)

	crypter := openpgp.CrypterFromKeyPath(PrivateKeyFilePath, noPassphrase)

	compressor := GetLz4Compressor()
	compressed := internal.CompressAndEncrypt(bytes.NewReader(b), compressor, crypter)

	compressedBuffer, _ := ioutil.ReadAll(compressed)
	brm := &BufferReaderMaker{bytes.NewBuffer(compressedBuffer), "/usr/local/test.tar.lz4"}

	decompressed := &bytes.Buffer{}
	err := internal.DecryptAndDecompressTar(decompressed, brm, nil)

	if err != nil {
		t.Logf("%+v\n", err)
	}

	assert.Error(t, err)
	originalError := errors.Cause(err)
	assert.IsType(t, internal.DecompressionError{}, originalError)
}

func TestDecryptAndDecompressTar_wrongCrypter(t *testing.T) {
	b := generateRandomBytes()

	// Copy generated bytes to another slice to make the test more robust against modifications of "b".
	bCopy := make([]byte, len(b))
	copy(bCopy, b)

	crypter := openpgp.CrypterFromKeyPath(PrivateKeyFilePath, noPassphrase)

	compressor := GetLz4Compressor()
	compressed := internal.CompressAndEncrypt(bytes.NewReader(b), compressor, crypter)

	compressedBuffer, _ := ioutil.ReadAll(compressed)
	brm := &BufferReaderMaker{bytes.NewBuffer(compressedBuffer), "/usr/local/test.tar.lzma"}

	decompressed := &bytes.Buffer{}
	err := internal.DecryptAndDecompressTar(decompressed, brm, crypter)

	assert.Error(t, err)
	originalError := errors.Cause(err)
	assert.IsType(t, internal.DecompressionError{}, originalError)
}

func TestDecryptAndDecompressTar_unknownFormat(t *testing.T) {
	b := generateRandomBytes()

	// Copy generated bytes to another slice to make the test more robust against modifications of "b".
	bCopy := make([]byte, len(b))
	copy(bCopy, b)

	brm := &BufferReaderMaker{bytes.NewBuffer(b), "/usr/local/test.some_unsupported_file_format"}

	decompressed := &bytes.Buffer{}
	err := internal.DecryptAndDecompressTar(decompressed, brm, nil)

	if err != nil {
		t.Logf("%+v\n", err)
	}

	assert.Error(t, err)
	assert.IsType(t, internal.UnsupportedFileTypeError{}, err)
}

func TestDecryptAndDecompressTar_uncompressed(t *testing.T) {
	b := generateRandomBytes()
	bCopy := make([]byte, len(b))
	copy(bCopy, b)

	compressed := internal.CompressAndEncrypt(bytes.NewReader(b), nil, nil)

	compressedBuffer := &bytes.Buffer{}
	_, _ = compressedBuffer.ReadFrom(compressed)
	brm := &BufferReaderMaker{compressedBuffer, "/usr/local/test.tar"}

	decompressed := &bytes.Buffer{}
	err := internal.DecryptAndDecompressTar(decompressed, brm, nil)

	if err != nil {
		t.Logf("%+v\n", err)
	}
	assert.Equalf(t, bCopy, decompressed.Bytes(), "decompressed tar does not match the input")
}

// Used to mock files in memory.
type BufferReaderMaker struct {
	Buf *bytes.Buffer
	Key string
}

func (b *BufferReaderMaker) Reader() (io.ReadCloser, error) { return ioutil.NopCloser(b.Buf), nil }
func (b *BufferReaderMaker) Path() string                   { return b.Key }
