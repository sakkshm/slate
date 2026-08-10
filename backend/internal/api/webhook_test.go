package api

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

func TestVerifyWebhookSignature(t *testing.T) {
	secret := "slate-deploy"
	body := []byte(`{"ref":"refs/heads/main"}`)

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	sig := "sha256=" + hex.EncodeToString(mac.Sum(nil))

	if !verifyWebhookSignature(secret, body, sig) {
		t.Fatal("valid signature should pass")
	}

	if verifyWebhookSignature("wrong-secret", body, sig) {
		t.Fatal("signature with wrong secret should fail")
	}

	tampered := []byte(`{"ref":"refs/heads/dev"}`)
	if verifyWebhookSignature(secret, tampered, sig) {
		t.Fatal("tampered body should fail")
	}

	if verifyWebhookSignature(secret, body, "sha1=abc123") {
		t.Fatal("sha1 signature should be rejected")
	}

	if verifyWebhookSignature(secret, body, "invalid") {
		t.Fatal("malformed signature should be rejected")
	}

	if verifyWebhookSignature(secret, body, "") {
		t.Fatal("empty signature should be rejected")
	}
}

func TestParseRepoOwnerName(t *testing.T) {
	cases := []struct {
		url   string
		owner string
		repo  string
		ok    bool
	}{
		{"https://github.com/owner/repo.git", "owner", "repo", true},
		{"https://github.com/owner/repo", "owner", "repo", true},
		{"https://github.com/owner/repo/", "owner", "repo", true},
		{"https://github.com/owner/repo/extra/path.git", "extra", "path", true},
		{"not-a-url", "", "", false},
		{"", "", "", false},
	}

	for _, tc := range cases {
		owner, repo, ok := parseRepoOwnerName(tc.url)
		if ok != tc.ok || (ok && (owner != tc.owner || repo != tc.repo)) {
			t.Fatalf("parseRepoOwnerName(%q) = (%q, %q, %v), want (%q, %q, %v)",
				tc.url, owner, repo, ok, tc.owner, tc.repo, tc.ok)
		}
	}
}
