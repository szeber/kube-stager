package errors

import "fmt"

type UnresolvedTemplatesError struct {
	TemplateVariables []string
	EntityType        string
	Key               string
}

func (r UnresolvedTemplatesError) Error() string {
	if "" == r.Key {
		return fmt.Sprintf(
			"Not all templates have been resolved in the %s. Unresolved template variables: %v",
			r.EntityType,
			r.TemplateVariables,
		)
	} else {
		return fmt.Sprintf(
			"Not all templates have been resolved in the %s at key %s. Unresolved template variables: %v",
			r.EntityType,
			r.Key,
			r.TemplateVariables,
		)
	}
}

func (r UnresolvedTemplatesError) IsFinal() bool {
	return true
}
