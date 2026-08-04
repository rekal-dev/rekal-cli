// Package transport encodes checkpoints to the append-only wire format on the
// rekal orphan branch and decodes them back — the git-side sync mechanism that
// keeps data.db thin on the wire. It sits above codec (frame format), db
// (storage), and gitx (git plumbing).
package transport

import (
	"database/sql"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/codec"
	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/db"
	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/gitx"
)

func ExportNewFrames(gitRoot string) ([]byte, []byte, []string, error) {
	dataDB, err := db.OpenData(gitRoot)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("open data DB: %w", err)
	}
	defer dataDB.Close()

	checkpoints, err := db.QueryUnexportedCheckpoints(dataDB)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("query unexported checkpoints: %w", err)
	}
	if len(checkpoints) == 0 {
		return nil, nil, nil, nil
	}

	// Merged-only gate: share only sessions whose code has landed on the
	// mainline. data.db keeps every branch locally; only the wire is filtered,
	// so unmerged/abandoned work never reaches teammates. Held-back checkpoints
	// stay unexported and re-evaluate on the next push, releasing automatically
	// once their branch merges. See docs/design/merged-only-sharing.md.
	checkpoints = filterMerged(dataDB, gitRoot, checkpoints)
	if len(checkpoints) == 0 {
		return nil, nil, nil, nil
	}

	// Load existing wire format from orphan branch.
	branch := gitx.RekalBranchName()
	bodyData := gitx.ShowFile(gitRoot, branch, codec.BodyFilename)
	dictData := gitx.ShowFile(gitRoot, branch, codec.DictFilename)

	dict := codec.NewDict()
	if len(dictData) > 0 {
		loaded, err := codec.LoadDict(dictData)
		if err == nil {
			dict = loaded
		}
	}
	body := bodyData
	if len(body) == 0 {
		body = codec.NewBody()
	}

	return encodeCheckpointFrames(dataDB, body, dict, checkpoints)
}

// ExportAllFrames re-encodes every merged checkpoint in data.db into a fresh
// body and dict, ignoring the orphan branch's current contents and exported
// flags. Used by `rekal push --re-export` to regenerate a branch whose wire
// data is corrupt (frames written before the v2 count fix) or bloated (stale
// meta frames) — data.db is the source of truth, the branch is derived.
//
// The merged-only gate applies here too: a repair regenerates the branch as
// merged-only, so it can never re-leak unmerged work that a plain push would
// have held back.
func ExportAllFrames(gitRoot string) ([]byte, []byte, []string, error) {
	dataDB, err := db.OpenData(gitRoot)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("open data DB: %w", err)
	}
	defer dataDB.Close()

	checkpoints, err := db.QueryAllCheckpoints(dataDB)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("query checkpoints: %w", err)
	}
	if len(checkpoints) == 0 {
		return nil, nil, nil, nil
	}

	checkpoints = filterMerged(dataDB, gitRoot, checkpoints)
	if len(checkpoints) == 0 {
		return nil, nil, nil, nil
	}

	return encodeCheckpointFrames(dataDB, codec.NewBody(), codec.NewDict(), checkpoints)
}

// preferLocalIfAhead upgrades a remote-tracking default ref (origin/main) to
// the local branch of the same name (main) when the local branch is at least
// as advanced — the exact situation `rekal push` runs in from the pre-push
// hook: the hook fires before the push actually transfers anything, so
// origin/main is still whatever it was as of the last fetch, one full cycle
// behind a merge that just landed locally. Judging ancestry against that
// stale ref means a checkpoint from a branch merged and pushed in the very
// same `git push` gets held back until the *next*, unrelated push — which
// contradicts "ships automatically once the branch merges." Only upgrades
// when origin/<name> is an ancestor of (or equal to) local <name> — a
// divergence *this checkout already knows about* (it fetched, and local and
// origin/<name> are now on different lines) falls back to the original,
// remote-tracking ref instead of judging against local-only state.
//
// This is a local-only, no-network check, so it cannot see a divergence
// this checkout hasn't fetched yet — a stale origin/<name> that the actual
// remote has since moved past looks identical to a safe fast-forward. That
// gap is bounded by git itself: the code push this hook precedes would be
// rejected as non-fast-forward in that case, so the work never lands under
// the sha the checkpoint is anchored to. Cheap and correct for the common
// case (an up-to-date checkout merging and pushing its own work); the
// residual risk is a checkout that is both stale and has advanced on its
// own, which is exactly the state a `git fetch` before pushing avoids.
func preferLocalIfAhead(gitRoot, defaultRef string) string {
	local := strings.TrimPrefix(defaultRef, "origin/")
	if local == defaultRef || local == "" {
		return defaultRef // already a local ref, nothing to upgrade
	}
	if gitx.BranchTip(gitRoot, local) == "" {
		return defaultRef // no local branch of that name
	}
	if gitx.IsAncestor(gitRoot, defaultRef, local) {
		return local
	}
	return defaultRef
}

