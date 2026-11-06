package db

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"strings"
	"unicode/utf8"
)

const passageDDL = `
CREATE TABLE IF NOT EXISTS session_passages (
 id VARCHAR PRIMARY KEY, session_id VARCHAR NOT NULL,
 start_turn INTEGER NOT NULL, end_turn INTEGER NOT NULL,
 start_byte INTEGER NOT NULL, end_byte INTEGER NOT NULL,
 content VARCHAR NOT NULL, content_hash VARCHAR NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_passage_session ON session_passages(session_id);
CREATE TABLE IF NOT EXISTS passage_embeddings (
 passage_id VARCHAR NOT NULL, model VARCHAR NOT NULL,
 embedding FLOAT[], generated_at TIMESTAMP,
 PRIMARY KEY (passage_id, model)
);
CREATE TABLE IF NOT EXISTS session_passage_state (
 session_id VARCHAR PRIMARY KEY, turn_count BIGINT NOT NULL, max_turn INTEGER NOT NULL
);`

// Passage is a bounded view of original turns. Byte offsets refer to the
// original first/last turn, never the formatted embedding document.
type Passage struct {
	ID, SessionID                          string
	StartTurn, EndTurn, StartByte, EndByte int
	Content, ContentHash                   string
}

type PassageTurn struct {
	Index                    int
	Role, Content, Timestamp string
}

// These are document/output budgets, not relevance thresholds. Every speaker
// gets the same treatment. Long turns split without losing a byte; neighboring
// documents overlap by one piece so decisions at a boundary stay searchable.
const PassageMaxBytes = 8 * 1024
const passageBytes = PassageMaxBytes
const passageTurns = 8

func passageHash(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}

func BuildPassages(sessionID string, turns []PassageTurn) []Passage {
	type piece struct {
		turn       PassageTurn
		start, end int
		text       string
	}
	var pieces []piece
	for _, t := range turns {
		for start := 0; start < len(t.Content); {
			end := min(start+passageBytes-512, len(t.Content))
			for end < len(t.Content) && !utf8.RuneStart(t.Content[end]) {
				end--
			}
			text := fmt.Sprintf("[turn=%d role=%s ts=%s bytes=%d:%d]\n%s", t.Index, t.Role, t.Timestamp, start, end, t.Content[start:end])
			pieces = append(pieces, piece{t, start, end, text})
			start = end
		}
	}
	var out []Passage
	for start := 0; start < len(pieces); {
		end, size := start+1, len(pieces[start].text)
		for end < len(pieces) && end-start < passageTurns && size+2+len(pieces[end].text) <= passageBytes {
			size += 2 + len(pieces[end].text)
			end++
		}
		text := make([]string, 0, end-start)
		for _, p := range pieces[start:end] {
			text = append(text, p.text)
		}
		content := strings.Join(text, "\n\n")
		p := Passage{SessionID: sessionID, StartTurn: pieces[start].turn.Index, EndTurn: pieces[end-1].turn.Index,
			StartByte: pieces[start].start, EndByte: pieces[end-1].end, Content: content,
			ContentHash: passageHash("rekal-passage-v1\x00" + content)}
		p.ID = passageHash(sessionID + "\x00" + p.ContentHash)
		out = append(out, p)
		if end == len(pieces) {
			break
		}
		if end-start > 1 {
			start = end - 1
		} else {
			start = end
		}
	}
	return out
}

