package user

import "testing"

func TestHashAndVerifyPassword(t *testing.T) {
	hash, err := HashPassword("minhaSenhaSegura123")
	if err != nil {
		t.Fatalf("hash não deveria falhar: %v", err)
	}
	if hash == "minhaSenhaSegura123" {
		t.Fatal("hash não deveria ser igual à senha em texto puro")
	}

	ok, err := VerifyPassword(hash, "minhaSenhaSegura123")
	if err != nil {
		t.Fatalf("verify não deveria falhar: %v", err)
	}
	if !ok {
		t.Fatal("senha correta deveria ser aceita")
	}
}

func TestVerifyPassword_WrongPassword(t *testing.T) {
	hash, _ := HashPassword("senhaCorreta123")
	ok, err := VerifyPassword(hash, "senhaErrada123")
	if err != nil {
		t.Fatalf("verify não deveria falhar: %v", err)
	}
	if ok {
		t.Fatal("senha incorreta não deveria ser aceita")
	}
}

func TestHashPassword_DifferentSaltsProduceDifferentHashes(t *testing.T) {
	h1, _ := HashPassword("mesmaSenha123")
	h2, _ := HashPassword("mesmaSenha123")
	if h1 == h2 {
		t.Fatal("hashes da mesma senha deveriam diferir devido ao salt aleatório")
	}
}

func TestValidateEmail(t *testing.T) {
	cases := map[string]bool{
		"user@example.com":     true,
		"user.name@sub.ex.com": true,
		"invalid":              false,
		"":                     false,
		"@example.com":         false,
		"user@":                false,
	}
	for email, want := range cases {
		err := ValidateEmail(email)
		got := err == nil
		if got != want {
			t.Errorf("ValidateEmail(%q) = %v, want %v", email, got, want)
		}
	}
}

func TestValidatePassword(t *testing.T) {
	if err := ValidatePassword("1234567"); err == nil {
		t.Error("senha com 7 caracteres deveria ser inválida")
	}
	if err := ValidatePassword("12345678"); err != nil {
		t.Error("senha com 8 caracteres deveria ser válida")
	}
}

func TestNormalizeEmail(t *testing.T) {
	if got := NormalizeEmail("  User@Example.COM "); got != "user@example.com" {
		t.Errorf("NormalizeEmail retornou %q", got)
	}
}