// filterMerged returns only the checkpoints whose code has merged into the
// mainline, or nil when the mainline can't be resolved. Shared by the
// incremental (ExportNewFrames) and full re-export (ExportAllFrames) paths so
// every route that writes the wire honors the merged-only guarantee —
// unmerged work must never be encoded, including by a repair. See
// docs/design/merged-only-sharing.md.
//
// Two merge signals, both exact and fail-closed:
//   - ancestry: git_sha reachable from the default branch (merge-commit and
//     rebase workflows)
//   - squash: the branch's cumulative change landed as a patch-equivalent
//     commit (gitx.IsSquashMergedInto) — covering squash-merge workflows,
//     where the original commits never become ancestors
func filterMerged(dataDB *sql.DB, gitRoot string, checkpoints []db.CheckpointRow) []db.CheckpointRow {
	defaultRef := gitx.DefaultBranch(gitRoot)
	if defaultRef == "" {
		// No resolvable mainline — nothing is provably merged, so share
		// nothing (fail closed).
		return nil
	}
	defaultRef = preferLocalIfAhead(gitRoot, defaultRef)

	// Memoize the gate against the mainline tip. A checkpoint held back because
	// its branch was abandoned is re-litigated on every push otherwise, and the
	// squash probe is not cheap: a commit-tree plus a git cherry per held
	// checkpoint, forever, to reach the same answer.
	//
	// The cache is strictly an accelerator. Every failure path below falls
	// through to the real predicates, because the gate decides what leaves the
	// machine and a cache must never be able to widen that.
	targetTip := gitx.BranchTip(gitRoot, defaultRef)
	if targetTip == "" {
		if out, err := exec.Command("git", "-C", gitRoot, "rev-parse", defaultRef).Output(); err == nil {
			targetTip = strings.TrimSpace(string(out))
		}
	}
	cached := map[string]bool{}
	if dataDB != nil && targetTip != "" {
		if v, err := db.LoadMergeGateVerdicts(dataDB, targetTip); err == nil {
			cached = v
		}
	}
	fresh := map[string]bool{}
	memo := func(name string, sha string, probe func() bool) bool {
		key := name + "\x1f" + sha
		if v, ok := cached[key]; ok {
			return v
		}
		v := probe()
		fresh[key] = v
		return v
	}

	result := shareableCheckpoints(checkpoints,
		func(sha string) bool {
			return memo("anc", sha, func() bool { return gitx.IsAncestor(gitRoot, sha, defaultRef) })
		},
		// Landed on the mainline under a different sha. This is tree- and
		// patch-based, not name-based, so it also covers a commit rewritten
		// after capture — a rebase or an amended message orphans the sha the
		// checkpoint was anchored to, and comparing patches still proves the
		// work landed.
		func(sha string) bool {
			return memo("squash", sha, func() bool { return gitx.IsSquashMergedInto(gitRoot, sha, defaultRef) })
		},
		func(branch string) string { return gitx.BranchTip(gitRoot, branch) },
		func(sha, tip string) bool { return gitx.IsAncestor(gitRoot, sha, tip) },
	)

	if dataDB != nil && targetTip != "" && len(fresh) > 0 {
		// Best-effort: a cache that cannot be written costs a repeat, not a
		// wrong answer.
		_ = db.SaveMergeGateVerdicts(dataDB, targetTip, fresh)
		_ = db.PruneMergeGateCache(dataDB, targetTip)
	}
	return result
}

