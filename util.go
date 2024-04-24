package netfunnel

import (
	"bytes"
)

func appendStr(str ...string) string {
	var buf bytes.Buffer
	for _, v := range str {
		buf.WriteString(v)
	}

	return buf.String()
}
