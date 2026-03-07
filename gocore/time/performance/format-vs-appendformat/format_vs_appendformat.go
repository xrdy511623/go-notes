package formatvsappendformat

import "time"

// FormatRFC3339 formats a time using Format (allocates a new string).
func FormatRFC3339(t time.Time) string {
	return t.Format(time.RFC3339)
}

// AppendFormatRFC3339 formats a time using AppendFormat (appends to existing buffer).
func AppendFormatRFC3339(buf []byte, t time.Time) []byte {
	return t.AppendFormat(buf, time.RFC3339)
}

// FormatCustom formats a time using a custom layout.
func FormatCustom(t time.Time) string {
	return t.Format("2006-01-02 15:04:05.000")
}

// AppendFormatCustom formats a time using AppendFormat with a custom layout.
func AppendFormatCustom(buf []byte, t time.Time) []byte {
	return t.AppendFormat(buf, "2006-01-02 15:04:05.000")
}
