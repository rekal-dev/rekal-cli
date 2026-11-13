package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/codec"
	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/memoryframe"
	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/transport"
)

func must(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func read(path string) []byte     { b, e := os.ReadFile(path); must(e); return b }
func write(path string, b []byte) { must(os.WriteFile(path, b, 0600)) }
func emit(v any)                  { b, e := json.Marshal(v); must(e); fmt.Println(string(b)) }

func main() {
	a := os.Args[1:]
	if len(a) == 0 {
		panic("mode required")
	}
	switch a[0] {
	case "seed":
		body, dict, err := memoryframe.Seed()
		must(err)
		write(a[1], body)
		write(a[2], dict.Encode())
	case "append":
		dict, err := codec.LoadDict(read(a[2]))
		must(err)
		payloads := [][]byte{}
		for _, p := range a[7:] {
			payloads = append(payloads, read(p))
		}
		body, err := memoryframe.AppendCheckpoint(read(a[1]), dict, a[5], a[6], payloads, true)
		must(err)
		write(a[3], body)
		write(a[4], dict.Encode())
	case "extract":
		v, err := memoryframe.MemoryRecords(read(a[1]))
		must(err)
		emit(v)
	case "legacy-read":
		v, err := memoryframe.LegacyCounts(read(a[1]))
		must(err)
		emit(v)
	case "session-text":
		dict, err := codec.LoadDict(read(a[2]))
		must(err)
		members, err := memoryframe.Members(read(a[1]))
		must(err)
		out := map[string][]string{}
		for _, m := range members {
			if m.Type != codec.FrameSession && m.Type != codec.FrameSessionV2 {
				continue
			}
			session, err := codec.DecodeSessionPayload(m.Payload)
			must(err)
			id, err := dict.Get(codec.NSSessions, session.SessionRef)
			must(err)
			turns := []string{}
			for _, turn := range session.Turns {
				turns = append(turns, turn.Text)
			}
			out[id] = turns
		}
		emit(out)
	case "reexport":
		v, err := memoryframe.ReexportModel(read(a[1]), a[3] == "aware")
		must(err)
		write(a[2], v)
	case "commit":
		sha, err := transport.CommitWireFormat(a[1], read(a[2]), read(a[3]))
		must(err)
		fmt.Println(sha)
	default:
		panic("unknown mode")
	}
}