// RefreshSessionPassages is index-only. Unchanged sessions need no transcript
// read. On append only changed tail documents/vectors are replaced; stable
// content-addressed passages retain their vectors. No ledger table is opened.
func RefreshSessionPassages(d *sql.DB, sessionIDs ...string) error {
	if _, err := d.Exec(passageDDL); err != nil {
		return err
	}
	if len(sessionIDs) == 0 {
		for _, q := range []string{
			`DELETE FROM passage_embeddings WHERE passage_id IN (SELECT id FROM session_passages WHERE session_id NOT IN (SELECT session_id FROM session_facets))`,
			`DELETE FROM session_passages WHERE session_id NOT IN (SELECT session_id FROM session_facets)`,
			`DELETE FROM session_passage_state WHERE session_id NOT IN (SELECT session_id FROM session_facets)`,
		} {
			if _, err := d.Exec(q); err != nil {
				return err
			}
		}
	}
	where, args := "", []interface{}{}
	if len(sessionIDs) > 0 {
		marks := make([]string, len(sessionIDs))
		for i, sid := range sessionIDs {
			marks[i] = fmt.Sprintf("$%d", i+1)
			args = append(args, sid)
		}
		where = " WHERE t.session_id IN (" + strings.Join(marks, ",") + ")"
	}
	rows, err := d.Query(`WITH extents AS (SELECT t.session_id,count(*) AS n,max(t.turn_index) AS last_turn
 FROM turns_ft t JOIN session_facets f ON f.session_id=t.session_id`+where+`
 GROUP BY t.session_id)
 SELECT e.session_id,e.n,e.last_turn FROM extents e LEFT JOIN session_passage_state s ON s.session_id=e.session_id
 WHERE s.turn_count IS DISTINCT FROM e.n OR s.max_turn IS DISTINCT FROM e.last_turn`, args...)
	if err != nil {
		return err
	}
	type state struct {
		sid   string
		count int64
		last  int
	}
	var changed []state
	for rows.Next() {
		var s state
		if err := rows.Scan(&s.sid, &s.count, &s.last); err != nil {
			_ = rows.Close()
			return err
		}
		changed = append(changed, s)
	}
	err = rows.Err()
	_ = rows.Close()
	if err != nil {
		return err
	}
	for _, s := range changed {
		rows, err = d.Query(`SELECT turn_index,role,content,COALESCE(ts,'') FROM turns_ft WHERE session_id=$1 ORDER BY turn_index,id`, s.sid)
		if err != nil {
			return err
		}
		var turns []PassageTurn
		for rows.Next() {
			var t PassageTurn
			if err := rows.Scan(&t.Index, &t.Role, &t.Content, &t.Timestamp); err != nil {
				_ = rows.Close()
				return err
			}
			turns = append(turns, t)
		}
		err = rows.Err()
		_ = rows.Close()
		if err != nil {
			return err
		}
		if err := replacePassages(d, s.sid, BuildPassages(s.sid, turns), s.count, s.last); err != nil {
			return err
		}
	}
	return nil
}

func replacePassages(d *sql.DB, sid string, passages []Passage, count int64, last int) error {
	tx, err := d.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck
	if _, err = tx.Exec(`CREATE OR REPLACE TEMP TABLE current_passage_ids(id VARCHAR)`); err != nil {
		return err
	}
	for _, p := range passages {
		if _, err = tx.Exec(`INSERT INTO current_passage_ids VALUES ($1)`, p.ID); err != nil {
			return err
		}
		if _, err = tx.Exec(`INSERT INTO session_passages VALUES ($1,$2,$3,$4,$5,$6,$7,$8) ON CONFLICT DO NOTHING`, p.ID, p.SessionID, p.StartTurn, p.EndTurn, p.StartByte, p.EndByte, p.Content, p.ContentHash); err != nil {
			return err
		}
	}
	if _, err = tx.Exec(`DELETE FROM passage_embeddings WHERE passage_id IN
 (SELECT id FROM session_passages WHERE session_id=$1 AND id NOT IN (SELECT id FROM current_passage_ids))`, sid); err != nil {
		return err
	}
	if _, err = tx.Exec(`DELETE FROM session_passages WHERE session_id=$1 AND id NOT IN (SELECT id FROM current_passage_ids)`, sid); err != nil {
		return err
	}
	if _, err = tx.Exec(`INSERT INTO session_passage_state VALUES ($1,$2,$3) ON CONFLICT (session_id) DO UPDATE SET turn_count=excluded.turn_count,max_turn=excluded.max_turn`, sid, count, last); err != nil {
		return err
	}
	return tx.Commit()
}