// shareableCheckpoints keeps only checkpoints whose code has merged into the
// mainline. It never mutates the input slice — the held-back checkpoints must
// stay untouched so they remain unexported and are re-evaluated on the next
// push. A checkpoint with no git_sha is never shared.
//
// A checkpoint is shareable when any of:
//   - isAncestor(git_sha): its commit is literally on the mainline
//   - isSquashMerged(git_sha): its commit was the branch state whose
//     cumulative diff landed as a squash commit
//   - it is an ancestor of a proven squash point on the same branch: a squash
//     lands the branch's *final* state, so mid-branch checkpoints (earlier
//     commits of the same branch) are part of the landed history exactly as
//     they would be under a merge commit. Proven points are the branch's local
//     tip (if it still exists) and any sibling checkpoint that passed the
//     squash test.
func shareableCheckpoints(
	checkpoints []db.CheckpointRow,
	isAncestor func(sha string) bool,
	isSquashMerged func(sha string) bool,
	branchTip func(branch string) string,
	isAncestorOf func(sha, tip string) bool,
) []db.CheckpointRow {
	// Pass 1: direct proofs. Track squash-proven shas per branch for pass 2.
	proven := make(map[string]bool, len(checkpoints)) // git_sha → shareable
	squashPoints := make(map[string][]string)         // branch → proven squash shas
	var held []int

	for i, cp := range checkpoints {
		switch {
		case cp.Exported:
			// Already on the wire, so it passed this gate once. A repair
			// (push --re-export) must not re-litigate that: a commit rebased or
			// squashed away after capture leaves an orphaned git_sha, and once
			// the branch ref moves to match the mainline its cumulative diff is
			// empty, which isSquashMerged rejects by design. Re-proving is then
			// impossible and the repair would silently delete already-shared
			// conversations. Withholding unmerged work is the guarantee;
			// retracting merged work is not.
			proven[cp.GitSHA] = true
		case cp.GitSHA == "":
			// never shareable
		case isAncestor(cp.GitSHA):
			proven[cp.GitSHA] = true
		case isSquashMerged(cp.GitSHA):
			proven[cp.GitSHA] = true
			squashPoints[cp.GitBranch] = append(squashPoints[cp.GitBranch], cp.GitSHA)
		default:
			held = append(held, i)
		}
	}

	// Pass 2: release mid-branch checkpoints that are ancestors of a proven
	// squash point on their branch. The branch's surviving local tip counts as
	// a candidate too (probed at most once per branch).
	tipChecked := make(map[string]bool)
	for _, i := range held {
		cp := checkpoints[i]
		if cp.GitBranch == "" {
			continue
		}
		if !tipChecked[cp.GitBranch] {
			tipChecked[cp.GitBranch] = true
			if tip := branchTip(cp.GitBranch); tip != "" && isSquashMerged(tip) {
				squashPoints[cp.GitBranch] = append(squashPoints[cp.GitBranch], tip)
			}
		}
		for _, point := range squashPoints[cp.GitBranch] {
			if isAncestorOf(cp.GitSHA, point) {
				proven[cp.GitSHA] = true
				break
			}
		}
	}

	out := make([]db.CheckpointRow, 0, len(checkpoints))
	for _, cp := range checkpoints {
		if proven[cp.GitSHA] {
			out = append(out, cp)
		}
	}
	return out
}

// appendBatch encodes members as one FrameBatch and appends it to body. If the
// batch would overflow the frame envelope's length field (a single checkpoint
// with an implausible amount of compressed data), it falls back to appending
// each member as its own standalone frame, so the export still succeeds.
func appendBatch(body []byte, enc *codec.Encoder, members []codec.BatchMember) ([]byte, error) {
	batch, err := enc.EncodeBatch(members)
	if err == nil {
		return codec.AppendFrame(body, batch), nil
	}
	for _, m := range members {
		frame, ferr := enc.EncodeMemberFrame(m)
		if ferr != nil {
			return nil, ferr
		}
		body = codec.AppendFrame(body, frame)
	}
	return body, nil
}

