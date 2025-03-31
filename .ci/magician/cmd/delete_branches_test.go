package cmd

import (
	"magician/exec"
	"magician/source"
	"os"
	"testing"
)

func TestFetchPRNumber(t *testing.T) {
	githubToken, ok := os.LookupEnv("GITHUB_TOKEN_CLASSIC")
	if !ok {
		t.Errorf("did not provide GITHUB_TOKEN_CLASSIC environment variable")
	}

	goPath, ok := os.LookupEnv("GOPATH")
	if !ok {
		t.Errorf("did not provide GOPATH environment variable")
	}

	rnr, err := exec.NewRunner()
	if err != nil {
		t.Errorf("error creating Runner: %s", err)
	}

	ctlr := source.NewController(goPath, "modular-magician", githubToken, rnr)

	prNumber, err := fetchPRNumber("8c6e61bb62d52c950008340deafc1e2a2041898a", "main", rnr, ctlr)

	if err != nil {
		t.Errorf("error fetching PR number: %s", err)
	}

	if prNumber != "6504" {
		t.Errorf("PR number is %s, expected 6504", prNumber)
	}
}

func TestDeleteBranches(t *testing.T) {
	githubToken, ok := os.LookupEnv("GITHUB_TOKEN_CLASSIC")
	if !ok {
		t.Errorf("did not provide GITHUB_TOKEN_CLASSIC environment variable")
	}

	goPath, ok := os.LookupEnv("GOPATH")
	if !ok {
		t.Errorf("did not provide GOPATH environment variable")
	}

	rnr, err := exec.NewRunner()
	if err != nil {
		t.Errorf("error creating Runner: %s", err)
	}

	ctlr := source.NewController(goPath, "modular-magician", githubToken, rnr)

	// Create branches
	for _, repoName := range repoList {
		repo := &source.Repo{
			Name:   repoName,
			Branch: "main",
		}
		ctlr.Clone(repo)
		for _, branch := range []string{
			"auto-pr-TEST",
			"auto-pr-TEST-old",
		} {
			_, err = rnr.Run("git", []string{
				"checkout",
				"-b",
				branch,
			}, nil)
			if err != nil {
				t.Errorf("error creating branch: %s", err)
			}

			_, err = rnr.Run("git", []string{
				"push",
				"-u",
				"origin",
				branch,
			}, nil)
			if err != nil {
				t.Errorf("error pushing branch: %s", err)
			}
		}
	}

	err = deleteBranches("TEST", githubToken, rnr)

	if err != nil {
		t.Errorf("error deleting branches: %s", err)
	}

}
