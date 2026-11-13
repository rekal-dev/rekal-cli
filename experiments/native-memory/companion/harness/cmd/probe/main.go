package main

import (
	"fmt"
	"os"

	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/transport"
)

func main() {
	if len(os.Args) < 3 { panic("usage: probe legacy REPO BODY DICT | preserve REPO REF PARENT|- NAME LOCALFILE ...") }
	mode, repo := os.Args[1], os.Args[2]
	var commit string
	var err error
	if mode == "legacy" {
		if len(os.Args) != 5 { panic("legacy arguments") }
		var body, dict []byte
		body, err = os.ReadFile(os.Args[3]); if err != nil { panic(err) }
		dict, err = os.ReadFile(os.Args[4]); if err != nil { panic(err) }
		commit, err = transport.CommitWireFormat(repo, body, dict)
	} else if mode == "preserve" {
		if len(os.Args) < 7 || len(os.Args) % 2 != 1 { panic("preserve arguments") }
		ref, parent := os.Args[3], os.Args[4]
		if parent == "-" { parent = "" }
		updates := map[string][]byte{}
		for i := 5; i < len(os.Args); i += 2 {
			data, e := os.ReadFile(os.Args[i+1]); if e != nil { panic(e) }
			updates[os.Args[i]] = data
		}
		commit, err = transport.CommitFiles(repo, ref, parent, updates)
	} else { panic("unknown mode") }
	if err != nil { fmt.Fprintln(os.Stderr, err); os.Exit(1) }
	fmt.Println(commit)
}
