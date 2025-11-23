package placeholders

import (
	"errors"
	"fmt"
	"testing"

	"github.com/go-git/go-git/v6/plumbing"
	"github.com/go-git/go-git/v6/plumbing/object"
	"github.com/stretchr/testify/require"
)

type mockTag struct {
	Name string
	Hash string
}

type mockGitRepoInfoService struct {
	Branch string
	Commit string
	Tag    *mockTag
	Tags   []mockTag
}

func (m mockGitRepoInfoService) getMockValueOrError(value any) (any, error) {
	switch v := value.(type) {
	case string:
		if value == "" {
			return "", errors.New("value is empty")
		}
		return v, nil
	case []mockTag:
		if len(v) == 0 {
			return nil, errors.New("value is empty")
		}
		return v, nil
	case *mockTag:
		if v == nil {
			return nil, errors.New("value is nil")
		}
		return v, nil
	default:
		return nil, fmt.Errorf("unsupported type: %T", value)
	}
}

func (m mockGitRepoInfoService) CurrentBranch() (string, error) {
	b, err := m.getMockValueOrError(m.Branch)
	return b.(string), err
}

func (m mockGitRepoInfoService) CurrentCommit() (*object.Commit, error) {
	c, err := m.getMockValueOrError(m.Commit)
	if err != nil {
		return nil, err
	}
	hash, ok := plumbing.FromHex(c.(string))
	if !ok {
		return nil, fmt.Errorf("parsing commit hash is not successful: %s", c.(string))
	}
	return &object.Commit{
		Hash: hash,
	}, nil
}

func (m mockGitRepoInfoService) CurrentTag() (*plumbing.Reference, error) {
	t, err := m.getMockValueOrError(m.Tag)
	if err != nil {
		return nil, err
	}
	hash, ok := plumbing.FromHex(t.(*mockTag).Hash)
	if !ok {
		return nil, fmt.Errorf("parsing tag hash is not successful: %s", t.(*mockTag).Hash)
	}
	return plumbing.NewHashReference(plumbing.NewTagReferenceName(t.(*mockTag).Name), hash), nil
}

func (m mockGitRepoInfoService) TagsPointingAt(hash plumbing.Hash) ([]*plumbing.Reference, error) {
	tags, err := m.getMockValueOrError(m.Tags)
	if err != nil {
		return nil, err
	}
	var refs []*plumbing.Reference
	for _, t := range tags.([]mockTag) {
		tagHash, ok := plumbing.FromHex(t.Hash)
		if !ok {
			return nil, fmt.Errorf("parsing tag hash is not successful")
		}
		if tagHash == hash {
			refs = append(refs, plumbing.NewHashReference(plumbing.NewTagReferenceName(t.Name), tagHash))
		}
	}
	return refs, nil
}

