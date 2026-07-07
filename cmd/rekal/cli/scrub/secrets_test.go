package scrub

import (
	"strings"
	"testing"
)

func TestRedactJWT(t *testing.T) {
	t.Parallel()
	input := "token: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.dozjgNryP4J3jVmNHl0w5N_XgL0n3I9PlFUP0THsR8U"
	got := RedactText(input)
	if strings.Contains(got, "eyJ") {
		t.Errorf("JWT not redacted: %s", got)
	}
	if !strings.Contains(got, redacted) {
		t.Errorf("expected %s in output: %s", redacted, got)
	}
}

func TestRedactAnthropicKey(t *testing.T) {
	t.Parallel()
	input := "key=sk-ant-api03-abcdefghijklmnopqrstuvwxyz"
	got := RedactText(input)
	if strings.Contains(got, "sk-ant-") {
		t.Errorf("Anthropic key not redacted: %s", got)
	}
}

func TestRedactOpenAIKey(t *testing.T) {
	t.Parallel()
	input := "OPENAI_API_KEY=sk-proj1234567890abcdefghij"
	got := RedactText(input)
	if strings.Contains(got, "sk-proj") {
		t.Errorf("OpenAI key not redacted: %s", got)
	}
}

func TestRedactGitHubToken(t *testing.T) {
	t.Parallel()
	for _, prefix := range []string{"ghp_", "gho_", "ghu_", "ghs_", "ghr_"} {
		tok := prefix + "ABCDEFGHIJKLMNOPQRSTUVWXYZab"
		got := RedactText("token: " + tok)
		if strings.Contains(got, prefix) {
			t.Errorf("GitHub token %s not redacted: %s", prefix, got)
		}
	}
}

func TestRedactGitHubFinegrained(t *testing.T) {
	t.Parallel()
	input := "github_pat_ABCDEFGHIJKLMNOPQRSTUVWXYZ1234"
	got := RedactText(input)
	if strings.Contains(got, "github_pat_") {
		t.Errorf("GitHub fine-grained token not redacted: %s", got)
	}
}

func TestRedactAWSAccessKey(t *testing.T) {
	t.Parallel()
	input := "AKIAIOSFODNN7EXAMPLE"
	got := RedactText(input)
	if strings.Contains(got, "AKIA") {
		t.Errorf("AWS access key not redacted: %s", got)
	}
}

func TestRedactSlackToken(t *testing.T) {
	t.Parallel()
	input := "xoxb-1234567890-abcdefghij"
	got := RedactText(input)
	if strings.Contains(got, "xoxb-") {
		t.Errorf("Slack token not redacted: %s", got)
	}
}

func TestRedactNPMToken(t *testing.T) {
	t.Parallel()
	input := "npm_aBcDeFgHiJkLmNoPqRsTuVwXyZ"
	got := RedactText(input)
	if strings.Contains(got, "npm_") {
		t.Errorf("NPM token not redacted: %s", got)
	}
}

func TestRedactPyPIToken(t *testing.T) {
	t.Parallel()
	input := "pypi-AgEIcHlwaS5vcmcCJGY4ZTU2"
	got := RedactText(input)
	if strings.Contains(got, "pypi-") {
		t.Errorf("PyPI token not redacted: %s", got)
	}
}

func TestRedactPrivateKey(t *testing.T) {
	t.Parallel()
	input := "-----BEGIN RSA PRIVATE KEY-----\nMIIEpAIBAAK..."
	got := RedactText(input)
	if strings.Contains(got, "BEGIN RSA PRIVATE KEY") {
		t.Errorf("private key header not redacted: %s", got)
	}
}

func TestRedactBearerToken(t *testing.T) {
	t.Parallel()
	input := "Authorization: Bearer eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9abcdef"
	got := RedactText(input)
	if strings.Contains(got, "Bearer eyJ") {
		t.Errorf("bearer token not redacted: %s", got)
	}
}

func TestRedactCLITokenFlag(t *testing.T) {
	t.Parallel()
	input := "curl --token=mysecrettoken123"
	got := RedactText(input)
	if strings.Contains(got, "mysecrettoken") {
		t.Errorf("CLI token flag not redacted: %s", got)
	}
}

func TestRedactDBConnectionString(t *testing.T) {
	t.Parallel()
	input := "postgres://admin:s3cret@db.example.com:5432/mydb"
	got := RedactText(input)
	if strings.Contains(got, "postgres://") {
		t.Errorf("DB connection string not redacted: %s", got)
	}
}

func TestRedactEnvSecret(t *testing.T) {
	t.Parallel()
	input := "export API_KEY=abcdef1234567890"
	got := RedactText(input)
	if strings.Contains(got, "abcdef1234567890") {
		t.Errorf("env secret not redacted: %s", got)
	}
}

func TestRedactGenericSecretAssign(t *testing.T) {
	t.Parallel()
	input := `password: "SuperSecretPass1234!"`
	got := RedactText(input)
	if strings.Contains(got, "SuperSecretPass") {
		t.Errorf("generic secret not redacted: %s", got)
	}
}

func TestRedactURLSecretParam(t *testing.T) {
	t.Parallel()
	input := "https://api.example.com/v1?token=abc123def456ghi"
	got := RedactText(input)
	if strings.Contains(got, "abc123def456") {
		t.Errorf("URL secret param not redacted: %s", got)
	}
}