// encodeCheckpointFrames appends session + checkpoint frames for the given
// checkpoints to body, interning strings into dict, and finishes with a meta
// frame. Returns the updated body, encoded dict, and the checkpoint IDs
// encoded — which the caller must only mark exported after the bytes are
// durably committed (see ExportNewFrames' doc comment).
// withheldSessions is the duplicate filter's decision: a superseded session may
// be dropped only when the survivor that replaces it is itself going out in this
// run. A filter whose job is to remove redundancy must never remove the last
// copy — see the call site for why the two can diverge.
func withheldSessions(survivors map[string]string, shipping map[string]bool) map[string]bool {
	out := make(map[string]bool, len(survivors))
	for id, survivor := range survivors {
		if shipping[survivor] {
			out[id] = true
		}
	}
	return out
}

func encodeCheckpointFrames(dataDB *sql.DB, body []byte, dict *codec.Dict, checkpoints []db.CheckpointRow) ([]byte, []byte, []string, error) {
	enc, err := codec.NewEncoder()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("create encoder: %w", err)
	}
	defer enc.Close()

	// Duplicate re-captures of one conversation must not reach the wire. This
	// fails loud on purpose: swallowing the error would silently ship the
	// duplicates it exists to prevent, and a failed export is recoverable while
	// bytes already on a teammate's branch are not.
	survivors, err := db.SupersededSessionSurvivors(dataDB)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("supersession filter: %w", err)
	}
	// A superseded session may only be withheld when its survivor is actually
	// going out in this run. The two can land on different checkpoints, and the
	// merged-only gate judges those independently — so an older, merged copy can
	// be shareable while the longer one is still held back on an unmerged
	// branch. Skipping it blindly would withhold the conversation entirely,
	// which is the opposite of what a duplicate filter is for.
	shipping := make(map[string]bool)
	for _, cp := range checkpoints {
		ids, err := db.QuerySessionsByCheckpoint(dataDB, cp.ID)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("pre-scan sessions for %s: %w", cp.ID, err)
		}
		for _, id := range ids {
			shipping[id] = true
		}
	}
	superseded := withheldSessions(survivors, shipping)

	var exportedIDs []string

	for _, cp := range checkpoints {
		// Query sessions linked to this checkpoint.
		sessionIDs, err := db.QuerySessionsByCheckpoint(dataDB, cp.ID)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("query sessions for checkpoint %s: %w", cp.ID, err)
		}

		var sessionRefs []uint64
		// One checkpoint's session frames plus its checkpoint frame are
		// batched into a single FrameBatch: sessions in one commit share
		// paths, author, and phrasing, so compressing them together is far
		// tighter than one zstd stream per frame (see codec.EncodeBatch).
		var members []codec.BatchMember

		for _, sid := range sessionIDs {
			// Skip a session that is a strict prefix of a longer one — the same
			// conversation captured again at an earlier commit, before capture
			// learned to append. data.db keeps every copy (it is the ledger),
			// but the orphan branch is derived data, so there is no reason to
			// ship one conversation several times. Shipping only the longest
			// loses nothing: it contains every turn the shorter ones held.
			if superseded[sid] {
				continue
			}
			sess, err := db.QuerySession(dataDB, sid)
			if err != nil {
				return nil, nil, nil, fmt.Errorf("query session %s: %w", sid, err)
			}
			turns, err := db.QueryTurns(dataDB, sid)
			if err != nil {
				return nil, nil, nil, fmt.Errorf("query turns for %s: %w", sid, err)
			}
			toolCalls, err := db.QueryToolCalls(dataDB, sid)
			if err != nil {
				return nil, nil, nil, fmt.Errorf("query tool_calls for %s: %w", sid, err)
			}

			sessRef := dict.LookupOrAdd(codec.NSSessions, sid)
			emailRef := dict.LookupOrAdd(codec.NSEmails, sess.Email)
			branchRef := uint64(0)
			if sess.Branch != "" {
				branchRef = dict.LookupOrAdd(codec.NSBranches, sess.Branch)
			}

			actorType := codec.ActorHuman
			agentIDRef := uint64(0)
			if sess.ActorType == "agent" {
				actorType = codec.ActorAgent
				if sess.AgentID != "" {
					agentIDRef = dict.LookupOrAdd(codec.NSEmails, sess.AgentID)
				}
			}

			capturedAt, _ := time.Parse(time.RFC3339, sess.CapturedAt)
			sf := &codec.SessionFrame{
				SessionRef: sessRef,
				CapturedAt: capturedAt,
				EmailRef:   emailRef,
				ActorType:  actorType,
				AgentIDRef: agentIDRef,

				// Harness metadata rides the v2 payload; zero values cost
				// nothing on the wire (single flags byte).
				TeamName:     sess.TeamName,
				WorkflowName: sess.WorkflowName,
				AgentType:    sess.AgentType,
				Description:  sess.Description,
				SpawnDepth:   sess.SpawnDepth,
			}
			if sess.ParentSessionID != "" {
				// A subagent whose trunk is a superseded copy must follow the
				// survivor, or the teammate receives a parent that was never
				// shipped and the subagent detaches from its conversation.
				parent := sess.ParentSessionID
				if s, ok := survivors[parent]; ok {
					parent = s
				}
				sf.HasParent = true
				sf.ParentRef = dict.LookupOrAdd(codec.NSSessions, parent)
			}

			// Build turn records with delta timestamps.
			var prevTs time.Time
			for _, t := range turns {
				role := codec.RoleHuman
				switch t.Role {
				case "assistant":
					role = codec.RoleAssistant
				case "human_steering":
					role = codec.RoleHumanSteering
				case "summary":
					role = codec.RoleSummary
				}
				var tsDelta uint64
				if t.Ts != "" {
					ts, _ := time.Parse(time.RFC3339, t.Ts)
					if !prevTs.IsZero() && !ts.IsZero() {
						delta := ts.Sub(prevTs)
						if delta > 0 {
							tsDelta = uint64(delta.Seconds())
						}
					}
					prevTs = ts
				}
				sf.Turns = append(sf.Turns, codec.TurnRecord{
					Role:      role,
					TsDelta:   tsDelta,
					BranchRef: branchRef,
					Text:      t.Content,
				})
			}

			// Build tool call records.
			for _, tc := range toolCalls {
				toolCode := codec.ToolCode(tc.Tool)
				tcr := codec.ToolCallRecord{
					Tool: toolCode,
				}
				if tc.Path == "" {
					tcr.PathFlag = codec.PathNull
				} else {
					pathRef := dict.LookupOrAdd(codec.NSPaths, tc.Path)
					tcr.PathFlag = codec.PathDictRef
					tcr.PathRef = pathRef
				}
				tcr.CmdPrefix = tc.CmdPrefix
				sf.ToolCalls = append(sf.ToolCalls, tcr)
			}

			members = append(members, codec.SessionMember(sf))
			sessionRefs = append(sessionRefs, sessRef)
		}

		// Build checkpoint frame.
		cpRef := dict.LookupOrAdd(codec.NSSessions, cp.ID)
		cpBranchRef := dict.LookupOrAdd(codec.NSBranches, cp.GitBranch)
		cpEmailRef := dict.LookupOrAdd(codec.NSEmails, cp.Email)

		cpTs, _ := time.Parse(time.RFC3339, cp.Ts)

		actorType := codec.ActorHuman
		agentIDRef := uint64(0)
		if cp.ActorType == "agent" {
			actorType = codec.ActorAgent
			if cp.AgentID != "" {
				agentIDRef = dict.LookupOrAdd(codec.NSEmails, cp.AgentID)
			}
		}

		// Query files touched.
		filesTouched, err := db.QueryFilesTouched(dataDB, cp.ID)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("query files_touched for %s: %w", cp.ID, err)
		}
		var fileRecords []codec.FileTouchedRecord
		for _, ft := range filesTouched {
			pathRef := dict.LookupOrAdd(codec.NSPaths, ft.Path)
			changeType := byte('M')
			if len(ft.ChangeType) > 0 {
				changeType = ft.ChangeType[0]
			}
			fileRecords = append(fileRecords, codec.FileTouchedRecord{
				PathRef:    pathRef,
				ChangeType: changeType,
			})
		}

		cf := &codec.CheckpointFrame{
			CheckpointRef: cpRef,
			GitSHA:        cp.GitSHA,
			BranchRef:     cpBranchRef,
			EmailRef:      cpEmailRef,
			Timestamp:     cpTs,
			ActorType:     actorType,
			AgentIDRef:    agentIDRef,
			SessionRefs:   sessionRefs,
			Files:         fileRecords,
		}
		members = append(members, codec.CheckpointMember(cf))

		body, err = appendBatch(body, enc, members)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("encode checkpoint %s: %w", cp.ID, err)
		}

		exportedIDs = append(exportedIDs, cp.ID)
	}

	// Append meta frame.
	existingFrames, _ := codec.ScanFrames(body)
	nFrames := uint32(len(existingFrames))

	email := gitx.ConfigValue("user.email")
	metaEmailRef := dict.LookupOrAdd(codec.NSEmails, email)

	mf := &codec.MetaFrame{
		FormatVersion: 0x01,
		EmailRef:      metaEmailRef,
		CheckpointSHA: strings.Repeat("0", 40), // placeholder
		Timestamp:     time.Now().UTC(),
		NSessions:     uint32(dict.Len(codec.NSSessions)),
		NCheckpoints:  uint32(len(exportedIDs)),
		NFrames:       nFrames + 1, // +1 for this meta frame
		NDictEntries:  uint32(dict.TotalEntries()),
	}
	metaFrame, err := enc.EncodeMetaFrame(mf)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("encode meta frame: %w", err)
	}
	body = codec.AppendFrame(body, metaFrame)

	// Checkpoints are NOT marked exported here — see the doc comment above.
	// The caller marks them only once CommitWireFormat has durably committed
	// this body/dict to the orphan branch.
	return body, dict.Encode(), exportedIDs, nil
}

