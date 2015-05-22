package turtle

import (
	"strings"
	"unicode/utf8"
	"fmt"
)

var EscapeIriChars        = "<>\"{}|^`\\" +
	"\x00\x01\x02\x03\x04\x05\x06\x07\x08\x09\x0A\x0B\x0C\x0D\x0E\x0F" +
	"\x10\x11\x12\x13\x14\x15\x16\x17\x18\x19\x1A\x1B\x1C\x1D\x1E\x1F\x20"
var EscapeStringChars     = "\"'\r\n\\\x00"
var EscapeStringMoreChars = "\"'\r\n\\\t\b\f\x00"

func Escape(str, charsToEscape string) string {
	var res string
	var r rune
	var buf [16]byte
	for _, r = range str {
		if r == '\\' {
			res = res + `\\`
		} else if strings.ContainsRune(charsToEscape, r) {
			switch(r) {
				case 0x0009: res = res + `\t`; break
				case 0x0008: res = res + `\b`; break
				case 0x000A: res = res + `\n`; break
				case 0x000D: res = res + `\r`; break
				case 0x000C: res = res + `\f`; break
				case 0x0022: res = res + `\"`; break
				case 0x0027: res = res + `\'`; break
				case 0x005C: res = res + `\\`; break
				default:     res = res + fmt.Sprintf("\\u%04x", r); break
			}
		} else {
			i := utf8.EncodeRune(buf[:], r)
			res = res + string(buf[:i])
		}
	}
	return res
}