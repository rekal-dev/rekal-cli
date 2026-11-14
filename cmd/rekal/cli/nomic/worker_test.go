package nomic

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"testing"
	"time"
)

func TestSocketUnavailableClassification(t *testing.T) {
	t.Parallel()
	for _, err := range []error{syscall.EPERM, syscall.EAFNOSUPPORT, syscall.EPROTONOSUPPORT} {
		if !socketUnavailable(fmt.Errorf("dial: %w", err)) {
			t.Errorf("did not recognize %v", err)
		}
	}
	for _, err := range []error{nil, syscall.ENOENT, syscall.ECONNREFUSED, syscall.EACCES} {
		if socketUnavailable(err) {
			t.Errorf("normal connection failure selected pipe worker: %v", err)
		}
	}
}
func TestPipeWorkerLifecycle(t *testing.T) {
	t.Parallel()
	for _, mode := range []string{"serve", "crash", "stall", "die-query"} {
		t.Run(mode, func(t *testing.T) {
			t.Parallel()
			cmd := exec.Command(os.Args[0], "-test.run=^TestPipeWorkerHelper$")
			cmd.Env = append(os.Environ(), "REKAL_PIPE_TEST="+mode)
			timeout := 5 * time.Second
			if mode == "stall" {
				timeout = 100 * time.Millisecond
			}
			client, err := startPipeClient(cmd, timeout)
			if mode == "crash" || mode == "stall" {
				if err == nil {
					client.Close()
					t.Fatal("expected startup failure")
				}
				if cmd.ProcessState == nil {
					t.Fatal("failed worker was not reaped")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			vec, err := client.EmbedQuery("test query")
			if mode == "die-query" {
				if err == nil {
					t.Error("worker crash did not reach caller as an error")
				}
			} else if err != nil || len(vec) != 2 || vec[0] != 1 {
				t.Fatalf("worker round trip: %v %v", vec, err)
			}
			client.Close()
			client.Close()
			if cmd.ProcessState == nil {
				t.Fatal("worker not reaped on close")
			}
		})
	}
}
func TestPipeWorkerHelper(t *testing.T) {
	mode := os.Getenv("REKAL_PIPE_TEST")
	if mode == "" {
		return
	}
	if mode == "crash" {
		os.Exit(2)
	}
	if mode == "stall" {
		time.Sleep(time.Minute)
		os.Exit(0)
	}
	for {
		var req daemonRequest
		if err := readMsg(os.Stdin, &req); err != nil {
			os.Exit(0)
		}
		resp := daemonResponse{OK: true}
		if req.Op == "embed_query" {
			if mode == "die-query" {
				os.Exit(2)
			}
			resp.Vector = []float64{1, 0}
		}
		if err := writeMsg(os.Stdout, resp); err != nil {
			os.Exit(3)
		}
	}
}
