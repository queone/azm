package utl

import "testing"

func TestDateHelpersAcceptNumericAndAbbreviatedMonths(t *testing.T) {
	if !ValidDate("2025-04-18", "2006-01-02") || !ValidDate("2025-apr-18", "2006-01-02") || ValidDate("18/04/2025", "2006-01-02") {
		t.Fatal("ValidDate() wrong")
	}
	got, err := ConvertDateFormat("2025-Apr-18", "2006-01-02", "02 Jan 2006")
	if err != nil || got != "18 Apr 2025" {
		t.Fatalf("ConvertDateFormat() = %q, %v", got, err)
	}
	epoch, err := DateStringToEpocInt64("1970-01-02", "2006-01-02")
	if err != nil || epoch != 86400 {
		t.Fatalf("DateStringToEpocInt64() = %d, %v", epoch, err)
	}
}

func TestIntegerStringConversions(t *testing.T) {
	n, err := StringToInt64("42")
	if err != nil || n != 42 {
		t.Fatalf("StringToInt64() = %d, %v", n, err)
	}
	if _, err := StringToInt64("x"); err == nil {
		t.Fatal("StringToInt64(\"x\") returned no error")
	}
	if got := Int64ToString(-7); got != "-7" {
		t.Fatalf("Int64ToString() = %q", got)
	}
	future := GetDateInDays("1")
	today := GetDateInDays("0")
	if !future.After(today) {
		t.Fatal("GetDateInDays(\"1\") is not after today")
	}
}
