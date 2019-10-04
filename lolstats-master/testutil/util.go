package testutil

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/aryann/difflib"
)

// TODO(Cameron): If other tests end up needing this we should move it into
// some common testing lib.
func DiffLines(aData, bData []byte, t *testing.T) string {
	// compute the diff
	diff := difflib.Diff(lines(aData), lines(bData))

	// write the diff
	buf := bytes.Buffer{}
	buf.WriteString("\n")
	numDiffs := 0
	for line, d := range diff {
		if d.Delta != difflib.Common {
			numDiffs++

			_, err := buf.WriteString(fmt.Sprintf("L%d\t %s\n", line+1, d))
			if err != nil {
				t.Fatal(err)
			}

			if numDiffs > 50 {
				buf.WriteString("too many errors (50) found. stopping...\n")
				break
			}
		}
	}

	return buf.String()
}

func lines(data []byte) []string {
	return strings.Split(string(data), "\n")
}
