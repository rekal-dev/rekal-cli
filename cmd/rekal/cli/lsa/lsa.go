package lsa

import (
	"math"
	"sort"
	"strings"
	"unicode"

	"gonum.org/v1/gonum/mat"
)

const (
	// DefaultDimension is the default SVD truncation rank.
	DefaultDimension = 128
	// minTermFreq is the minimum number of sessions a term must appear in.
	minTermFreq = 2
)

// Model holds the trained LSA components.
type Model struct {
	// Vocabulary maps term → column index in the term-document matrix.
	Vocabulary map[string]int
	// IDF weights per term (indexed by column).
	IDF []float64
	// Uk is the truncated left singular vectors (terms × k).
	Uk *mat.Dense
	// Sk is the truncated singular values (k diagonal).
	Sk []float64
	// Vk is the truncated right singular vectors (sessions × k).
	Vk *mat.Dense
	// SessionIDs maps row index in Vk → session_id.
	SessionIDs []string
	// Dim is the actual dimensionality used (may be < DefaultDimension).
	Dim int
}

// Build constructs an LSA model from session_id → concatenated content.
// Returns nil model if there are too few sessions or terms.
func Build(sessions map[string]string, dim int) (*Model, error) {
	if len(sessions) < 2 {
		return nil, nil
	}
	if dim <= 0 {
		dim = DefaultDimension
	}

	// Stable ordering of sessions.
	sessionIDs := make([]string, 0, len(sessions))
	for id := range sessions {
		sessionIDs = append(sessionIDs, id)
	}
	sort.Strings(sessionIDs)

	// Tokenize all sessions, build document frequency.
	docTerms := make([]map[string]float64, len(sessionIDs)) // tf per doc
	df := make(map[string]int)                              // document frequency

	for i, id := range sessionIDs {
		tokens := Tokenize(sessions[id])
		tf := make(map[string]float64)
		for _, tok := range tokens {
			tf[tok]++
		}
		docTerms[i] = tf
		for term := range tf {
			df[term]++
		}
	}

	// Build vocabulary: terms appearing in >= minTermFreq sessions.
	vocab := make(map[string]int)
	var terms []string
	for term, freq := range df {
		if freq >= minTermFreq {
			terms = append(terms, term)
		}
	}
	sort.Strings(terms)
	for i, term := range terms {
		vocab[term] = i
	}

	if len(vocab) < 2 {
		return nil, nil
	}

	nDocs := len(sessionIDs)
	nTerms := len(vocab)

	// Cap dimension.
	actualDim := dim
	if actualDim > nTerms {
		actualDim = nTerms
	}
	if actualDim > nDocs {
		actualDim = nDocs
	}

	// Compute IDF.
	idf := make([]float64, nTerms)
	for term, col := range vocab {
		idf[col] = math.Log(float64(nDocs)/float64(df[term])) + 1.0
	}

	// Build TF-IDF matrix (terms × documents).
	data := make([]float64, nTerms*nDocs)
	for docIdx, tf := range docTerms {
		// Compute max tf for normalization.
		var maxTF float64
		for term := range tf {
			if _, ok := vocab[term]; ok && tf[term] > maxTF {
				maxTF = tf[term]
			}
		}
		if maxTF == 0 {
			continue
		}
		for term, count := range tf {
			col, ok := vocab[term]
			if !ok {
				continue
			}
			// Augmented TF * IDF.
			tfNorm := 0.5 + 0.5*(count/maxTF)
			data[col*nDocs+docIdx] = tfNorm * idf[col]
		}
	}

	A := mat.NewDense(nTerms, nDocs, data)

	// Truncated SVD.
	var svd mat.SVD
	if !svd.Factorize(A, mat.SVDThin) {
		return nil, nil
	}

	var uFull, vFull mat.Dense
	svd.UTo(&uFull)
	svd.VTo(&vFull)
	sValues := svd.Values(nil)

	// Truncate to actualDim.
	uRows, _ := uFull.Dims()
	_, vCols := vFull.Dims()
	_ = vCols

	uk := mat.NewDense(uRows, actualDim, nil)
	for i := 0; i < uRows; i++ {
		for j := 0; j < actualDim; j++ {
			uk.Set(i, j, uFull.At(i, j))
		}
	}

	vRows := nDocs
	vk := mat.NewDense(vRows, actualDim, nil)
	for i := 0; i < vRows; i++ {
		for j := 0; j < actualDim; j++ {
			vk.Set(i, j, vFull.At(i, j))
		}
	}

	sk := make([]float64, actualDim)
	copy(sk, sValues[:actualDim])

	return &Model{
		Vocabulary: vocab,
		IDF:        idf,
		Uk:         uk,
		Sk:         sk,
		Vk:         vk,
		SessionIDs: sessionIDs,
		Dim:        actualDim,
	}, nil
}