// CommitWireFormat commits rekal.body and dict.bin to the orphan branch.
// Returns the new commit SHA.
func CommitWireFormat(gitRoot string, bodyData, dictData []byte) (string, error) {
	branch := gitx.RekalBranchName()

	// Get the current tip of the orphan branch.
	parentOut, err := exec.Command("git", "-C", gitRoot, "rev-parse", branch).Output()
	if err != nil {
		return "", fmt.Errorf("resolve branch %s: %w", branch, err)
	}
	parent := strings.TrimSpace(string(parentOut))

	bodyHash, err := gitx.HashObject(gitRoot, bodyData)
	if err != nil {
		return "", fmt.Errorf("hash rekal.body: %w", err)
	}
	dictHash, err := gitx.HashObject(gitRoot, dictData)
	if err != nil {
		return "", fmt.Errorf("hash dict.bin: %w", err)
	}

	treeEntry := fmt.Sprintf("100644 blob %s\tdict.bin\n100644 blob %s\trekal.body\n", dictHash, bodyHash)
	mktreeCmd := exec.Command("git", "-C", gitRoot, "mktree")
	mktreeCmd.Stdin = strings.NewReader(treeEntry)
	treeOut, err := mktreeCmd.Output()
	if err != nil {
		return "", fmt.Errorf("mktree: %w", err)
	}
	treeHash := strings.TrimSpace(string(treeOut))

	// Use the HEAD commit message from the main branch.
	msg := "rekal: checkpoint"
	if headMsg, err := exec.Command("git", "-C", gitRoot, "log", "-1", "--format=%s", "HEAD").Output(); err == nil {
		if m := strings.TrimSpace(string(headMsg)); m != "" {
			msg = m
		}
	}

	commitOut, err := exec.Command("git", "-C", gitRoot,
		"commit-tree", treeHash, "-p", parent, "-m", msg,
	).Output()
	if err != nil {
		return "", fmt.Errorf("commit-tree: %w", err)
	}
	commitSHA := strings.TrimSpace(string(commitOut))

	if err := exec.Command("git", "-C", gitRoot, "update-ref", "refs/heads/"+branch, commitSHA).Run(); err != nil {
		return "", fmt.Errorf("update-ref: %w", err)
	}

	return commitSHA, nil
}

