package placeholders

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/AnotherFullstackDev/cloud-ctl/internal/lib"
	"github.com/AnotherFullstackDev/cloud-ctl/internal/placeholders/git"
)

type PlaceholderResolver func() (string, error)

type placeholderModifier struct {
	name string
	args []string
}

type modifierResolver func(string, []string) (string, error)

type placeholder struct {
	raw           string
	value         string
	modifiers     []placeholderModifier
	rawStartIdx   int
	rawEndIdx     int
	valueStartIdx int
	valueEndIdx   int
}

type Service struct {
	gitRepoInfo git.RepositoryInfoService
}

func NewService(gitRepoInfo git.RepositoryInfoService) *Service {
	return &Service{
		gitRepoInfo: gitRepoInfo,
	}
}

// TODO: consider using Go templates for placeholder resolution
func (s *Service) extractPlaceholders(value string) ([]placeholder, error) {
	placeholderRegExp, err := regexp.Compile(`{{\s*([^{}]+)\s*}}`)
	if err != nil {
		return nil, fmt.Errorf("failed to compile placeholders regex: %w", err)
	}
	modifierRefExp, err := regexp.Compile(`(\w+)(\(([^()]*)\))?`)
	if err != nil {
		return nil, fmt.Errorf("failed to compile placeholder modifier regex: %w", err)
	}

	matches := placeholderRegExp.FindAllStringSubmatchIndex(value, -1)
	placeholders := make([]placeholder, 0, len(matches))

	for _, match := range matches {
		if len(match) < 4 {
			return nil, fmt.Errorf("invalid match structure")
		}

		rawStartIdx, rawEndIdx := match[0], match[1]
		valueStartIdx, valueEndIdx := match[2], match[3]

		if rawStartIdx < 0 || valueStartIdx < 0 || rawEndIdx > len(value) || valueEndIdx > len(value) {
			return nil, fmt.Errorf("match indices out of bounds")
		}
		if rawStartIdx >= rawEndIdx || valueStartIdx >= valueEndIdx {
			return nil, fmt.Errorf("invalid match indices")
		}
		if rawStartIdx >= valueStartIdx || valueEndIdx >= rawEndIdx {
			return nil, fmt.Errorf("mismatched match indices")
		}

		raw := value[rawStartIdx:rawEndIdx]
		fullInnerValue := value[valueStartIdx:valueEndIdx]

		valueParts := strings.Split(fullInnerValue, "|")
		innerValue := strings.TrimSpace(valueParts[0])
		modifiers := make([]placeholderModifier, 0, len(valueParts)-1)
		for _, part := range valueParts[1:] {
			rawModifier := strings.TrimSpace(part)
			if rawModifier == "" {
				continue
			}
			modifierMatch := modifierRefExp.FindStringSubmatch(rawModifier)

			modifierName := modifierMatch[1]
			if modifierName == "" {
				return nil, fmt.Errorf("invalid modifier format in placeholder: %s. %w", raw, lib.BadUserInputError)
			}

			modifierArgs := make([]string, 0, 5) // preallocate for up to 5 args. Most modifiers won't have that many.
			modifierArgsRaw := modifierMatch[3]
			if modifierArgsRaw != "" {
				modifierArgs = strings.Split(modifierArgsRaw, ",")
				for i := range modifierArgs {
					modifierArgs[i] = strings.TrimSpace(modifierArgs[i])
					if unquoted, err := strconv.Unquote(modifierArgs[i]); err == nil {
						modifierArgs[i] = unquoted
					}
				}
			}

			modifiers = append(modifiers, placeholderModifier{
				name: modifierName,
				args: modifierArgs,
			})
		}

		placeholders = append(placeholders, placeholder{
			raw:           raw,
			value:         innerValue,
			modifiers:     modifiers,
			rawStartIdx:   rawStartIdx,
			rawEndIdx:     rawEndIdx,
			valueStartIdx: valueStartIdx,
			valueEndIdx:   valueEndIdx,
		})
	}

	return placeholders, nil
}

func (s *Service) ResolvePlaceholders(value string, extraResolvers ...map[string]PlaceholderResolver) (string, error) {
	placeholders, err := s.extractPlaceholders(value)
	if err != nil {
		return "", fmt.Errorf("extracting placeholders: %w", err)
	}

	placeholderResolvers := map[string]PlaceholderResolver{
		"git.branch":     s.resolveGitBranch,
		"git.commit":     s.resolveGitCommit,
		"git.tag":        s.resolveGitTag,
		"time.timestamp": resolveUnixTimestamp,
		"time.iso8601":   resolveISO8601Timestamp,
	}

	modifierResolvers := map[string]modifierResolver{
		"upper":       upperModifier,
		"lower":       lowerModifier,
		"trim":        trimModifier,
		"replace":     replaceModifier,
		"replace_all": replaceAllModifier,
	}

	for _, placeholder := range placeholders {
		resolver, ok := placeholderResolvers[placeholder.value]

		if !ok {
		extraResolversLoop:
			for _, resolvers := range extraResolvers {
				if extraResolver, exists := resolvers[placeholder.value]; exists {
					resolver = extraResolver
					ok = true
					break extraResolversLoop
				}
			}
		}

		if !ok {

			return "", fmt.Errorf("no resolver found for placeholder: %s. %w", placeholder.raw, lib.BadUserInputError)
		}

		resolvedValue, err := resolver()
		if err != nil {
			return "", fmt.Errorf("resolving placeholder %s: %w", placeholder.raw, err)
		}

		if len(placeholder.modifiers) > 0 {
			for _, modifier := range placeholder.modifiers {
				modifierFunc, ok := modifierResolvers[modifier.name]
				if !ok {
					return "", fmt.Errorf("no resolver found for modifier: %s in placeholder: %s. %w", modifier.name, placeholder.raw, lib.BadUserInputError)
				}

				resolvedValue, err = modifierFunc(resolvedValue, modifier.args)
				if err != nil {
					return "", fmt.Errorf("applying modifier %s to placeholder %s: %w", modifier.name, placeholder.raw, err)
				}
			}
		}

		value = strings.Replace(value, placeholder.raw, resolvedValue, 1)
	}

	return value, nil
}

func (s *Service) resolveGitBranch() (string, error) {
	branch, err := s.gitRepoInfo.CurrentBranch()
	if err != nil {
		return "", fmt.Errorf("getting current git branch: %w", err)
	}
	return branch, nil
}

func (s *Service) resolveGitTag() (string, error) {
	tag, err := s.gitRepoInfo.CurrentTag()
	if err != nil {
		return "", fmt.Errorf("getting current git tag: %w", err)
	}

	if tag == nil {
		return "", fmt.Errorf("no git tag found for current commit: %w", lib.BadUserInputError)
	}

	return tag.Name().Short(), nil
}

func (s *Service) resolveGitCommit() (string, error) {
	commit, err := s.gitRepoInfo.CurrentCommit()
	if err != nil {
		return "", fmt.Errorf("getting current git commit: %w", err)
	}
	return commit.Hash.String(), nil
}
