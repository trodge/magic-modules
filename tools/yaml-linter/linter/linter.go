package linter

import (
	"github.com/GoogleCloudPlatform/magic-modules/mmv1/loader"
)

func ApplyRules() error {
	loader := loader.NewLoader()

	products := loader.LoadProducts()

	for _, product := range products {

	}

	return nil
}
