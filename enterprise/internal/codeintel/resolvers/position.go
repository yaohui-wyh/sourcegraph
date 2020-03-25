package resolvers

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/sourcegraph/go-diff/diff"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/backend"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/types"
	"github.com/sourcegraph/sourcegraph/internal/vcs/git"
)

type Position struct {
	Line      int32
	Character int32
}

// adjustPosition transforms the position denoted by `line` and `character` in the target commit
// to a position in the source commit. This function returns a nil position if that particular line
// does not exist or has been edited in between the target and source commit.
func adjustPosition(
	ctx context.Context,
	repo *types.Repo,
	sourceCommit string,
	targetCommit string,
	path string,
	line int32,
	character int32,
) (*Position, error) {
	if sourceCommit == targetCommit {
		// Trivial case, we have this exact commit indexed
		return &Position{Line: line, Character: character}, nil
	}

	cachedRepo, err := backend.CachedGitRepo(ctx, repo)
	if err != nil {
		return nil, err
	}

	reader, err := git.ExecReader(
		ctx,
		*cachedRepo,
		[]string{"diff", sourceCommit, targetCommit, "--", path},
	)
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	// Non-trivial case, adjust the line offset based on diff hunks
	return adjustPositionFromDiff(reader, line, character)
}

// adjustPositionFromDiff transforms the position denoted by `line` and `character` in the
// *original file* into a position in the *new file* according to the git diff output contained
// in the given reader. This function returns a nil position if the line has been edited or does
// not exist in the new file.
func adjustPositionFromDiff(reader io.Reader, line, character int32) (*Position, error) {
	// Extract hunks from the git diff output
	diff, err := diff.NewFileDiffReader(reader).Read()
	if err != nil {
		return nil, err
	}
	hunks := diff.Hunks

	// Find the index of the first hunk that starts after the target line and use the
	// previous hunk (if it exists) as the point of reference in `adjustPositionFromHunk`.

	i := 0
	for i < len(hunks) && hunks[i].OrigStartLine <= line {
		i++
	}

	if i == 0 {
		return adjustPositionFromHunk(nil, line, character)
	}
	return adjustPositionFromHunk(hunks[i-1], line, character)
}

// adjustPositionFromHunk transforms the position denoted by `line` and `character` in the
// *original file* into a position in the *new file* according to the given git diff hunk.
// This parameter is expected to be the *last* such hunk in the diff between the original
// and the new file that does not begin after the given position in the original file.
func adjustPositionFromHunk(hunk *diff.Hunk, line, character int32) (*Position, error) {
	if hunk == nil {
		// No hunk before this line, so no line offset
		return &Position{Line: line, Character: character}, nil
	}

	if line >= hunk.OrigStartLine+hunk.OrigLines {
		// Hunk ends before this line, so we can simply adjust the line offset by the
		// relative difference between the line offsets in each file after this hunk.
		relativeDifference := (hunk.NewStartLine + hunk.NewLines) - (hunk.OrigStartLine + hunk.OrigLines)
		return &Position{Line: line + relativeDifference, Character: character}, nil
	}

	// Create two *fingers* pointing at the first line of this hunk in each file
	origFileOffset := hunk.OrigStartLine
	newFileOffset := hunk.NewStartLine

	// Bump each of these cursors for every line in hunk body that is attributed
	// to the corresponding file.
	for _, bodyLine := range strings.Split(string(hunk.Body), "\n") {
		// Bump original file offset unless it's an addition in the new file
		added := strings.HasPrefix(bodyLine, "+")
		if !added {
			origFileOffset++
		}

		// Bump new file offset unless it's a deletion of a line from the new file
		removed := strings.HasPrefix(bodyLine, "-")
		if !removed {
			newFileOffset++
		}

		// Keep skipping ahead in the original file until we hit our target line
		if origFileOffset-1 < line {
			continue
		}

		// This line exists in both files
		if !added && !removed {
			return &Position{Line: newFileOffset - 1, Character: character}, nil
		}

		// Fail the position adjustment. This particular line
		//   (1) edited;
		//   (2) removed in which case we can't point to it; or
		//   (3) added, in which case it hasn't been indexed.
		//
		// In all cases we don't want to return any results here as we
		// don't have enough information to give a precise result matching
		// the current query text.

		return nil, nil
	}

	// This should never happen unless the git diff content is malformed.
	// We know the target line occurs within the hunk, but iteration of the
	// hunk's body did not contain enough lines attributed to the original
	// file.
	return nil, fmt.Errorf("malformed git diff hunk")
}
