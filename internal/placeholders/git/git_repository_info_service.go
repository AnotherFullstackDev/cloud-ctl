package git

import (
	"fmt"

	"github.com/go-git/go-git/v6"
	"github.com/go-git/go-git/v6/plumbing"
	"github.com/go-git/go-git/v6/plumbing/object"
)

type RepositoryInfoService interface {
	CurrentBranch() (string, error)
	CurrentCommit() (*object.Commit, error)
	CurrentTag() (*plumbing.Reference, error)
	TagsPointingAt(hash plumbing.Hash) ([]*plumbing.Reference, error)
}

type repositoryInfoServiceImpl struct {
	r *git.Repository
}

func NewRepositoryInfoService(repoPath string) (RepositoryInfoService, error) {
	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return nil, fmt.Errorf("opening repository: %w", err)
	}

	return &repositoryInfoServiceImpl{
		r: repo,
	}, nil
}

func (s *repositoryInfoServiceImpl) CurrentBranch() (string, error) {
	headRef, err := s.r.Head()
	if err != nil {
		return "", fmt.Errorf("getting HEAD reference: %w", err)
	}

	name := headRef.Name()
	if !name.IsBranch() {
		return "", fmt.Errorf("HEAD is not pointing to a branch")
	}

	return name.Short(), nil
}

func (s *repositoryInfoServiceImpl) CurrentCommit() (*object.Commit, error) {
	headRef, err := s.r.Head()
	if err != nil {
		return nil, fmt.Errorf("getting HEAD reference: %w", err)
	}

	commit, err := s.r.CommitObject(headRef.Hash())
	if err != nil {
		return nil, fmt.Errorf("getting commit object: %w", err)
	}

	return commit, nil
}

// TagsPointingAt returns a slice of tag references that point to the given commit hash.
// tag.Name() is like "refs/tags/v1.0.0"
// tag.Hash() is the hash of the tag object or commit it points to.
func (s *repositoryInfoServiceImpl) TagsPointingAt(hash plumbing.Hash) ([]*plumbing.Reference, error) {
	tagsIter, err := s.r.Tags()
	if err != nil {
		return nil, fmt.Errorf("getting tags iterator: %w", err)
	}
	defer tagsIter.Close()

	// Allocating tags slice with an initial capacity that must cover most use cases without internal array resize
	tags := make([]*plumbing.Reference, 0, 10)
	err = tagsIter.ForEach(func(reference *plumbing.Reference) error {
		// For lightweight tags, the reference points directly to the commit
		// For annotated tags, the reference points to a tag object, which in turn points to the commit
		if reference.Hash().Equal(hash) {
			tags = append(tags, reference)
			return nil
		}

		// Check for annotated tags
		obj, err := s.r.TagObject(reference.Hash())
		if err != nil {
			// Not a tag object, skip
			return nil
		}

		if obj.Hash.Equal(hash) {
			tags = append(tags, reference)
			return nil
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("iterating over tags: %w", err)
	}

	return tags, nil
}

func (s *repositoryInfoServiceImpl) CurrentTag() (*plumbing.Reference, error) {
	commit, err := s.CurrentCommit()
	if err != nil {
		return nil, fmt.Errorf("getting current commit: %w", err)
	}

	tags, err := s.TagsPointingAt(commit.Hash)
	if err != nil {
		return nil, fmt.Errorf("getting tags pointing at current commit: %w", err)
	}
	if len(tags) == 0 {
		return nil, nil
	}

	return tags[0], nil
}