func TestRedactHFToken(t *testing.T) {
	t.Parallel()
	input := "hf_aBcDeFgHiJkLmNoPqRsTuVwXyZ"
	got := RedactText(input)
	if strings.Contains(got, "hf_") {
		t.Errorf("HF token not redacted: %s", got)
	}
}

// --- Allowlist tests ---

func TestAllowlistNoreplyEmail(t *testing.T) {
	t.Parallel()
	input := "Co-Authored-By: noreply@anthropic.com"
	got := RedactText(input)
	if !strings.Contains(got, "noreply@anthropic.com") {
		t.Errorf("noreply email should not be redacted: %s", got)
	}
}

func TestAllowlistExampleEmail(t *testing.T) {
	t.Parallel()
	input := "contact user@example.com for help"
	got := RedactText(input)
	if !strings.Contains(got, "user@example.com") {
		t.Errorf("example.com email should not be redacted: %s", got)
	}
}

func TestRedactRealEmail(t *testing.T) {
	t.Parallel()
	input := "send to alice@company.io"
	got := RedactText(input)
	if strings.Contains(got, "alice@company.io") {
		t.Errorf("real email should be redacted: %s", got)
	}
}

func TestAllowlistPrivateIP(t *testing.T) {
	t.Parallel()
	for _, ip := range []string{"192.168.1.1", "10.0.0.1", "127.0.0.1", "172.16.0.1"} {
		got := RedactText("connect to " + ip)
		if !strings.Contains(got, ip) {
			t.Errorf("private IP %s should not be redacted: %s", ip, got)
		}
	}
}

func TestRedactPublicIP(t *testing.T) {
	t.Parallel()
	input := "server at 8.8.8.8"
	got := RedactText(input)
	if strings.Contains(got, "8.8.8.8") {
		t.Errorf("public IP should be redacted: %s", got)
	}
}

// --- Entropy tests ---

func TestShannonEntropy(t *testing.T) {
	t.Parallel()
	// Low entropy: repeated chars
	low := shannonEntropy("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	if low > 1.0 {
		t.Errorf("expected low entropy for repeated chars, got %f", low)
	}
	// High entropy: random-looking
	high := shannonEntropy("aB3$kL9!mN2@pQ5#rT8&vX1*yZ4^cF7+")
	if high < 4.0 {
		t.Errorf("expected high entropy, got %f", high)
	}
}

func TestHighEntropyAllowHexHash(t *testing.T) {
	t.Parallel()
	// A 40-char hex string (git SHA) should be allowed
	input := "commit abc123def456789012345678901234567890ab done"
	got := RedactText(input)
	if strings.Contains(got, redacted) {
		t.Errorf("hex hash should not be redacted: %s", got)
	}
}

func TestHighEntropyRedactsRandom(t *testing.T) {
	t.Parallel()
	// A 40+ char high-entropy non-hex string should be redacted
	input := "secret: aB3kL9mN2pQ5rT8vX1yZ4cF7dG0hJ6wE2qR9sU5tW8x"
	got := RedactText(input)
	if !strings.Contains(got, redacted) {
		t.Errorf("high-entropy string should be redacted: %s", got)
	}
}

func TestIsHexString(t *testing.T) {
	t.Parallel()
	cases := []struct {
		s    string
		want bool
	}{
		{"deadbeef0123456789abcdefABCDEF", true},
		{"e7b3a91f2c4d5e6f7890abcdef1234567890abcd", true}, // git SHA
		{"deadbeefz", false},                               // one non-hex rune
		{"", true},                                         // vacuously hex
	}
	for _, tc := range cases {
		if got := isHexString(tc.s); got != tc.want {
			t.Errorf("isHexString(%q) = %v, want %v", tc.s, got, tc.want)
		}
	}
}

func TestIsAllowedHighEntropy(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, s string
		want    bool
	}{
		{"git sha allowed", "e7b3a91f2c4d5e6f7890abcdef1234567890abcd", true},
		{"file path allowed", "/usr/local/lib/libgomp/x86_64-linux-gnu-12345", true},
		{"random base64 not allowed", "qZx9Kf3mWv8Rt2Yp6Ln4Jh0Gd5Bs7Cn1", false},
	}
	for _, tc := range cases {
		if got := isAllowedHighEntropy(tc.s); got != tc.want {
			t.Errorf("%s: isAllowedHighEntropy(%q) = %v, want %v", tc.name, tc.s, got, tc.want)
		}
	}
}

// TestRedactText_GitSHAPreserved pins the allowlist behavior end to end:
// commit SHAs are load-bearing context for recall (checkpoints anchor to
// them) and must survive redaction, while real secrets in the same text go.
func TestRedactText_GitSHAPreserved(t *testing.T) {
	t.Parallel()
	sha := "e7b3a91f2c4d5e6f7890abcdef1234567890abcd"
	in := "fixed in commit " + sha + " using token ghp_abcdefghijklmnopqrstuvwxyz123456"
	out := RedactText(in)
	if !strings.Contains(out, sha) {
		t.Fatalf("git SHA was redacted from %q", out)
	}
	if strings.Contains(out, "ghp_") {
		t.Fatalf("github token survived: %q", out)
	}
}
