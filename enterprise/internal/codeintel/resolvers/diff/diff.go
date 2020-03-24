package diff

import (
	"fmt"
	"io"
	"strings"

	"github.com/sourcegraph/go-diff/diff"
)

type Position struct {
	Line      int32
	Character int32
}

// TODO - document, refactor

func AdjustPosition(reader io.Reader, line, character int32) (*Position, error) {
	diff, err := diff.NewFileDiffReader(reader).Read()
	if err != nil {
		return nil, err
	}

	for i, hunk := range diff.Hunks {
		if line < hunk.OrigStartLine {
			if i == 0 {
				return &Position{Line: line, Character: character}, nil
			}

			l := diff.Hunks[i-1]
			return &Position{
				Line:      line + (l.NewStartLine - l.OrigStartLine) + (l.NewLines - l.OrigLines),
				Character: character,
			}, nil
		}

		if line < hunk.OrigStartLine+hunk.OrigLines {
			bodyLines := strings.Split(string(hunk.Body), "\n")

			left := hunk.OrigStartLine - 1
			right := hunk.NewStartLine - 1
			for _, bodyLine := range bodyLines {
				changed := false
				if strings.HasPrefix(bodyLine, "+") {
					right++
					changed = true
				} else if strings.HasPrefix(bodyLine, "-") {
					left++
					changed = true
				} else {
					left++
					right++
				}

				if left == line {
					if changed {
						return nil, nil
					}

					return &Position{Line: right, Character: character}, nil
				}
			}

			return nil, fmt.Errorf("malformed git diff hunk")
		}
	}

	l := diff.Hunks[len(diff.Hunks)-1]
	return &Position{
		Line:      line + (l.NewStartLine - l.OrigStartLine) + (l.NewLines - l.OrigLines),
		Character: character,
	}, nil
}
