package cli

import (
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/db"
	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/evidence"
	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/memory"
	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/search"
	"github.com/spf13/cobra"
)

const readByteBudget = 16000

type readCursor struct {
	Version  int               `json:"v"`
	Ref      string            `json:"ref"`
	Request  *evidence.Request `json:"request,omitempty"`
	LastTurn int               `json:"last_turn,omitempty"`
	Offset   int               `json:"offset,omitempty"`
}

type referenceOutput struct {
	Ref        string           `json:"ref"`
	Kind       string           `json:"kind"`
	Evidence   *evidence.Output `json:"evidence,omitempty"`
	Source     *memory.Source   `json:"source,omitempty"`
	Path       string           `json:"path,omitempty"`
	Checkpoint string           `json:"checkpoint_id,omitempty"`
	GitSHA     string           `json:"git_sha,omitempty"`
	ObservedAt string           `json:"observed_at,omitempty"`
	Content    string           `json:"content,omitempty"`
	StartByte  int              `json:"start_byte,omitempty"`
	EndByte    int              `json:"end_byte,omitempty"`
	Deleted    bool             `json:"deleted,omitempty"`
	Next       string           `json:"next,omitempty"`
}

func runReferenceRead(cmd *cobra.Command, root, ref, cursor string, jsonOut bool) error {
	state := readCursor{Version: 1, Ref: ref}
	if cursor != "" {
		if ref != "" || len(cursor) > 16384 {
			return fmt.Errorf("use either --ref or --cursor")
		}
		b, err := base64.RawURLEncoding.DecodeString(cursor)
		if err != nil {
			return fmt.Errorf("invalid read continuation")
		}
		if err := json.Unmarshal(b, &state); err != nil || state.Version != 1 || state.Ref == "" || state.Offset < 0 {
			return fmt.Errorf("invalid read continuation")
		}
	}
	var out referenceOutput
	var next *readCursor
	if strings.HasPrefix(state.Ref, "memory:") || strings.HasPrefix(state.Ref, "local-memory:") {
		d, err := db.OpenIndex(root)
		if err != nil {
			return err
		}
		defer d.Close()
		local := strings.HasPrefix(state.Ref, "local-memory:")
		table := "memory_archive"
		if local {
			cfg, err := readMergedConfig(root)
			if err != nil {
				return err
			}
			if err := refreshLocalMemory(d, root, cfg); err != nil {
				return err
			}
			table = "local_memory_documents"
		}
		id := strings.SplitN(state.Ref, ":", 2)[1]
		var payload string
		if err := d.QueryRow("SELECT payload FROM "+table+" WHERE id=?", id).Scan(&payload); err != nil {
			if err == sql.ErrNoRows {
				return fmt.Errorf("reference unavailable: %s (local snapshots can expire; repository history may need sync)", state.Ref)
			}
			return err
		}
		var r memory.Revision
		if err := json.Unmarshal([]byte(payload), &r); err != nil {
			return err
		}
		if err := r.Validate(); err != nil {
			return err
		}
		if state.Offset > len(r.Content) || (state.Offset < len(r.Content) && !utf8.RuneStart(r.Content[state.Offset])) {
			return fmt.Errorf("invalid read offset")
		}
		end := min(len(r.Content), state.Offset+readByteBudget)
		for end < len(r.Content) && !utf8.RuneStart(r.Content[end]) {
			end--
		}
		out = referenceOutput{Ref: state.Ref, Kind: "memory", Source: &r.Source, Path: r.Path, Checkpoint: r.CheckpointID, GitSHA: r.GitSHA, ObservedAt: r.ObservedAt, Content: r.Content[state.Offset:end], StartByte: state.Offset, EndByte: end, Deleted: r.Deleted}
		if local {
			out.Kind = "local-memory"
		}
		if end < len(r.Content) {
			state.Offset = end
			next = &state
		}
	} else {
		d, err := db.OpenIndexReadOnly(root)
		if err != nil {
			return err
		}
		defer d.Close()
		if state.Request == nil {
			if strings.HasPrefix(state.Ref, "passage:") {
				a, err := parseEvidenceAnchor(strings.TrimPrefix(state.Ref, "passage:"))
				if err != nil {
					return err
				}
				m, err := db.LoadSessionSIDMap(d)
				if err != nil {
					return err
				}
				a.SessionID, err = m.Resolve(a.SessionID)
				if err != nil {
					return err
				}
				state.Ref = search.AnchorReference(a)
				state.Request = &evidence.Request{Anchors: []search.Anchor{a}}
			} else {
				handle := strings.TrimPrefix(state.Ref, "session:")
				m, err := db.LoadSessionSIDMap(d)
				if err != nil {
					return err
				}
				id, err := m.Resolve(handle)
				if err != nil {
					return err
				}
				var last sql.NullInt64
				if err := d.QueryRow("SELECT max(turn_index) FROM turns_ft WHERE session_id=?", id).Scan(&last); err != nil {
					return err
				}
				if !last.Valid {
					return fmt.Errorf("session reference unavailable: %s", handle)
				}
				state.Ref = "session:" + id
				state.LastTurn = int(last.Int64)
				state.Request = &evidence.Request{Anchors: []search.Anchor{{SessionID: id, StartTurn: 0, EndTurn: min(255, state.LastTurn)}}}
			}
		}
		if len(state.Request.Anchors) != 1 || state.LastTurn < 0 {
			return fmt.Errorf("invalid read continuation anchors")
		}
		a := state.Request.Anchors[0]
		// A continuation must remain attached to its named source.
		if strings.HasPrefix(state.Ref, "session:") && a.SessionID != strings.TrimPrefix(state.Ref, "session:") {
			return fmt.Errorf("read continuation source mismatch")
		}
		if strings.HasPrefix(state.Ref, "passage:") && search.AnchorReference(a) != state.Ref {
			return fmt.Errorf("read continuation source mismatch")
		}
		got, err := evidence.Fetch(d, *state.Request, 0, readByteBudget, 80)
		if err != nil {
			return err
		}
		out = referenceOutput{Ref: state.Ref, Kind: "conversation", Evidence: &got}
		if got.Next != nil {
			state.Request = got.Next
			next = &state
		} else if strings.HasPrefix(state.Ref, "session:") && a.EndTurn < state.LastTurn {
			start := a.EndTurn + 1
			state.Request = &evidence.Request{Anchors: []search.Anchor{{SessionID: a.SessionID, StartTurn: start, EndTurn: min(start+255, state.LastTurn)}}}
			next = &state
		}
	}
	if next != nil {
		b, _ := json.Marshal(next)
		out.Next = base64.RawURLEncoding.EncodeToString(b)
	}
	if jsonOut {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(out)
	}
	w := cmd.OutOrStdout()
	fmt.Fprintf(w, "SOURCE kind=%s ref=%s\n", out.Kind, out.Ref)
	if out.Evidence != nil {
		for _, window := range out.Evidence.Windows {
			for _, turn := range window.Turns {
				fmt.Fprintf(w, "[%s %s %s]\n%s\n", turn.Citation, turn.Role, turn.Ts, turn.Content)
			}
			for _, gap := range window.Gaps {
				fmt.Fprintf(w, "[gap: turns %d-%d]\n", gap.StartTurn, gap.EndTurn)
			}
		}
		for _, warning := range out.Evidence.Warnings {
			fmt.Fprintf(w, "[warning: %s]\n", warning)
		}
	} else {
		fmt.Fprintf(w, "scope=%s path=%q observed=%s bytes=%d-%d deleted=%t\n%s\n", out.Source.Scope, out.Path, out.ObservedAt, out.StartByte, out.EndByte, out.Deleted, out.Content)
	}
	if out.Next != "" {
		fmt.Fprintf(w, "more: rekal query --cursor %s\n", out.Next)
	}
	return nil
}
