package resolvers

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"

	"github.com/sourcegraph/sourcegraph/cmd/frontend/backend"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/graphqlbackend"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/lsifserver/client"
	"github.com/sourcegraph/sourcegraph/enterprise/internal/codeintel/resolvers/diff"
	"github.com/sourcegraph/sourcegraph/internal/api"
	"github.com/sourcegraph/sourcegraph/internal/lsif"
	"github.com/sourcegraph/sourcegraph/internal/vcs/git"
)

type lsifQueryResolver struct {
	repositoryResolver *graphqlbackend.RepositoryResolver
	// commit is the requested target commit
	commit graphqlbackend.GitObjectID
	path   string
	// uploads are ordered by their commit distance from the target commit
	uploads []*lsif.LSIFUpload
}

var _ graphqlbackend.LSIFQueryResolver = &lsifQueryResolver{}

func (r *lsifQueryResolver) Definitions(ctx context.Context, args *graphqlbackend.LSIFQueryPositionArgs) (graphqlbackend.LocationConnectionResolver, error) {
	for _, upload := range r.uploads {
		position, err := r.adjustPosition(ctx, upload.Commit, args.Line, args.Character)
		if err != nil {
			return nil, err
		}

		opts := &struct {
			RepoID    api.RepoID
			Commit    graphqlbackend.GitObjectID
			Path      string
			Line      int32
			Character int32
			UploadID  int64
		}{
			RepoID:    r.repositoryResolver.Type().ID,
			Commit:    r.commit,
			Path:      r.path,
			Line:      position.Line,
			Character: position.Character,
			UploadID:  upload.ID,
		}

		locations, _, err := client.DefaultClient.Definitions(ctx, opts)
		if err != nil {
			return nil, err
		}

		if len(locations) > 0 {
			return &locationConnectionResolver{
				locations: locations,
			}, nil
		}
	}

	return &locationConnectionResolver{locations: nil}, nil
}

func (r *lsifQueryResolver) References(ctx context.Context, args *graphqlbackend.LSIFPagedQueryPositionArgs) (graphqlbackend.LocationConnectionResolver, error) {
	// Decode a map of upload ids to the next url that serves
	// the new page of results. This may not include an entry
	// for every upload if their result sets have already been
	// exhausted.
	nextURLs, err := readCursor(args.After)
	if err != nil {
		return nil, err
	}

	// We need to maintain a symmetric map for the next page
	// of results that we can encode into the endCursor of
	// this request.
	newCursors := map[int64]string{}

	var allLocations []*lsif.LSIFLocation
	for _, upload := range r.uploads {
		position, err := r.adjustPosition(ctx, upload.Commit, args.Line, args.Character)
		if err != nil {
			return nil, err
		}

		opts := &struct {
			RepoID    api.RepoID
			Commit    graphqlbackend.GitObjectID
			Path      string
			Line      int32
			Character int32
			UploadID  int64
			Limit     *int32
			Cursor    *string
		}{
			RepoID:    r.repositoryResolver.Type().ID,
			Commit:    r.commit,
			Path:      r.path,
			Line:      position.Line,
			Character: position.Character,
			UploadID:  upload.ID,
		}
		if args.First != nil {
			opts.Limit = args.First
		}
		if nextURL, ok := nextURLs[upload.ID]; ok {
			opts.Cursor = &nextURL
		} else if len(nextURLs) != 0 {
			// Result set is exhausted or newer than the first page
			// of results. Skip anything from this upload as it will
			// have duplicate results, or it will be out of order.
			continue
		}

		locations, nextURL, err := client.DefaultClient.References(ctx, opts)
		if err != nil {
			return nil, err
		}
		allLocations = append(allLocations, locations...)

		if nextURL != "" {
			newCursors[upload.ID] = nextURL
		}
	}

	endCursor, err := makeCursor(newCursors)
	if err != nil {
		return nil, err
	}

	return &locationConnectionResolver{
		locations: allLocations,
		endCursor: endCursor,
	}, nil
}

func (r *lsifQueryResolver) Hover(ctx context.Context, args *graphqlbackend.LSIFQueryPositionArgs) (graphqlbackend.HoverResolver, error) {
	for _, upload := range r.uploads {
		position, err := r.adjustPosition(ctx, upload.Commit, args.Line, args.Character)
		if err != nil {
			return nil, err
		}

		text, lspRange, err := client.DefaultClient.Hover(ctx, &struct {
			RepoID    api.RepoID
			Commit    graphqlbackend.GitObjectID
			Path      string
			Line      int32
			Character int32
			UploadID  int64
		}{
			RepoID:    r.repositoryResolver.Type().ID,
			Commit:    r.commit,
			Path:      r.path,
			Line:      position.Line,
			Character: position.Character,
			UploadID:  upload.ID,
		})
		if err != nil {
			return nil, err
		}

		if text != "" {
			return &hoverResolver{
				text:     text,
				lspRange: lspRange,
			}, nil
		}
	}

	return nil, nil
}

// readCursor decodes a cursor into a map from upload ids to URLs that
// serves the next page of results.
func readCursor(after *string) (map[int64]string, error) {
	if after == nil {
		return nil, nil
	}

	decoded, err := base64.StdEncoding.DecodeString(*after)
	if err != nil {
		return nil, err
	}

	var cursors map[int64]string
	if err := json.Unmarshal(decoded, &cursors); err != nil {
		return nil, err
	}
	return cursors, nil
}

// makeCursor encodes a map from upload ids to URLs that serves the next
// page of results into a single string that can be sent back for use in
// cursor pagination.
func makeCursor(cursors map[int64]string) (string, error) {
	if len(cursors) == 0 {
		return "", nil
	}

	encoded, err := json.Marshal(cursors)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(encoded), nil
}

func (r *lsifQueryResolver) adjustPosition(ctx context.Context, uploadCommit string, line, character int32) (*diff.Position, error) {
	if uploadCommit == string(r.commit) {
		return &diff.Position{
			Line:      line,
			Character: character,
		}, nil
	}

	reader, err := r.diffReader(ctx, uploadCommit)
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	return diff.AdjustPosition(reader, line, character)
}

func (r lsifQueryResolver) diffReader(ctx context.Context, uploadCommit string) (io.ReadCloser, error) {
	cachedRepo, err := backend.CachedGitRepo(ctx, r.repositoryResolver.Type())
	if err != nil {
		return nil, err
	}

	return git.ExecReader(
		ctx,
		*cachedRepo,
		[]string{
			"diff",
			uploadCommit,
			string(r.commit),
			"--",
			r.path,
		},
	)
}
