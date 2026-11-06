package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/db"
	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/evidence"
	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/search"
	"github.com/spf13/cobra"
)

type evidenceRequest = evidence.Request

func newEvidenceCmd() *cobra.Command {
	var anchors []string
	var input string
	var radius, maxBytes, maxTurns int
	var jsonOut bool
	cmd := &cobra.Command{
		Use:   "evidence --anchor <session:turn[-turn]>... | --input <file|->",
		Short: "Fetch and merge bounded evidence from several recall anchors",
		Long: `Fetch original turns for several anchors in one call. Accepts recall JSON
with --input (use - for stdin), a previous evidence page with next, or repeated
--anchor session:turn[-turn] flags. Append @startByte-endByte for a precise
slice of a long turn. Short session handles and full IDs work.

Matching turns are admitted before neighboring context. Overlapping slices are
deduplicated; output groups each session chronologically and marks turn gaps.
Every slice carries its original role, assertion timestamp, source session and
turn citation, plus byte bounds when a long turn is only partly delivered.

--max-bytes bounds original UTF-8 text, not JSON metadata or model tokens.
--max-turns bounds delivered slices. Truncated output includes a next request;
pass that output back through --input to continue. Across pages, neighboring
context may overlap. This command never infers a turn for an unlocated session
match, and never writes to the ledger or publishes memory.`,
		Example: `  rekal --json "why did we change retries?" | rekal evidence --input - -j
  rekal evidence --anchor s3:12-15 --anchor s8:42 --max-bytes 16000
  rekal evidence --input page.json --json`,
		Args: rejectExtraArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			root, err := RequireInitializedRepo(cmd)
			if err != nil {
				return err
			}
			if radius < 0 || radius > 8 {
				return fmt.Errorf("--radius must be between 0 and 8")
			}
			if maxBytes < 128 || maxBytes > 1024*1024 {
				return fmt.Errorf("--max-bytes must be between 128 and 1048576")
			}
			if maxTurns < 1 || maxTurns > 512 {
				return fmt.Errorf("--max-turns must be between 1 and 512")
			}
			var req evidenceRequest
			var warnings []string
			if input != "" {
				r := cmd.InOrStdin()
				if input != "-" {
					f, e := os.Open(input)
					if e != nil {
						return e
					}
					defer f.Close()
					r = f
				}
				data, e := io.ReadAll(io.LimitReader(r, 4*1024*1024+1))
				if e != nil {
					return e
				}
				if len(data) > 4*1024*1024 {
					return fmt.Errorf("evidence input exceeds 4 MiB")
				}
				req, warnings, err = decodeEvidenceRequest(data)
				if err != nil {
					return err
				}
			}
			for _, text := range anchors {
				a, e := parseEvidenceAnchor(text)
				if e != nil {
					return e
				}
				req.Anchors = append(req.Anchors, a)
			}
			if len(req.Anchors) == 0 && len(warnings) == 0 {
				return fmt.Errorf("provide --anchor or --input with located recall anchors")
			}
			d, e := db.OpenIndexReadOnly(root)
			if e != nil {
				return e
			}
			defer d.Close()
			out, e := evidence.Fetch(d, req, radius, maxBytes, maxTurns)
			if e != nil {
				return e
			}
			out.Warnings = append(out.Warnings, warnings...)
			if jsonOut {
				return json.NewEncoder(cmd.OutOrStdout()).Encode(out)
			}
			for _, w := range out.Windows {
				fmt.Fprintf(cmd.OutOrStdout(), "session %s turns %d-%d\n", w.SessionID, w.StartTurn, w.EndTurn)
				for _, t := range w.Turns {
					fmt.Fprintf(cmd.OutOrStdout(), "[%s %s %s]\n%s\n\n", t.Citation, t.Role, t.Ts, t.Content)
				}
				for _, g := range w.Gaps {
					fmt.Fprintf(cmd.OutOrStdout(), "[gap: turns %d-%d not delivered]\n", g.StartTurn, g.EndTurn)
				}
			}
			if out.Truncated {
				fmt.Fprintln(cmd.OutOrStdout(), "[budget reached; use --json to retain the next-page request]")
			}
			for _, warning := range out.Warnings {
				fmt.Fprintf(cmd.ErrOrStderr(), "rekal: %s\n", warning)
			}
			return nil
		},
	}
	cmd.Flags().StringArrayVar(&anchors, "anchor", nil, "Source session:turn[-turn][@startByte-endByte]; repeat for several anchors")
	cmd.Flags().StringVar(&input, "input", "", "Recall/evidence JSON file, or - for stdin")
	cmd.Flags().IntVar(&radius, "radius", 2, "Neighboring turns on each side (0..8)")
	cmd.Flags().IntVar(&maxBytes, "max-bytes", 16000, "Maximum original UTF-8 text bytes per page (metadata excluded)")
	cmd.Flags().IntVar(&maxTurns, "max-turns", 80, "Maximum delivered turn slices per page")
	cmd.Flags().BoolVarP(&jsonOut, "json", "j", false, "JSON with source citations, gaps and next-page request")
	cmd.MarkFlagsMutuallyExclusive("anchor", "input")
	return cmd
}

