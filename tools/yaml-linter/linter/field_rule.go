package linter

import "github.com/GoogleCloudPlatform/magic-modules/mmv1/api"

type FieldRule struct {
	Name     string
	Messages func(field *api.Type) []string
}

func ImmutableWithUpdateURLOrVerbMessages(field *api.Type) []string {
	if !field.Immutable {
		return nil
	}
	var messages []string
	if field.UpdateVerb != "" {
		messages = append(messages, "update_verb included with immutable on "+field.Name)
	}
	if field.UpdateUrl != "" {
		messages = append(messages, "update_url included with immutable on "+field.Name)
	}
	if len(messages) > 0 {
		return messages
	}
	return nil
}
