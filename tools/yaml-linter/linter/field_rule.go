package linter

import "github.com/GoogleCloudPlatform/magic-modules/mmv1/api"

type FieldRule struct {
	Name     string
	Messages func(field *api.Type) []FieldMessage
}

type FieldMessage struct {
	Path    string
	Message string
}

func ImmutableWithUpdateURLOrVerbMessages(field *api.Type, path string) []FieldMessage {
	if !field.Immutable {
		return nil
	}
	var messages []FieldMessage
	if field.UpdateVerb != "" {
		messages = append(messages, FieldMessage{
			Path:    path,
			Message: "update_verb included with immutable on " + field.Name,
		})
	}
	if field.UpdateUrl != "" {
		messages = append(messages, FieldMessage{
			Path:    path,
			Message: "update_url included with immutable on " + field.Name,
		})
	}
	if len(messages) > 0 {
		return messages
	}
	return nil
}