// EnsureOrphanBranch creates or fetches the local rekal orphan branch.
// If the branch exists locally, it's left as-is.
// If it exists on the remote, it's fetched.
// Otherwise, a new orphan branch is created with empty rekal.body and dict.bin.
// FastForwardOrphanBranch advances the local orphan branch to the remote tip
// when the remote strictly contains it. Reports whether the ref moved.
//
// Without this, one identity on two machines is a dead end. The wire body is
// cumulative along the *local* branch — ExportNewFrames reads the body at the
// local tip and appends — so a machine whose branch is behind builds a body
// missing everything the other machine added, and its push is rejected. Nothing
// else moves the ref: EnsureOrphanBranch seeds it from the remote only at
// first creation, and neither sync nor push touches it afterwards. The only
// exits were a manual reset or a --force that overwrites the other machine's
// conversations.
//
// Strictly a fast-forward: if the local tip is not an ancestor of the remote,
// the branches have genuinely forked and this does nothing, leaving the push to
// be rejected and remoteDiverged to report it honestly. Never rewinds, never
// merges, never discards a local commit.
func FastForwardOrphanBranch(gitRoot string) (bool, error) {
	branch := gitx.RekalBranchName()
	local, err := exec.Command("git", "-C", gitRoot, "rev-parse", "--verify", branch).Output()
	if err != nil {
		return false, nil // no local branch yet — EnsureOrphanBranch seeds it
	}
	fetch := exec.Command("git", "-C", gitRoot, "fetch", "origin", branch)
	fetch.Stdin = nil
	if err := fetch.Run(); err != nil {
		return false, nil // offline or no such remote branch: nothing to do
	}
	remote, err := exec.Command("git", "-C", gitRoot, "rev-parse", "FETCH_HEAD").Output()
	if err != nil {
		return false, nil
	}
	localSHA := strings.TrimSpace(string(local))
	remoteSHA := strings.TrimSpace(string(remote))
	if localSHA == remoteSHA {
		return false, nil
	}
	if !gitx.IsAncestor(gitRoot, localSHA, remoteSHA) {
		return false, nil // forked — not ours to resolve
	}
	if err := exec.Command("git", "-C", gitRoot,
		"update-ref", "refs/heads/"+branch, remoteSHA, localSHA).Run(); err != nil {
		return false, fmt.Errorf("fast-forward %s: %w", branch, err)
	}
	return true, nil
}

