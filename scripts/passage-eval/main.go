// passage-eval compares evidence delivery, not answer accuracy. It performs
// no network calls and never opens a developer's .rekal store.
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/db"
	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/evidence"
	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/lsa"
	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/nomic"
	"github.com/rekal-dev/rekal-cli/cmd/rekal/cli/search"
)

type turn struct {
	Role string `json:"role"`
	Text string `json:"text"`
}
type session struct {
	ID    string `json:"session_id"`
	Date  string `json:"date"`
	Turns []turn `json:"turns"`
}
type question struct {
	ID       string `json:"question_id"`
	Text     string `json:"question"`
	Category string `json:"category"`
	Extra    struct {
		Evidence []string `json:"evidence_dia_ids"`
	} `json:"extra"`
}
type conversation struct {
	ID        string     `json:"conversation_id"`
	Sessions  []session  `json:"sessions"`
	Questions []question `json:"questions"`
}
type rawConversation struct {
	ID           string                     `json:"sample_id"`
	Conversation map[string]json.RawMessage `json:"conversation"`
}
type cachedQuery struct{ vec []float64 }

func (cachedQuery) ModelName() string                      { return nomic.ModelName }
func (cachedQuery) Backend() string                        { return search.EmbedderBackendEmbedded }
func (q cachedQuery) EmbedQuery(string) ([]float64, error) { return q.vec, nil }
func (cachedQuery) Close()                                 {}

type totals struct {
	Questions           int `json:"questions"`
	AllSupport          int `json:"all_supporting_turns_delivered"`
	SupportingDelivered int `json:"supporting_turns_delivered"`
	SupportingTotal     int `json:"supporting_turns_total"`
	ContentBytes        int `json:"content_bytes"`
	DeliveredTurns      int `json:"delivered_turn_slices"`
}

func digest(data []byte) string { h := sha256.Sum256(data); return hex.EncodeToString(h[:]) }
func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func run() error {
	input := flag.String("input", "scripts/industry-bench/datasets/data/locomo-conversations.jsonl", "normalized LoCoMo JSONL")
	source := flag.String("source", "scripts/industry-bench/datasets/data/locomo10.json", "raw source for exact dialogue-ID mapping")
	output := flag.String("out", "/tmp/rekal-passage-eval", "output directory; no source text is written")
	maxConversations := flag.Int("conversations", 1, "first N conversations (0=all)")
	questionLimit := flag.Int("questions", 0, "first N supported questions per conversation (0=all)")
	flag.Parse()
	if err := os.MkdirAll(*output, 0755); err != nil {
		return err
	}
	sourceBytes, err := os.ReadFile(*source)
	if err != nil {
		return err
	}
	var raw []rawConversation
	if err := json.Unmarshal(sourceBytes, &raw); err != nil {
		return err
	}
	rawByID := map[string]rawConversation{}
	for _, r := range raw {
		rawByID[r.ID] = r
	}
	f, err := os.Open(*input)
	if err != nil {
		return err
	}
	defer f.Close()
	var conversations []conversation
	dec := json.NewDecoder(f)
	for {
		var c conversation
		err := dec.Decode(&c)
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		conversations = append(conversations, c)
	}
	if *maxConversations > 0 && len(conversations) > *maxConversations {
		conversations = conversations[:*maxConversations]
	}
	model, err := nomic.NewEmbedder()
	if err != nil {
		return err
	}
	defer model.Close()
	modes := []string{"session-primary", "passage-primary", "passage-multiple"}
	aggregate := map[string]*totals{}
	for _, mode := range modes {
		aggregate[mode] = &totals{}
	}
	records, err := os.Create(filepath.Join(*output, "per_question.jsonl"))
	if err != nil {
		return err
	}
	defer records.Close()
	writer := json.NewEncoder(records)
	started := time.Now()
	for _, c := range conversations {
		if err := evaluate(c, rawByID[c.ID], model, *questionLimit, writer, aggregate, modes); err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "completed %s (%d sessions)\n", c.ID, len(c.Sessions))
	}
	report := map[string]interface{}{"dataset_sha256": digest(sourceBytes), "model": nomic.ModelName, "conversations": len(conversations), "weights": search.DefaultWeights(), "query_framings": 1, "session_limit": 5, "radius": 2, "max_content_bytes": 16000, "max_turn_slices": 80, "elapsed_seconds": time.Since(started).Seconds(), "answering_model": nil, "qa_accuracy": nil, "modes": aggregate, "scope": "Matched retrieval/evidence ablation; no reader or judge is run. Same query vectors, questions, weights and context limits. Session-primary reproduces the former turn-zero fallback for session-only hits."}
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(*output, "aggregate.json"), append(data, '\n'), 0644)
}

