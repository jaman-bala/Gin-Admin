package utils

import "strings"

// NormalizePhone strips spaces and dashes, then ensures a leading "+".
// "996500500500" → "+996500500500", "+996 500 500 500" → "+996500500500"
func NormalizePhone(phone string) string {
	phone = strings.ReplaceAll(phone, " ", "")
	phone = strings.ReplaceAll(phone, "-", "")
	if phone != "" && !strings.HasPrefix(phone, "+") {
		phone = "+" + phone
	}
	return phone
}
