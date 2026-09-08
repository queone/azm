package utl

import "testing"

func TestStrReturnsOnlyStringValues(t *testing.T) {
	if got := Str("a"); got != "a" {
		t.Fatalf("Str(\"a\") = %q", got)
	}
	if got := Str(7); got != "" {
		t.Fatalf("Str(7) = %q, want empty", got)
	}
}

func TestMapAndSliceCastSafely(t *testing.T) {
	m := map[string]any{"k": "v"}
	if got := Map(m); got["k"] != "v" {
		t.Fatalf("Map() lost the key")
	}
	if Map("no") != nil || Slice("no") != nil {
		t.Fatalf("Map/Slice must return nil for other types")
	}
	if got := Slice([]any{1, 2}); len(got) != 2 {
		t.Fatalf("Slice() = %v", got)
	}
}

func TestBoolAndInt64Conversions(t *testing.T) {
	if !Bool(true) || !Bool("true") || Bool("nope") || Bool(0) {
		t.Fatal("Bool() conversions wrong")
	}
	if Int64(3) != 3 || Int64(int64(4)) != 4 || Int64(5.9) != 5 || Int64("6") != 6 || Int64("x") != 0 {
		t.Fatal("Int64() conversions wrong")
	}
}

func TestPaddingHelpersRespectWidth(t *testing.T) {
	if got := PreSpc("ab", 5); got != "   ab" {
		t.Fatalf("PreSpc() = %q", got)
	}
	if got := PostSpc("ab", 5); got != "ab   " {
		t.Fatalf("PostSpc() = %q", got)
	}
	if got := PadSpaces(5, 2); got != "   " {
		t.Fatalf("PadSpaces() = %q", got)
	}
	if got := PadSpaces(2, 5); got != "" {
		t.Fatalf("PadSpaces() with negative padding = %q", got)
	}
}

func TestCommafyInsertsThousandSeparators(t *testing.T) {
	if got := Commafy(1234567); got != "1,234,567" {
		t.Fatalf("Commafy(int) = %q", got)
	}
	if got := Commafy(1234.5); got != "1,234.5" {
		t.Fatalf("Commafy(float) = %q", got)
	}
	if got := Int2StrWithCommas(int64(999)); got != "999" {
		t.Fatalf("Int2StrWithCommas() = %q", got)
	}
}

func TestStringHelpers(t *testing.T) {
	if !SubString("Hello World", "world") || SubString("abc", "z") {
		t.Fatal("SubString() case-insensitive match wrong")
	}
	if got := LastElemByDot("a.b.c"); got != "c" {
		t.Fatalf("LastElemByDot() = %q", got)
	}
	if got := LastElemByDot("nodot"); got != "" {
		t.Fatalf("LastElemByDot() without dot = %q", got)
	}
	if got := StrSingleQuote("*star"); got != "'*star'" {
		t.Fatalf("StrSingleQuote() = %q", got)
	}
	if got := ToStr(3.5); got != "3.5" {
		t.Fatalf("ToStr() = %q", got)
	}
	if !ValidUuid("c44154ad-6b37-4972-8067-0ef1068079b2") || ValidUuid("nope") {
		t.Fatal("ValidUuid() wrong")
	}
	if got := len(GenerateRandomString(12)); got != 12 {
		t.Fatalf("GenerateRandomString(12) length = %d", got)
	}
	keys := SortMapStringKeys(map[string]string{"b": "", "a": ""})
	if len(keys) != 2 || keys[0] != "a" {
		t.Fatalf("SortMapStringKeys() = %v", keys)
	}
}

func TestStringSetTracksMembership(t *testing.T) {
	s := StringSet{}
	s.Add("x")
	if !s.Exists("x") || s.Exists("y") || s.Size() != 1 {
		t.Fatal("StringSet membership wrong")
	}
	s.Remove("x")
	if s.Size() != 0 {
		t.Fatal("StringSet Remove() failed")
	}
}
