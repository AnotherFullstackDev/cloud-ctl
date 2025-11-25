package container_image

type PlaceholdersResolver interface {
	ResolvePlaceholders(input string) (string, error)
}
