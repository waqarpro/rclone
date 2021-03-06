// +build linux darwin freebsd

package mount

import (
	"io"
	"io/ioutil"
	"os"
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Read by byte including don't read any bytes
func TestReadByByte(t *testing.T) {
	run.skipIfNoFUSE(t)

	var data = []byte("hellohello")
	run.createFile(t, "testfile", string(data))
	run.checkDir(t, "testfile 10")

	for i := 0; i < len(data); i++ {
		fd, err := os.Open(run.path("testfile"))
		assert.NoError(t, err)
		for j := 0; j < i; j++ {
			buf := make([]byte, 1)
			n, err := io.ReadFull(fd, buf)
			assert.NoError(t, err)
			assert.Equal(t, 1, n)
			assert.Equal(t, buf[0], data[j])
		}
		err = fd.Close()
		assert.NoError(t, err)
	}

	run.rm(t, "testfile")
}

func TestReadChecksum(t *testing.T) {
	run.skipIfNoFUSE(t)

	// create file big enough so we exceed any single FUSE read
	// request
	b := make([]rune, 3*128*1024)
	for i := range b {
		b[i] = 'r'
	}
	run.createFile(t, "bigfile", string(b))

	// The hash comparison would fail in Flush, if we did not
	// ensure we read the whole file
	fd, err := os.Open(run.path("bigfile"))
	assert.NoError(t, err)
	buf := make([]byte, 10)
	_, err = io.ReadFull(fd, buf)
	assert.NoError(t, err)
	err = fd.Close()
	assert.NoError(t, err)

	// The hash comparison would fail, because we only read parts
	// of the file
	fd, err = os.Open(run.path("bigfile"))
	assert.NoError(t, err)
	// read at start
	_, err = io.ReadFull(fd, buf)
	assert.NoError(t, err)
	// read at end
	_, err = fd.Seek(int64(len(b)-len(buf)), 0)
	assert.NoError(t, err)
	_, err = io.ReadFull(fd, buf)
	// ensure we don't compare hashes
	err = fd.Close()
	assert.NoError(t, err)

	run.rm(t, "bigfile")
}

// Test double close
func TestReadFileDoubleClose(t *testing.T) {
	run.skipIfNoFUSE(t)

	run.createFile(t, "testdoubleclose", "hello")

	in, err := os.Open(run.path("testdoubleclose"))
	assert.NoError(t, err)
	fd := in.Fd()

	fd1, err := syscall.Dup(int(fd))
	assert.NoError(t, err)

	fd2, err := syscall.Dup(int(fd))
	assert.NoError(t, err)

	// close one of the dups - should produce no error
	err = syscall.Close(fd1)
	assert.NoError(t, err)

	// read from the file
	buf := make([]byte, 1)
	_, err = in.Read(buf)
	assert.NoError(t, err)

	// close it
	err = in.Close()
	assert.NoError(t, err)

	// read from the other dup - should produce no error as this
	// file is now buffered
	n, err := syscall.Read(fd2, buf)
	assert.NoError(t, err)
	assert.Equal(t, 1, n)

	// close the dup - should not produce an error
	err = syscall.Close(fd2)
	assert.NoError(t, err, "input/output error")

	run.rm(t, "testdoubleclose")
}

// Test seeking
func TestReadSeek(t *testing.T) {
	run.skipIfNoFUSE(t)

	var data = []byte("helloHELLO")
	run.createFile(t, "testfile", string(data))
	run.checkDir(t, "testfile 10")

	fd, err := os.Open(run.path("testfile"))
	assert.NoError(t, err)

	// Seek to half way
	_, err = fd.Seek(5, 0)
	assert.NoError(t, err)

	buf, err := ioutil.ReadAll(fd)
	assert.NoError(t, err)
	assert.Equal(t, buf, []byte("HELLO"))

	// Test seeking to the end
	_, err = fd.Seek(10, 0)
	assert.NoError(t, err)

	buf, err = ioutil.ReadAll(fd)
	assert.NoError(t, err)
	assert.Equal(t, buf, []byte(""))

	// Test seeking beyond the end
	_, err = fd.Seek(1000000, 0)
	assert.NoError(t, err)

	buf, err = ioutil.ReadAll(fd)
	assert.NoError(t, err)
	assert.Equal(t, buf, []byte(""))

	// Now back to the start
	_, err = fd.Seek(0, 0)
	assert.NoError(t, err)

	buf, err = ioutil.ReadAll(fd)
	assert.NoError(t, err)
	assert.Equal(t, buf, []byte("helloHELLO"))

	err = fd.Close()
	assert.NoError(t, err)

	run.rm(t, "testfile")
}