func EnsureOrphanBranch(gitRoot string) error {
	branch := gitx.RekalBranchName()

	// Check if local branch already exists.
	if err := exec.Command("git", "-C", gitRoot, "rev-parse", "--verify", branch).Run(); err == nil {
		return nil // already exists locally
	}

	// Check if remote branch exists and fetch it.
	remote := "origin"
	remoteBranch := remote + "/" + branch
	// Fetch the specific branch (ignore errors — remote may not exist or branch may not exist).
	_ = exec.Command("git", "-C", gitRoot, "fetch", remote, branch).Run()

	// If remote branch now exists locally as a remote-tracking branch, create local from it.
	if err := exec.Command("git", "-C", gitRoot, "rev-parse", "--verify", remoteBranch).Run(); err == nil {
		return exec.Command("git", "-C", gitRoot, "branch", branch, remoteBranch).Run()
	}

	// Create new orphan branch with initial wire format files.
	bodyData := codec.NewBody()
	dictData := codec.NewDict().Encode()

	bodyHash, err := gitx.HashObject(gitRoot, bodyData)
	if err != nil {
		return fmt.Errorf("hash rekal.body: %w", err)
	}
	dictHash, err := gitx.HashObject(gitRoot, dictData)
	if err != nil {
		return fmt.Errorf("hash dict.bin: %w", err)
	}

	treeEntry := fmt.Sprintf("100644 blob %s\tdict.bin\n100644 blob %s\trekal.body\n", dictHash, bodyHash)
	mktreeCmd := exec.Command("git", "-C", gitRoot, "mktree")
	mktreeCmd.Stdin = strings.NewReader(treeEntry)
	treeOut, err := mktreeCmd.Output()
	if err != nil {
		return fmt.Errorf("mktree: %w", err)
	}
	treeHash := strings.TrimSpace(string(treeOut))

	commitOut, err := exec.Command("git", "-C", gitRoot,
		"commit-tree", treeHash, "-m", "rekal: initialize checkpoint branch",
	).Output()
	if err != nil {
		return fmt.Errorf("create initial commit: %w", err)
	}
	commitHash := strings.TrimSpace(string(commitOut))

	return exec.Command("git", "-C", gitRoot, "update-ref", "refs/heads/"+branch, commitHash).Run()
}
