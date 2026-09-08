package utl

import (
	"strconv"
	"strings"
	"time"
)

// Return absolute value of int value

// Returns absolute value of int64 value

// Converts string number to int64
func StringToInt64(s string) (int64, error) {
	i, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, err
	}
	return i, nil
}

// Converts int64 number to string
func Int64ToString(i int64) string {
	return strconv.FormatInt(i, 10)
}

// normalizeMonthAbbrev normalizes a 3-letter month string to "Jan"-style.
// Returns input unchanged if too short.
func normalizeMonthAbbrev(m string) string {
	if len(m) < 3 {
		return m
	}
	return strings.ToUpper(m[:1]) + strings.ToLower(m[1:3])
}

// parseFlexibleDate tries parsing with both YYYY-MM-DD and YYYY-MMM-DD layouts.
// Accepts case-insensitive three-letter months (e.g., Jan, JAN, jan).
func parseFlexibleDate(dateStr string) (time.Time, error) {
	layouts := []string{"2006-01-02", "2006-Jan-02"}
	var lastErr error

	for _, layout := range layouts {
		try := dateStr
		if layout == "2006-Jan-02" && len(try) >= 8 {
			year := try[0:4]
			month := normalizeMonthAbbrev(try[5:8])
			rest := try[8:]
			try = year + "-" + month + rest
		}
		t, err := time.Parse(layout, try)
		if err == nil {
			return t, nil
		}
		lastErr = err
	}
	return time.Time{}, lastErr
}

// Check if string is a valid date against one or more layouts.
// Accepts case-insensitive three-letter months for "2006-Jan-02".
func validDateFlex(dateString string, layouts ...string) bool {
	for _, layout := range layouts {
		try := dateString
		if layout == "2006-Jan-02" && len(try) >= 8 {
			year := try[0:4]
			month := normalizeMonthAbbrev(try[5:8])
			rest := try[8:]
			try = year + "-" + month + rest
		}
		if _, err := time.Parse(layout, try); err == nil {
			return true
		}
	}
	return false
}

// Check if string is a valid date in expectedFormat. Return true if so, false otherwise.
// See https://pkg.go.dev/time. Backward-compatible wrapper for validDateFlex.
// If expectedFormat is "2006-01-02", also allow "2006-Jan-02" in any casing.
func ValidDate(dateString, expectedFormat string) bool {
	if _, err := time.Parse(expectedFormat, dateString); err == nil {
		return true
	}
	if expectedFormat == "2006-01-02" {
		return validDateFlex(dateString, "2006-Jan-02")
	}
	return false
}

// Converts an epoch timestamp in int64 format to a time.Time object.
func epocInt64ToTime(epocInt int64) time.Time {
	return time.Unix(epocInt, 0)
}

// Converts an epoch timestamp in string format to a time.Time object.
// Returns a time.Time object and an error if the conversion fails.

// Converts dateString from source format to destination format.
// Returns date string in destination format and an error if the conversion fails.
func ConvertDateFormat(dateString, srcFormat, dstFormat string) (string, error) {
	// If source format is strict, use flexible parsing
	var t time.Time
	var err error
	if srcFormat == "2006-01-02" {
		t, err = parseFlexibleDate(dateString)
	} else {
		t, err = time.Parse(srcFormat, dateString)
	}
	if err != nil {
		return "", err
	}
	return t.Format(dstFormat), nil
}

// Convert dateString, given in dateFormat, to Unix Epoch seconds int64.
func DateStringToEpocInt64(dateString, dateFormat string) (int64, error) {
	var t time.Time
	var err error
	if dateFormat == "2006-01-02" {
		t, err = parseFlexibleDate(dateString)
	} else {
		t, err = time.Parse(dateFormat, dateString)
	}
	if err != nil {
		return 0, err
	}
	return t.Unix(), nil
}

// Print yyyy-mm-dd date for given number of +/- days in future or past
func GetDateInDays(days string) time.Time {
	now := time.Now().Unix()
	daysInt64, err := StringToInt64(days)
	if err != nil {
		panic(err.Error())
	}
	now += (daysInt64 * 86400) // 86400 seconds in a day
	return epocInt64ToTime(now)
}

// Returns true if given year is a leap year. False otherwise.

// Calculate and return number of +/- days from NOW to date given
// Note: Calculations are all in UTC time. And it takes leap year into account.

// Adjust for leap years

// Print number of days, also in years and days

// Return number of days between 2 dates