func parseEvidenceAnchor(text string) (search.Anchor, error) {
	var a search.Anchor
	separator := strings.LastIndex(text, ":")
	if separator <= 0 {
		return a, fmt.Errorf("invalid anchor %q: use session:turn[-turn]", text)
	}
	parts := []string{text[:separator], text[separator+1:]}
	a.SessionID = parts[0]
	location := strings.SplitN(parts[1], "@", 2)
	bounds := strings.SplitN(location[0], "-", 2)
	start, err := strconv.Atoi(bounds[0])
	if err != nil || start < 0 {
		return a, fmt.Errorf("invalid anchor turn in %q", text)
	}
	a.StartTurn, a.EndTurn = start, start
	if len(bounds) == 2 {
		end, e := strconv.Atoi(bounds[1])
		if e != nil || end < start {
			return a, fmt.Errorf("invalid anchor range in %q", text)
		}
		a.EndTurn = end
	}
	if len(location) == 2 {
		bytes := strings.SplitN(location[1], "-", 2)
		if len(bytes) != 2 {
			return a, fmt.Errorf("invalid byte range in %q", text)
		}
		lo, e := strconv.Atoi(bytes[0])
		if e != nil || lo < 0 {
			return a, fmt.Errorf("invalid byte start in %q", text)
		}
		hi, e := strconv.Atoi(bytes[1])
		if e != nil || hi <= 0 || (a.StartTurn == a.EndTurn && hi <= lo) {
			return a, fmt.Errorf("invalid byte end in %q", text)
		}
		a.StartByte, a.EndByte = lo, hi
	}
	return a, nil
}

func decodeEvidenceRequest(data []byte) (evidenceRequest, []string, error) {
	var envelope struct {
		Anchors    []search.Anchor  `json:"anchors"`
		Cursor     int              `json:"cursor"`
		ByteOffset int              `json:"byte_offset"`
		Radius     *int             `json:"radius"`
		PlanHash   string           `json:"plan_hash"`
		Next       *evidenceRequest `json:"next"`
		Results    []search.Result  `json:"results"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil {
		return evidenceRequest{}, nil, err
	}
	if envelope.Next != nil {
		return *envelope.Next, nil, nil
	}
	req := evidenceRequest{Anchors: envelope.Anchors, Cursor: envelope.Cursor, ByteOffset: envelope.ByteOffset, Radius: envelope.Radius, PlanHash: envelope.PlanHash}
	var warnings []string
	var collect func([]search.Result)
	collect = func(results []search.Result) {
		for _, r := range results {
			if len(r.Anchors) > 0 {
				req.Anchors = append(req.Anchors, r.Anchors...)
			} else if r.Snippet != "" && r.SnippetTurnIdx >= 0 && r.MatchScope != "session" && r.MatchScope != "preview" {
				sid := r.SnippetSessionID
				if sid == "" {
					sid = r.SessionID
				}
				req.Anchors = append(req.Anchors, search.Anchor{SessionID: sid, StartTurn: r.SnippetTurnIdx, EndTurn: r.SnippetTurnIdx})
			} else {
				warnings = append(warnings, fmt.Sprintf("session %s has no located evidence anchor; drill it explicitly", r.SessionID))
			}
			collect(r.Children)
		}
	}
	collect(envelope.Results)
	// Group headers can repeat a child's anchor. Dedup before enforcing the
	// input budget; unlike the recall seed cap this allows up to 128 locations.
	seen := map[string]bool{}
	var unique []search.Anchor
	for _, a := range req.Anchors {
		key := fmt.Sprintf("%s:%d:%d:%d:%d", a.SessionID, a.StartTurn, a.EndTurn, a.StartByte, a.EndByte)
		if !seen[key] {
			seen[key] = true
			unique = append(unique, a)
		}
	}
	req.Anchors = unique
	return req, warnings, nil
}
