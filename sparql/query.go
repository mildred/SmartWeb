package sparql

import (
	"regexp"
	"strconv"
	"fmt"
)

var replacement, _ = regexp.Compile("%(([0-9]+)[vsu]|%)")

func MakeQuery(template string, args ...interface{}) string {
	return replacement.ReplaceAllStringFunc(template, func(repl string) string {
		if repl == "%%" {
			return "%"
		}
		n, _ := strconv.ParseInt(repl[1:len(repl)-1], 10, 0)
		n--
		c := repl[len(repl)-1]
		if n < 0 || int(n) >= len(args) {
			return repl
		}

		switch c {
			default:  return repl
			case 'v': 
				lit, err := Literal(args[n])
				if err != nil {
					return repl
				} else {
					return lit
				}
			case 's': return StringLiteral(fmt.Sprintf("%v", args[n]))
			case 'u': return IRILiteral(fmt.Sprintf("%v", args[n]))
		}
	})
}