func evaluate(c conversation, raw rawConversation, model *nomic.Embedder, questionLimit int, writer *json.Encoder, aggregate map[string]*totals, modes []string) error {
	root, err := os.MkdirTemp("", "rekal-evidence-eval-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(root) //nolint:errcheck // isolated evaluation scratch store
	if err := os.MkdirAll(filepath.Join(root, ".rekal"), 0755); err != nil {
		return err
	}
	d, err := db.OpenIndex(root)
	if err != nil {
		return err
	}
	defer d.Close()
	if err := db.InitIndexSchema(d); err != nil {
		return err
	}
	if err := db.EnsureKnowledgeSchema(d); err != nil {
		return err
	}
	if err := db.LoadFTSExtension(d); err != nil {
		return err
	}
	diaByTurn := map[string]string{}
	lengthByDia := map[string]int{}
	for _, s := range c.Sessions {
		sid := c.ID + "_" + s.ID
		if _, err := d.Exec(`INSERT INTO session_facets(session_id,user_email,actor_type,captured_at,turn_count) VALUES ($1,'benchmark@example.com','human',$2,$3)`, sid, s.Date, len(s.Turns)); err != nil {
			return err
		}
		var sourceTurns []struct {
			DiaID string `json:"dia_id"`
		}
		if err := json.Unmarshal(raw.Conversation["session_"+strings.TrimPrefix(s.ID, "s")], &sourceTurns); err != nil {
			return err
		}
		if len(sourceTurns) != len(s.Turns) {
			return fmt.Errorf("source/normalized turn mismatch for %s", sid)
		}
		for i, t := range s.Turns {
			role := t.Role
			if role == "user" {
				role = "human"
			}
			if _, err := d.Exec(`INSERT INTO turns_ft(id,session_id,turn_index,role,content,ts) VALUES ($1,$2,$3,$4,$5,$6)`, fmt.Sprintf("%s_%d", sid, i), sid, i, role, t.Text, s.Date); err != nil {
				return err
			}
			diaByTurn[fmt.Sprintf("%s:%d", sid, i)] = sourceTurns[i].DiaID
			lengthByDia[sourceTurns[i].DiaID] = len(t.Text)
		}
	}
	if err := db.CreateFTSIndex(d); err != nil {
		return err
	}
	docs, err := db.QuerySessionContent(d)
	if err != nil {
		return err
	}
	whole := map[string][]float64{}
	for id, text := range docs {
		vec, e := model.EmbedDocument(text)
		if e != nil {
			return e
		}
		whole[id] = vec
	}
	if err := db.StoreEmbeddings(d, whole, nomic.ModelName); err != nil {
		return err
	}
	lsaModel, err := lsa.Build(docs, lsa.DefaultDimension)
	if err != nil {
		return err
	}
	if lsaModel != nil {
		if err := db.StoreEmbeddings(d, lsaModel.Vectors(), "lsa-v1"); err != nil {
			return err
		}
		if err := search.StoreLSAProjection(d, lsaModel); err != nil {
			return err
		}
	}
	if err := db.RefreshSessionPassages(d); err != nil {
		return err
	}
	passages, err := db.QueryPassageDocuments(d, nil)
	if err != nil {
		return err
	}
	passageVectors := map[string][]float64{}
	for id, text := range passages {
		vec, e := model.EmbedDocument(text)
		if e != nil {
			return e
		}
		passageVectors[id] = vec
	}
	queryVectors := map[string][]float64{}
	var questions []question
	for _, q := range c.Questions {
		if q.Category == "adversarial" || len(q.Extra.Evidence) == 0 {
			continue
		}
		if questionLimit > 0 && len(questions) >= questionLimit {
			break
		}
		vec, e := model.EmbedQuery(q.Text)
		if e != nil {
			return e
		}
		queryVectors[q.ID] = vec
		questions = append(questions, q)
	}
	for _, mode := range modes {
		if mode != "session-primary" {
			if err := db.StorePassageEmbeddings(d, passageVectors, nomic.ModelName); err != nil {
				return err
			}
		}
		for _, q := range questions {
			begin := time.Now()
			out, err := search.Run(d, search.Filters{Query: q.Text, TextQuery: true, Limit: 5, LimitExplicit: true}, root, search.DefaultWeights(), cachedQuery{queryVectors[q.ID]})
			if err != nil {
				return err
			}
			lookupMS := time.Since(begin).Seconds() * 1000
			var anchors []search.Anchor
			for _, r := range out.Results {
				switch mode {
				case "session-primary":
					idx := r.SnippetTurnIdx
					if idx < 0 {
						idx = 0
					}
					anchors = append(anchors, search.Anchor{SessionID: r.SessionID, StartTurn: idx, EndTurn: idx})
				case "passage-primary":
					if len(r.Anchors) > 0 {
						anchors = append(anchors, r.Anchors[0])
					}
				default:
					anchors = append(anchors, r.Anchors...)
				}
			}
			begin = time.Now()
			assembled, err := evidence.Fetch(d, evidence.Request{Anchors: anchors}, 2, 16000, 80)
			if err != nil {
				return err
			}
			fetchMS := time.Since(begin).Seconds() * 1000
			delivered := map[string][][2]int{}
			slices := 0
			for _, w := range assembled.Windows {
				for _, t := range w.Turns {
					key := diaByTurn[fmt.Sprintf("%s:%d", w.SessionID, t.Index)]
					if key != "" {
						delivered[key] = append(delivered[key], [2]int{t.StartByte, t.EndByte})
					}
					slices++
				}
			}
			full := map[string]bool{}
			for dia, parts := range delivered {
				covered := make([]bool, lengthByDia[dia])
				for _, p := range parts {
					for i := p[0]; i < p[1] && i < len(covered); i++ {
						covered[i] = true
					}
				}
				all := true
				for _, ok := range covered {
					if !ok {
						all = false
						break
					}
				}
				if all {
					full[dia] = true
				}
			}
			supportFound, unresolved := 0, 0
			for _, dia := range q.Extra.Evidence {
				if _, ok := lengthByDia[dia]; !ok {
					unresolved++
				}
				if full[dia] {
					supportFound++
				}
			}
			all := supportFound == len(q.Extra.Evidence)
			a := aggregate[mode]
			a.Questions++
			a.SupportingTotal += len(q.Extra.Evidence)
			a.SupportingDelivered += supportFound
			a.ContentBytes += assembled.ContentBytes
			a.DeliveredTurns += slices
			if all {
				a.AllSupport++
			}
			contextJSON, err := json.Marshal(assembled.Windows)
			if err != nil {
				return err
			}
			if err := writer.Encode(map[string]interface{}{"conversation_id": c.ID, "question_id": q.ID, "category": q.Category, "mode": mode, "support_ids": q.Extra.Evidence, "delivered_turn_ids": full, "unresolved_support_ids": unresolved, "all_support_delivered": all, "content_sha256": digest(contextJSON), "content_bytes": assembled.ContentBytes, "turn_slices": slices, "lookup_ms": lookupMS, "evidence_fetch_ms": fetchMS, "truncated": assembled.Truncated}); err != nil {
				return err
			}
		}
	}
	return nil
}
