package main

import (
	"os"
	"path/filepath"
	"regexp"
)

func editGoModFile(inPath, outPath string) error {
	inModFile, err := os.ReadFile(filepath.Join(inPath, "go.mod"))
	if err != nil {
		return err
	}
	inSumFile, err := os.ReadFile(filepath.Join(inPath, "go.sum"))
	if err != nil {
		return err
	}
	outModFile, err := os.ReadFile(filepath.Join(outPath, "go.mod"))
	if err != nil {
		return err
	}
	outSumFile, err := os.ReadFile(filepath.Join(outPath, "go.sum"))
	if err != nil {
		return err
	}
	re := regexp.MustCompile("^\t?github.com/GoogleCloudPlatform/declarative-resource-client-library.*$")
	modFileLine := re.Find(inModFile)
	sumFileLine := re.Find(inSumFile)

	return nil
}
