package container_image

type PlaceholdersResolver interface {
	ResolvePlaceholders(input string, extraResolvers map[string]func() (string, error)) (string, error)
}
