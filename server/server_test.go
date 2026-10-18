package server

import (
	"bufio"
	"net"
	"os"
	"strings"
	"testing"
	"time"
)

// The socket path must stay under the ~100 byte unix socket limit, which a
// t.TempDir() under TMPDIR does not.
func shortTempDir(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("/tmp", "codt-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	return dir
}

func newTestServer(t *testing.T) *serverImpl {
	t.Helper()
	t.Setenv("XDG_DATA_HOME", shortTempDir(t))
	t.Setenv("XDG_CONFIG_HOME", shortTempDir(t))
	cfg, err := DefaultConfiguration()
	if err != nil {
		t.Fatal(err)
	}
	for _, dir := range []string{cfg.GetRunDir(), cfg.GetLogDir()} {
		if err = os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
	}
	srv, err := NewServer(&cfg)
	if err != nil {
		t.Fatal(err)
	}
	si := srv.(*serverImpl)
	si.mutex.Lock()
	si.initialized = true // else waitAttach log.Fatal()s mid-test
	si.mutex.Unlock()
	return si
}

// A request is a single JSON line carrying the shell's whole environment, so it
// can exceed bufio's 64 KiB default token limit. Scan() then gave up and closed
// the connection without a response, and the client, still writing, died with
// "write unix -><socket>: write: broken pipe".
func TestOversizedRequestGetsResponse(t *testing.T) {
	si := newTestServer(t)
	go si.Serve()
	defer func() { _ = si.listener.Close() }()

	conn, err := net.Dial("unix", si.configuration.GetSocketFile())
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	pad := strings.Repeat("x", 128*1024)
	req := `{"Request":"ListClientsRequest","Payload":{"Pad":"` + pad + `"}}` + "\n"
	if _, err = conn.Write([]byte(req)); err != nil {
		t.Fatalf("write oversized request: %v", err)
	}

	if err = conn.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
		t.Fatal(err)
	}
	if _, err = bufio.NewReader(conn).ReadBytes('\n'); err != nil {
		t.Fatalf("no response to oversized request: %v", err)
	}
}
