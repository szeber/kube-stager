package errors

import "fmt"

type UnresolvedTemplatesError struct {
	UnresolvedTemplateVariables []string
	AvailableTemplateVariables  []string
	EntityType                  string
	Key                         string
}

func (r UnresolvedTemplatesError) Error() string {
	if "" == r.Key {
		return fmt.Sprintf(
			"Not all templates have been resolved in the %s. Unresolved template variables: %v. Available template variables: %v",
			r.EntityType,
			r.UnresolvedTemplateVariables,
			r.AvailableTemplateVariables,
		)
	} else {
		return fmt.Sprintf(
			"Not all templates have been resolved in the %s at key %s. Unresolved template variables: %v. Available template variables: %v",
			r.EntityType,
			r.Key,
			r.UnresolvedTemplateVariables,
			r.AvailableTemplateVariables,
		)
	}
}

func (r UnresolvedTemplatesError) IsFinal() bool {
	return true
}
