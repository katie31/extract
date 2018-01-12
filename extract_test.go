package walg_test

import (
	"bytes"
	"fmt"
	"github.com/wal-g/wal-g"
	"github.com/wal-g/wal-g/test_tools"
	"io"
	"io/ioutil"
	"testing"
)

func TestNoFilesProvided(t *testing.T) {
	buf := &tools.BufferTarInterpreter{}
	err := walg.ExtractAll(buf, []walg.ReaderMaker{})
	if err == nil {
		t.Errorf("extract: Did not catch no files provided error")
	}
}

func TestUnsupportedFileType(t *testing.T) {
	test := &bytes.Buffer{}
	brm := &BufferReaderMaker{test, "/usr/local", "gzip"}
	buf := &tools.BufferTarInterpreter{}
	files := []walg.ReaderMaker{brm}
	err := walg.ExtractAll(buf, files)

	err.Error()
	if serr, ok := err.(*walg.UnsupportedFileTypeError); ok {
		t.Errorf("extract: Extract should not support filetype %s", brm.FileFormat)
	} else if serr != nil {
		t.Log(serr)
	}
}

// Tests roundtrip for a tar file.
func TestTar(t *testing.T) {
	//Generate and save random bytes compare against compression-decompression cycle.
	sb := tools.NewStrideByteReader(10)
	lr := &io.LimitedReader{
		R: sb,
		N: int64(1024),
	}
	b, err := ioutil.ReadAll(lr)

	//Copy generated bytes to another slice to make the test more robust against modifications of "b".
	bCopy := make([]byte, len(b))
	copy(bCopy, b)
	if err != nil {
		t.Fatal()
	}

	//Make a tar in memory.
	member := &bytes.Buffer{}
	tools.CreateTar(member, &io.LimitedReader{
		R: bytes.NewBuffer(b),
		N: int64(len(b)),
	})

	//Extract the generated tar and check that its one member is the same as the bytes generated to begin with.
	brm := &BufferReaderMaker{member, "/usr/local", "tar"}
	buf := &tools.BufferTarInterpreter{}
	files := []walg.ReaderMaker{brm}
	err = walg.ExtractAll(buf, files)
	if err != nil {
		t.Log(err)
	}

	if !bytes.Equal(bCopy, buf.Out) {
		t.Error("extract: Unbundled tar output does not match input.")
	}
}

// Test extraction of various lzo compressed tar files.
func testLzopRoundTrip(t *testing.T, stride, nBytes int) {
	//Generate and save random bytes compare against compression-decompression cycle.
	sb := tools.NewStrideByteReader(stride)
	lr := &io.LimitedReader{
		R: sb,
		N: int64(nBytes),
	}
	b, err := ioutil.ReadAll(lr)

	//Copy generated bytes to another slice to make the test more robust against modifications of "b".
	bCopy := make([]byte, len(b))
	copy(bCopy, b)
	if err != nil {
		t.Log(err)
	}

	//Compress bytes and make a tar in memory.
	tarR, memberW := io.Pipe()
	go func() {
		tools.CreateTar(memberW, &io.LimitedReader{
			R: bytes.NewBuffer(b),
			N: int64(len(b)),
		})
		memberW.Close()
	}()
	comReader := &tools.LzopReader{Uncompressed: tarR}
	lzopTarReader := bytes.NewBufferString(tools.LzopPrefix)
	_, err = io.Copy(lzopTarReader, comReader)
	lzopTarReader.Write(make([]byte, 12))
	if err != nil {
		t.Log(err)
	}

	//Extract the generated tar and check that its one member is the same as the bytes generated to begin with.
	brm := &BufferReaderMaker{lzopTarReader, "/usr/local", "lzo"}
	buf := &tools.BufferTarInterpreter{}
	files := []walg.ReaderMaker{brm}
	err = walg.ExtractAll(buf, files)
	if err != nil {
		t.Log(err)
	}

	if !bytes.Equal(bCopy, buf.Out) {
		t.Error("extract: Decompressed output does not match input.")
	}
}

func TestLzopUncompressableBytes(t *testing.T) {
	testLzopRoundTrip(t, tools.LzopBlockSize*2, tools.LzopBlockSize*2)
}
func TestLzop1Byte(t *testing.T)   { testLzopRoundTrip(t, 7924, 1) }
func TestLzop1MByte(t *testing.T)  { testLzopRoundTrip(t, 7924, 1024*1024) }
func TestLzop10MByte(t *testing.T) { testLzopRoundTrip(t, 7924, 10*1024*1024) }

// Used to mock files in memory.
type BufferReaderMaker struct {
	Buf        *bytes.Buffer
	Key        string
	FileFormat string
}

func (b *BufferReaderMaker) Reader() (io.ReadCloser, error) { return ioutil.NopCloser(b.Buf), nil }
func (b *BufferReaderMaker) Format() string                 { return b.FileFormat }
func (b *BufferReaderMaker) Path() string                   { return b.Key }

func setupRand(stride, nBytes int) *BufferReaderMaker {
	sb := tools.NewStrideByteReader(stride)
	lr := &io.LimitedReader{
		R: sb,
		N: int64(nBytes),
	}
	b := &BufferReaderMaker{bytes.NewBufferString(tools.LzopPrefix), "/usr/local", "lzo"}

	pr, pw := io.Pipe()

	go func() {
		tools.CreateTar(pw, lr)
		defer pw.Close()
	}()

	comReader := tools.LzopReader{Uncompressed: pr}
	io.Copy(b.Buf, &comReader)
	n, err := b.Buf.Write(make([]byte, 12))

	if n != 12 {
		panic("Did not write empty signal bytes. ")
	}

	if err != nil {
		panic(err)
	}

	return b
}

func BenchmarkExtractAll(b *testing.B) {
	b.SetBytes(int64(b.N * 1024 * 1024))
	out := make([]walg.ReaderMaker, 1)
	rand := setupRand(7924, b.N*1024*1024)
	fmt.Println("B.N", b.N)

	out[0] = rand

	b.ResetTimer()

	// f := &extract.FileTarInterpreter{
	// 		NewDir: "",
	// 	}
	// out[0] = f

	// extract.ExtractAll(f, out)

	// np := &extract.NOPTarInterpreter{}
	// extract.ExtractAll(np, out)

	buf := &tools.BufferTarInterpreter{}
	err := walg.ExtractAll(buf, out)
	if err != nil {
		b.Log(err)
	}

}
