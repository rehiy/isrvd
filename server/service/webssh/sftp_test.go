package webssh

import (
	"errors"
	"io"
	"reflect"
	"testing"
)

type trackedReadCloser struct {
	close func() error
}

func (t *trackedReadCloser) Read([]byte) (int, error) { return 0, io.EOF }
func (t *trackedReadCloser) Close() error             { return t.close() }

func TestSFTPFileClosesHandleBeforeConnection(t *testing.T) {
	var closed []string
	remoteErr := errors.New("remote close failed")
	reader := &trackedReadCloser{close: func() error {
		closed = append(closed, "handle")
		return remoteErr
	}}
	file := &sftpFile{
		ReadCloser: reader,
		closeConn: func() {
			closed = append(closed, "connection")
		},
	}
	if err := file.Close(); !errors.Is(err, remoteErr) {
		t.Fatalf("Close error = %v, want %v", err, remoteErr)
	}
	if !reflect.DeepEqual(closed, []string{"handle", "connection"}) {
		t.Fatalf("resource close order = %v", closed)
	}
}
