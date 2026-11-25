package postgres

import (
	"reflect"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestBuildCustomerFilterEmptyAndWhitespace(t *testing.T) {
	for _, query := range []string{"", "   \t\n  "} {
		where, args := buildCustomerFilter(query)
		if where != "" || args != nil {
			t.Fatalf("query %q: got (%q, %v), want (\"\", nil)", query, where, args)
		}
	}
}

func TestBuildCustomerFilterSingleAndMultiWord(t *testing.T) {
	where, args := buildCustomerFilter("Ana")
	wantWhere := `WHERE name ILIKE '%' || $1 || '%' ESCAPE '\'`
	if where != wantWhere || !reflect.DeepEqual(args, []any{"Ana"}) {
		t.Fatalf("single word: got (%q, %v), want (%q, [Ana])", where, args, wantWhere)
	}

	where, args = buildCustomerFilter("Ana Perez")
	wantMulti := `WHERE name ILIKE '%' || $1 || '%' ESCAPE '\' AND name ILIKE '%' || $2 || '%' ESCAPE '\'`
	if where != wantMulti || !reflect.DeepEqual(args, []any{"Ana", "Perez"}) {
		t.Fatalf("multi word: got (%q, %v), want (%q, [Ana Perez])", where, args, wantMulti)
	}
}

func TestBuildCustomerFilterMixedAndPlusToken(t *testing.T) {
	where, args := buildCustomerFilter("Ana 300")
	wantMixed := `WHERE name ILIKE '%' || $1 || '%' ESCAPE '\' AND ` +
		`(name ILIKE '%' || $2 || '%' ESCAPE '\' OR regexp_replace(phone, '[^0-9]', '', 'g') LIKE '%' || $3 || '%' ESCAPE '\')`
	if where != wantMixed || !reflect.DeepEqual(args, []any{"Ana", "300", "300"}) {
		t.Fatalf("mixed: got (%q, %v), want (%q, [Ana 300 300])", where, args, wantMixed)
	}

	where, args = buildCustomerFilter("+ 57")
	wantPlus := `WHERE (name ILIKE '%' || $1 || '%' ESCAPE '\' OR phone LIKE '+%') AND ` +
		`(name ILIKE '%' || $2 || '%' ESCAPE '\' OR regexp_replace(phone, '[^0-9]', '', 'g') LIKE '%' || $3 || '%' ESCAPE '\')`
	if where != wantPlus || !reflect.DeepEqual(args, []any{"+", "57", "57"}) {
		t.Fatalf("plus token: got (%q, %v), want (%q, [+ 57 57])", where, args, wantPlus)
	}
}

func TestBuildCustomerFilterEscapingAndNonASCIIDigits(t *testing.T) {
	where, args := buildCustomerFilter(`50%_\`)
	wantArgs := []any{`50\%\_\\`, "50"}
	if !strings.Contains(where, `ESCAPE '\'`) || !reflect.DeepEqual(args, wantArgs) {
		t.Fatalf("wildcards: got (%q, %v), want ESCAPE '\\' and %v", where, args, wantArgs)
	}

	where, args = buildCustomerFilter("٣٠٠")
	wantNonASCII := `WHERE name ILIKE '%' || $1 || '%' ESCAPE '\'`
	if where != wantNonASCII || strings.Contains(where, "regexp_replace") || !reflect.DeepEqual(args, []any{"٣٠٠"}) {
		t.Fatalf("non-ASCII digits: got (%q, %v), want (%q, [٣٠٠])", where, args, wantNonASCII)
	}
}

func TestBuildCustomerFilterTokenCap(t *testing.T) {
	where, args := buildCustomerFilter("a b c d e f g h i j k l")
	if count := strings.Count(where, "name ILIKE"); count != maxSearchTokens {
		t.Fatalf("token clause count = %d, want %d (where=%q)", count, maxSearchTokens, where)
	}
	wantArgs := []any{"a", "b", "c", "d", "e", "f", "g", "h"}
	if !reflect.DeepEqual(args, wantArgs) {
		t.Fatalf("args = %v, want %v", args, wantArgs)
	}
}

func TestBuildCustomerFilterRuneTruncationUTF8(t *testing.T) {
	prefix := strings.Repeat("a", maxSearchQueryRunes-2)
	query := prefix + " 🔥 tail"
	truncated := truncateRunes(query, maxSearchQueryRunes)
	if !utf8.ValidString(truncated) || utf8.RuneCountInString(truncated) != maxSearchQueryRunes {
		t.Fatalf("truncated invalid or wrong rune count: valid=%v runes=%d", utf8.ValidString(truncated), utf8.RuneCountInString(truncated))
	}
	_, args := buildCustomerFilter(query)
	if !reflect.DeepEqual(args, []any{prefix, "🔥"}) {
		t.Fatalf("args = %v, want [%s 🔥]", args, prefix)
	}

	boundary := strings.Repeat("a", maxSearchQueryRunes-1) + "🔥tail"
	cut := truncateRunes(boundary, maxSearchQueryRunes)
	if !utf8.ValidString(cut) || utf8.RuneCountInString(cut) != maxSearchQueryRunes || !strings.HasSuffix(cut, "🔥") {
		t.Fatalf("boundary cut invalid: valid=%v runes=%d cut=%q", utf8.ValidString(cut), utf8.RuneCountInString(cut), cut)
	}
}