func TestPlaceholdersParsing(t *testing.T) {
	emptyRepoInfoService := mockGitRepoInfoService{}
	r := require.New(t)

	t.Run("should parse simple placeholder", func(t *testing.T) {
		s := NewService(emptyRepoInfoService)
		value := "Deploy to {{git.branch}}"
		placeholders, err := s.extractPlaceholders(value)
		r.NoError(err)
		r.Len(placeholders, 1)
		r.Equal("git.branch", placeholders[0].value)
		r.Equal("{{git.branch}}", placeholders[0].raw)
		r.Empty(placeholders[0].modifiers)
	})

	t.Run("should parse multiple placeholders", func(t *testing.T) {
		s := NewService(emptyRepoInfoService)
		value := "Branch: {{git.branch}}, Commit: {{git.commit}}"
		placeholders, err := s.extractPlaceholders(value)
		r.NoError(err)
		r.Len(placeholders, 2)
		r.Equal("git.branch", placeholders[0].value)
		r.Equal("{{git.branch}}", placeholders[0].raw)
		r.Equal("git.commit", placeholders[1].value)
		r.Equal("{{git.commit}}", placeholders[1].raw)
	})

	t.Run("should parse placeholders when markup is harsh", func(t *testing.T) {
		s := NewService(emptyRepoInfoService)
		value := "Start{{git.branch}}Middle{{ git.commit | upper }}End{{{git.tag}}}"
		placeholders, err := s.extractPlaceholders(value)
		r.NoError(err)
		r.Len(placeholders, 3)
		r.Equal("git.branch", placeholders[0].value)
		r.Equal("{{git.branch}}", placeholders[0].raw)
		r.Equal("git.commit", placeholders[1].value)
		r.Equal("{{ git.commit | upper }}", placeholders[1].raw)
		r.Equal("git.tag", placeholders[2].value)
		r.Equal("{{git.tag}}", placeholders[2].raw)
	})

	t.Run("should parse simple modifiers", func(t *testing.T) {
		s := NewService(emptyRepoInfoService)
		value := "{{ git.branch | upper | trim }}"
		placeholders, err := s.extractPlaceholders(value)
		r.NoError(err)
		r.Len(placeholders, 1)
		r.Equal("git.branch", placeholders[0].value)
		r.Equal(2, len(placeholders[0].modifiers))
		r.Equal("upper", placeholders[0].modifiers[0].name)
		r.Empty(placeholders[0].modifiers[0].args)
		r.Equal("trim", placeholders[0].modifiers[1].name)
		r.Empty(placeholders[0].modifiers[1].args)
	})

	t.Run("should parse modifiers with arguments", func(t *testing.T) {
		s := NewService(emptyRepoInfoService)
		value := "{{ git.branch | replace_all(\"-\", \"_\") | trim(\"_\") }}"
		placeholders, err := s.extractPlaceholders(value)
		r.NoError(err)
		r.Len(placeholders, 1)
		r.Equal("git.branch", placeholders[0].value)
		r.Equal(2, len(placeholders[0].modifiers))
		r.Equal("replace_all", placeholders[0].modifiers[0].name)
		r.Equal([]string{"-", "_"}, placeholders[0].modifiers[0].args)
		r.Equal("trim", placeholders[0].modifiers[1].name)
		r.Equal([]string{"_"}, placeholders[0].modifiers[1].args)
	})

	t.Run("should parse modifiers in harsh markup", func(t *testing.T) {
		s := NewService(emptyRepoInfoService)
		value := "{{git.commit|upper|replace(\"A\",\"B\")}}}"
		placeholders, err := s.extractPlaceholders(value)
		r.NoError(err)
		r.Len(placeholders, 1)
		r.Equal("git.commit", placeholders[0].value)
		r.Equal(2, len(placeholders[0].modifiers))
		r.Equal("upper", placeholders[0].modifiers[0].name)
		r.Empty(placeholders[0].modifiers[0].args)
		r.Equal("replace", placeholders[0].modifiers[1].name)
		r.Equal([]string{"A", "B"}, placeholders[0].modifiers[1].args)
	})

	t.Run("should parse repeated modifiers", func(t *testing.T) {
		s := NewService(emptyRepoInfoService)
		value := "{{ git.tag | trim(\"v\") | trim(\"0\") | upper }} {{ git.branch | trim(\"v\") | trim(\"0\") | upper }}"
		placeholders, err := s.extractPlaceholders(value)
		r.NoError(err)
		r.Len(placeholders, 2)
		r.Equal("git.tag", placeholders[0].value)
		r.Equal(3, len(placeholders[0].modifiers))
		r.Equal("trim", placeholders[0].modifiers[0].name)
		r.Equal([]string{"v"}, placeholders[0].modifiers[0].args)
		r.Equal("trim", placeholders[0].modifiers[1].name)
		r.Equal([]string{"0"}, placeholders[0].modifiers[1].args)
		r.Equal("upper", placeholders[0].modifiers[2].name)
		r.Empty(placeholders[0].modifiers[2].args)
		r.Equal("git.branch", placeholders[1].value)
		r.Equal(3, len(placeholders[1].modifiers))
		r.Equal("trim", placeholders[1].modifiers[0].name)
		r.Equal([]string{"v"}, placeholders[1].modifiers[0].args)
		r.Equal("trim", placeholders[1].modifiers[1].name)
		r.Equal([]string{"0"}, placeholders[1].modifiers[1].args)
		r.Equal("upper", placeholders[1].modifiers[2].name)
		r.Empty(placeholders[1].modifiers[2].args)
	})
}

func TestPlaceholdersResolution(t *testing.T) {
	r := require.New(t)

	mockService := mockGitRepoInfoService{
		Branch: "main",
		Commit: "56b189842130315a634ce6d510a4578f151eca32",
		Tag:    &mockTag{Name: "v1.0.0", Hash: "56b189842130315a634ce6d510a4578f151eca32"},
		Tags: []mockTag{
			{Name: "v1.0.0", Hash: "56b189842130315a634ce6d510a4578f151eca32"},
			{Name: "latest", Hash: "56b189842130315a634ce6d510a4578f151eca32"},
		},
	}

	t.Run("should resolve simple placeholders", func(t *testing.T) {
		s := NewService(mockService)
		value := "Branch: {{git.branch}}, Commit: {{git.commit}}, Tag: {{git.tag}}"
		resolved, err := s.ResolvePlaceholders(value)
		r.NoError(err)
		expected := "Branch: main, Commit: 56b189842130315a634ce6d510a4578f151eca32, Tag: v1.0.0"
		r.Equal(expected, resolved)
	})

	t.Run("should resolve repeated simple placeholders", func(t *testing.T) {
		s := NewService(mockService)
		value := "Branch: {{{{git.branch}}, Again Branch: {{git.branch}}}}, Commit: {{{git.commit}}, Tag: {{git.tag}}} Again Commit: {{git.commit}}"
		resolved, err := s.ResolvePlaceholders(value)
		r.NoError(err)
		expected := "Branch: {{main, Again Branch: main}}, Commit: {56b189842130315a634ce6d510a4578f151eca32, Tag: v1.0.0} Again Commit: 56b189842130315a634ce6d510a4578f151eca32"
		r.Equal(expected, resolved)
	})

	t.Run("should resolve placeholders with modifiers", func(t *testing.T) {
		s := NewService(mockService)
		value := "Branch: {{ git.branch | upper }}, Commit: {{ git.commit | trim(\"56\") }}, Tag: {{ git.tag | replace(\"v\", \"version-\") }}"
		resolved, err := s.ResolvePlaceholders(value)
		r.NoError(err)
		expected := "Branch: MAIN, Commit: b189842130315a634ce6d510a4578f151eca32, Tag: version-1.0.0"
		r.Equal(expected, resolved)
	})

	t.Run("should resolve repeated placeholders with repeated modifiers", func(t *testing.T) {
		s := NewService(mockService)
		value := "{{ git.tag | trim(v) | trim(\"0\") | replace_all(., '') | upper }} {{ git.branch | trim(\"m\") | trim(\"n\") | upper }}"
		resolved, err := s.ResolvePlaceholders(value)
		r.NoError(err)
		expected := "10 AI"
		r.Equal(expected, resolved)
	})
}
