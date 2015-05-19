package sparql

import (
	"regexp"
	"strconv"
	"fmt"
)

var replacement, _ = regexp.Compile("%(([0-9]+)[vsuq]|%)")

// Format SPARQL query, replace patterns in the form %#C where # is the argument
// number (starting at 1) and C the format.
// %1u will put an URL value contained in the first argument
// %2v will put any SPARQL value contained in the second argument
// %3s will put a SPARQL string contained in the third argument
// %4q will insert the 4th argument as it is without formatting.
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
			case 'q': return fmt.Sprintf("%s", args[n])
		}
	})
}