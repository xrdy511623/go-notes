package matcher

import "testing"

func TestMatcherFunc(t *testing.T) {
	isLong := MatcherFunc(func(s string) bool { return len(s) > 5 })
	if !isLong.Match("hello world") {
		t.Fatal("expected match for long string")
	}
	if isLong.Match("hi") {
		t.Fatal("expected no match for short string")
	}
}

func TestContains(t *testing.T) {
	m := Contains("Go")
	cases := []struct {
		input string
		want  bool
	}{
		{"Go 1.24", true},
		{"Learning Go", true},
		{"Rust lang", false},
		{"", false},
		{"go", false}, // case-sensitive
	}
	for _, tc := range cases {
		if got := m.Match(tc.input); got != tc.want {
			t.Errorf("Contains(\"Go\").Match(%q) = %v, want %v", tc.input, got, tc.want)
		}
	}
}

func TestHasPrefix(t *testing.T) {
	m := HasPrefix("http://")
	if !m.Match("http://example.com") {
		t.Error("expected match")
	}
	if m.Match("https://example.com") {
		t.Error("expected no match")
	}
}

func TestHasSuffix(t *testing.T) {
	m := HasSuffix(".go")
	if !m.Match("main.go") {
		t.Error("expected match")
	}
	if m.Match("main.py") {
		t.Error("expected no match")
	}
}

func TestMinLength(t *testing.T) {
	m := MinLength(5)
	if !m.Match("hello") {
		t.Error("expected match for len=5")
	}
	if m.Match("hi") {
		t.Error("expected no match for len=2")
	}
	if !m.Match("hello world") {
		t.Error("expected match for len=11")
	}
}

func TestRegexpMatcher(t *testing.T) {
	m, err := NewRegexpMatcher(`^\d{4}-\d{2}-\d{2}$`)
	if err != nil {
		t.Fatal(err)
	}
	if !m.Match("2024-01-15") {
		t.Error("expected date match")
	}
	if m.Match("not-a-date") {
		t.Error("expected no match")
	}
	if m.Match("2024-1-5") {
		t.Error("expected no match for short date")
	}
}

func TestRegexpMatcher_InvalidPattern(t *testing.T) {
	_, err := NewRegexpMatcher(`[invalid`)
	if err == nil {
		t.Fatal("expected error for invalid regex pattern")
	}
}

func TestAll(t *testing.T) {
	m := All(
		HasPrefix("Go"),
		Contains("1."),
		MinLength(5),
	)
	if !m.Match("Go 1.24") {
		t.Error("expected match")
	}
	if m.Match("Go 语言") {
		t.Error("expected no match — missing '1.'")
	}
	if m.Match("Rust 1.0") {
		t.Error("expected no match — wrong prefix")
	}
}

func TestAll_Empty(t *testing.T) {
	m := All()
	if !m.Match("anything") {
		t.Error("empty All should match everything (vacuous truth)")
	}
}

func TestAny(t *testing.T) {
	m := Any(
		HasSuffix(".go"),
		HasSuffix(".rs"),
	)
	if !m.Match("main.go") {
		t.Error("expected match for .go")
	}
	if !m.Match("main.rs") {
		t.Error("expected match for .rs")
	}
	if m.Match("main.py") {
		t.Error("expected no match for .py")
	}
}

func TestAny_Empty(t *testing.T) {
	m := Any()
	if m.Match("anything") {
		t.Error("empty Any should match nothing")
	}
}

func TestNot(t *testing.T) {
	m := Not(Contains("error"))
	if !m.Match("all good") {
		t.Error("expected match")
	}
	if m.Match("error occurred") {
		t.Error("expected no match")
	}
}

func TestComposition_FuncAndStruct(t *testing.T) {
	dateMatcher, _ := NewRegexpMatcher(`^\d{4}-\d{2}-\d{2}$`)
	m := All(
		dateMatcher,            // struct-based Matcher
		Not(HasPrefix("0000")), // function-based Matcher via MatcherFunc
		MinLength(10),          // function-based Matcher via MatcherFunc
	)
	if !m.Match("2024-01-15") {
		t.Error("expected match")
	}
	if m.Match("0000-00-00") {
		t.Error("expected no match — year 0000")
	}
	if m.Match("24-1-5") {
		t.Error("expected no match — too short / wrong format")
	}
}

func TestComposition_Nested(t *testing.T) {
	goFile := All(HasSuffix(".go"), Not(Contains("_test")))
	rsFile := HasSuffix(".rs")
	srcFile := Any(goFile, rsFile)

	cases := []struct {
		input string
		want  bool
	}{
		{"main.go", true},
		{"main_test.go", false},
		{"lib.rs", true},
		{"readme.md", false},
	}
	for _, tc := range cases {
		if got := srcFile.Match(tc.input); got != tc.want {
			t.Errorf("srcFile.Match(%q) = %v, want %v", tc.input, got, tc.want)
		}
	}
}
