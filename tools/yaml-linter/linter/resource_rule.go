package linter

import "github.com/GoogleCloudPlatform/magic-modules/mmv1/api"

type ResourceRule struct {
	Name     string
	Messages func(resource *api.Resource) []string
}