// Embed projects a query string into the LSA space, returning a k-dimensional vector.
func (m *Model) Embed(text string) []float64 {
	tokens := Tokenize(text)
	if len(tokens) == 0 {
		return make([]float64, m.Dim)
	}

	// Build query TF vector.
	tf := make(map[string]float64)
	for _, tok := range tokens {
		tf[tok]++
	}
	var maxTF float64
	for _, c := range tf {
		if c > maxTF {
			maxTF = c
		}
	}

	// Query vector in term space.
	q := make([]float64, len(m.Vocabulary))
	for term, count := range tf {
		col, ok := m.Vocabulary[term]
		if !ok {
			continue
		}
		tfNorm := 0.5 + 0.5*(count/maxTF)
		q[col] = tfNorm * m.IDF[col]
	}

	// Project: q_k = q^T * U_k * S_k^{-1}
	result := make([]float64, m.Dim)
	nTerms := len(m.Vocabulary)
	for j := 0; j < m.Dim; j++ {
		if m.Sk[j] == 0 {
			continue
		}
		var dot float64
		for i := 0; i < nTerms; i++ {
			dot += q[i] * m.Uk.At(i, j)
		}
		result[j] = dot / m.Sk[j]
	}

	return result
}

// Vectors returns session_id → embedding for bulk storage.
// Each embedding is the row of Vk scaled by Sk.
func (m *Model) Vectors() map[string][]float64 {
	result := make(map[string][]float64, len(m.SessionIDs))
	for i, id := range m.SessionIDs {
		vec := make([]float64, m.Dim)
		for j := 0; j < m.Dim; j++ {
			vec[j] = m.Vk.At(i, j) * m.Sk[j]
		}
		result[id] = vec
	}
	return result
}

// CosineSimilarity computes the cosine similarity between two vectors.
func CosineSimilarity(a, b []float64) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}
	var dot, normA, normB float64
	for i := range a {
		dot += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}
	if normA == 0 || normB == 0 {
		return 0
	}
	return dot / (math.Sqrt(normA) * math.Sqrt(normB))
}

// Tokenize lowercases, splits on non-alphanumeric, removes stopwords,
// and applies simple stemming.
func Tokenize(text string) []string {
	text = strings.ToLower(text)
	var tokens []string
	var current strings.Builder

	for _, r := range text {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			current.WriteRune(r)
		} else {
			if current.Len() > 0 {
				word := current.String()
				current.Reset()
				if len(word) >= 2 && !stopwords[word] {
					tokens = append(tokens, simpleStem(word))
				}
			}
		}
	}
	if current.Len() > 0 {
		word := current.String()
		if len(word) >= 2 && !stopwords[word] {
			tokens = append(tokens, simpleStem(word))
		}
	}

	return tokens
}

// simpleStem applies basic English suffix stripping.
func simpleStem(word string) string {
	// Very basic suffix removal — enough for LSA to group related terms.
	suffixes := []string{"tion", "sion", "ment", "ness", "able", "ible", "ful", "less", "ous", "ive", "ing", "ied", "ies", "ers", "est", "ely", "ed", "ly", "er", "es", "al", "en", "s"}
	for _, suffix := range suffixes {
		if len(word) > len(suffix)+3 && strings.HasSuffix(word, suffix) {
			return word[:len(word)-len(suffix)]
		}
	}
	return word
}

var stopwords = map[string]bool{
	"the": true, "be": true, "to": true, "of": true, "and": true,
	"in": true, "that": true, "have": true, "it": true, "for": true,
	"not": true, "on": true, "with": true, "he": true, "as": true,
	"you": true, "do": true, "at": true, "this": true, "but": true,
	"his": true, "by": true, "from": true, "they": true, "we": true,
	"say": true, "her": true, "she": true, "or": true, "an": true,
	"will": true, "my": true, "one": true, "all": true, "would": true,
	"there": true, "their": true, "what": true, "so": true, "up": true,
	"out": true, "if": true, "about": true, "who": true, "get": true,
	"which": true, "go": true, "me": true, "when": true, "make": true,
	"can": true, "like": true, "no": true, "just": true, "him": true,
	"know": true, "take": true, "come": true, "could": true, "than": true,
	"look": true, "use": true, "into": true, "some": true, "them": true,
	"see": true, "other": true, "then": true, "now": true, "only": true,
	"its": true, "also": true, "after": true, "way": true, "our": true,
	"how": true, "more": true, "been": true, "was": true, "were": true,
	"are": true, "is": true, "am": true, "has": true, "had": true,
	"did": true, "does": true, "let": true, "may": true, "should": true,
	"must": true, "shall": true, "very": true, "much": true, "too": true,
}