func QueryPassageDocuments(d *sql.DB, sessionIDs []string) (map[string]string, error) {
	out := map[string]string{}
	where := ""
	var args []interface{}
	if len(sessionIDs) > 0 {
		var marks []string
		for i, sid := range sessionIDs {
			marks = append(marks, fmt.Sprintf("$%d", i+1))
			args = append(args, sid)
		}
		where = " AND p.session_id IN (" + strings.Join(marks, ",") + ")"
	}
	rows, err := d.Query(`SELECT p.id,p.content FROM session_passages p JOIN session_facets f ON f.session_id=p.session_id WHERE true`+where, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close() //nolint:errcheck
	for rows.Next() {
		var id, text string
		if err := rows.Scan(&id, &text); err != nil {
			return nil, err
		}
		out[id] = "[rekal-passage-v1]\n" + text
	}
	return out, rows.Err()
}

type PassageVector struct {
	Passage
	Vector []float64
}

func QueryPassageVectors(d *sql.DB, model string) ([]PassageVector, error) {
	var out []PassageVector
	err := VisitPassageVectors(d, model, func(p PassageVector) error { out = append(out, p); return nil })
	return out, err
}

// VisitPassageVectors streams vectors; recall retains only its best ranges,
// rather than materializing every vector and document in process memory.
func VisitPassageVectors(d *sql.DB, model string, visit func(PassageVector) error) error {
	rows, err := d.Query(`SELECT p.id,p.session_id,p.start_turn,p.end_turn,p.start_byte,p.end_byte,p.content,p.content_hash,e.embedding
 FROM session_passages p JOIN passage_embeddings e ON e.passage_id=p.id
 JOIN session_facets f ON f.session_id=p.session_id WHERE e.model=$1`, model)
	if err != nil {
		return err
	}
	defer rows.Close() //nolint:errcheck
	for rows.Next() {
		var p PassageVector
		var raw interface{}
		if err := rows.Scan(&p.ID, &p.SessionID, &p.StartTurn, &p.EndTurn, &p.StartByte, &p.EndByte, &p.Content, &p.ContentHash, &raw); err != nil {
			return err
		}
		p.Vector, err = toFloat64Slice(raw)
		if err != nil {
			return err
		}
		if err := visit(p); err != nil {
			return err
		}
	}
	return rows.Err()
}

func CountPassageEmbeddings(d *sql.DB, model string) (int, error) {
	var count int
	err := d.QueryRow(`SELECT count(*) FROM passage_embeddings e JOIN session_passages p ON p.id=e.passage_id JOIN session_facets f ON f.session_id=p.session_id WHERE e.model=$1`, model).Scan(&count)
	return count, err
}

// QueryMissingPassageDocuments bounds background working memory as well as
// model work. Cache hits inside this batch are applied without model calls.
func QueryMissingPassageDocuments(d *sql.DB, model string, sessionIDs []string, limit int) (map[string]string, error) {
	where := ""
	args := []interface{}{model}
	if len(sessionIDs) > 0 {
		var marks []string
		for i, sid := range sessionIDs {
			marks = append(marks, fmt.Sprintf("$%d", i+2))
			args = append(args, sid)
		}
		where = " AND p.session_id IN (" + strings.Join(marks, ",") + ")"
	}
	rows, err := d.Query(fmt.Sprintf(`SELECT p.id,p.content FROM session_passages p JOIN session_facets f ON f.session_id=p.session_id
 LEFT JOIN passage_embeddings e ON e.passage_id=p.id AND e.model=$1
 WHERE e.passage_id IS NULL%s ORDER BY p.id LIMIT %d`, where, limit), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close() //nolint:errcheck
	out := map[string]string{}
	for rows.Next() {
		var id, text string
		if err := rows.Scan(&id, &text); err != nil {
			return nil, err
		}
		out[id] = "[rekal-passage-v1]\n" + text
	}
	return out, rows.Err()
}

func QueryPassageEmbeddingMap(d *sql.DB, model string) (map[string][]float64, error) {
	vectors, err := QueryPassageVectors(d, model)
	out := map[string][]float64{}
	for _, p := range vectors {
		out[p.ID] = p.Vector
	}
	return out, err
}

func StorePassageEmbeddings(d *sql.DB, vectors map[string][]float64, model string) error {
	for id, vec := range vectors {
		// A passage ID is content-addressed: the same model/ID is already
		// complete. DuckDB also cannot UPDATE a list through ON CONFLICT.
		q := fmt.Sprintf(`INSERT INTO passage_embeddings VALUES ($1,$2,%s::FLOAT[],now()) ON CONFLICT (passage_id,model) DO NOTHING`, float64SliceToDuckDB(vec))
		if _, err := d.Exec(q, id, model); err != nil {
			return err
		}
	}
	return nil
}
