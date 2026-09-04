package auth

import (
	"strings"
	"testing"
)

func TestPasswordRoundTrip(t *testing.T) {
	hash, err := HashPassword("contrasena-segura-123")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}

	if !strings.HasPrefix(hash, "$argon2id$v=19$m=19456,t=2,p=1$") {
		t.Fatalf("hash has unexpected prefix: %s", hash)
	}

	ok, err := VerifyPassword("contrasena-segura-123", hash)
	if err != nil || !ok {
		t.Fatalf("correct password rejected: ok=%v err=%v", ok, err)
	}

	ok, err = VerifyPassword("contrasena-mala", hash)
	if err != nil || ok {
		t.Fatalf("wrong password accepted: ok=%v err=%v", ok, err)
	}
}

func TestPasswordHashesAreSalted(t *testing.T) {
	first, err := HashPassword("misma-clave")
	if err != nil {
		t.Fatal(err)
	}

	second, err := HashPassword("misma-clave")
	if err != nil {
		t.Fatal(err)
	}

	if first == second {
		t.Fatal("two hashes for the same password are identical")
	}
}

func TestVerifyPasswordRejectsMalformedHashes(t *testing.T) {
	for _, bad := range []string{
		"",
		"plaintext",
		"$argon2i$v=19$m=19456,t=2,p=1$c2FsdA$aGFzaA",
		"$argon2id$v=18$m=19456,t=2,p=1$c2FsdA$aGFzaA",
		"$argon2id$v=19$m=19456,t=2,p=1$!!badb64$aGFzaA",
	} {
		if ok, err := VerifyPassword("x", bad); ok || err == nil {
			t.Fatalf("malformed hash %q: ok=%v err=%v", bad, ok, err)
		}
	}
}

func TestGenerateTempPassword(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 10; i++ {
		password, err := GenerateTempPassword()
		if err != nil {
			t.Fatalf("GenerateTempPassword: %v", err)
		}

		if len(password) != 16 {
			t.Fatalf("len = %d, want 16", len(password))
		}

		for _, char := range password {
			if !strings.ContainsRune(tempPassAlphabet, char) {
				t.Fatalf("password %q contains invalid char %q", password, char)
			}
		}

		if seen[password] {
			t.Fatalf("duplicate temp password %q", password)
		}
		seen[password] = true
	}
}
