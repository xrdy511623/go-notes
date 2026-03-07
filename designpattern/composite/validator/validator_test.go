package validator

import (
	"fmt"
	"testing"
)

func TestNonEmpty(t *testing.T) {
	v := NonEmpty()
	if err := v.Validate("hello"); err != nil {
		t.Fatalf("NonEmpty rejected non-empty: %v", err)
	}
	if err := v.Validate(""); err == nil {
		t.Fatal("NonEmpty accepted empty string")
	}
}

func TestMinLength(t *testing.T) {
	v := MinLength(3)
	if err := v.Validate("abc"); err != nil {
		t.Fatalf("MinLength(3) rejected 3-char: %v", err)
	}
	if err := v.Validate("ab"); err == nil {
		t.Fatal("MinLength(3) accepted 2-char")
	}
}

func TestMaxLength(t *testing.T) {
	v := MaxLength(5)
	if err := v.Validate("hello"); err != nil {
		t.Fatalf("MaxLength(5) rejected 5-char: %v", err)
	}
	if err := v.Validate("toolong"); err == nil {
		t.Fatal("MaxLength(5) accepted 7-char")
	}
}

func TestMatches(t *testing.T) {
	v := Matches(`^[a-z]+$`)
	if err := v.Validate("hello"); err != nil {
		t.Fatalf("Matches rejected valid: %v", err)
	}
	if err := v.Validate("Hello123"); err == nil {
		t.Fatal("Matches accepted invalid")
	}
}

func TestAnd_AllPass(t *testing.T) {
	v := And(NonEmpty(), MinLength(3), MaxLength(10))
	if err := v.Validate("hello"); err != nil {
		t.Fatalf("And all-pass: %v", err)
	}
}

func TestAnd_ShortCircuit(t *testing.T) {
	v := And(NonEmpty(), MinLength(5), MaxLength(3))
	err := v.Validate("hi")
	if err == nil {
		t.Fatal("And should fail")
	}
	// Should fail on MinLength first (length 2 < 5), not MaxLength
	if err.Error() != "length 2 < minimum 5" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAnd_Empty(t *testing.T) {
	v := And()
	if err := v.Validate("anything"); err != nil {
		t.Fatalf("And() with no validators should pass: %v", err)
	}
}

func TestOr_AnyPass(t *testing.T) {
	v := Or(MinLength(100), Matches(`^go`), MaxLength(2))
	if err := v.Validate("golang"); err != nil {
		t.Fatalf("Or should pass on Matches: %v", err)
	}
}

func TestOr_AllFail(t *testing.T) {
	v := Or(MinLength(100), Matches(`^zzz`))
	if err := v.Validate("hello"); err == nil {
		t.Fatal("Or should fail when all children fail")
	}
}

func TestNot(t *testing.T) {
	// Must NOT contain digits
	v := Not(Matches(`\d`), "must not contain digits")
	if err := v.Validate("hello"); err != nil {
		t.Fatalf("Not should pass for non-digit: %v", err)
	}
	if err := v.Validate("hello123"); err == nil {
		t.Fatal("Not should fail for digit-containing string")
	}
}

func TestComposite_NestedTree(t *testing.T) {
	// username: nonEmpty AND (length 3..20) AND alphanumeric
	// password: nonEmpty AND (length 8..64) AND (has letter AND has digit)
	usernameRule := And(
		NonEmpty(),
		MinLength(3),
		MaxLength(20),
		Matches(`^[a-zA-Z0-9]+$`),
	)
	passwordRule := And(
		NonEmpty(),
		MinLength(8),
		MaxLength(64),
		And(Matches(`[a-zA-Z]`), Matches(`[0-9]`)),
	)

	tests := []struct {
		name  string
		rule  Validator
		value string
		ok    bool
	}{
		{"username: valid", usernameRule, "alice42", true},
		{"username: too short", usernameRule, "ab", false},
		{"username: has underscore", usernameRule, "alice_bob", false},
		{"password: valid", passwordRule, "secret42", true},
		{"password: no digit", passwordRule, "secretpass", false},
		{"password: too short", passwordRule, "s4", false},
	}
	for _, tt := range tests {
		err := tt.rule.Validate(tt.value)
		if tt.ok && err != nil {
			t.Errorf("%s: expected pass, got %v", tt.name, err)
		}
		if !tt.ok && err == nil {
			t.Errorf("%s: expected fail", tt.name)
		}
	}
}

func TestValidatorFunc_Adapter(t *testing.T) {
	v := ValidatorFunc(func(s string) error {
		if s == "bad" {
			return fmt.Errorf("bad value")
		}
		return nil
	})
	if err := v.Validate("good"); err != nil {
		t.Fatal(err)
	}
	if err := v.Validate("bad"); err == nil {
		t.Fatal("expected error")
	}
}
