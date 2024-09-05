/*
* Copyright 2023 Google LLC. All Rights Reserved.
*
* Licensed under the Apache License, Version 2.0 (the "License");
* you may not use this file except in compliance with the License.
* You may obtain a copy of the License at
*
*     http://www.apache.org/licenses/LICENSE-2.0
*
* Unless required by applicable law or agreed to in writing, software
* distributed under the License is distributed on an "AS IS" BASIS,
* WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
* See the License for the specific language governing permissions and
* limitations under the License.
 */
package cmd

import (
	"fmt"
	"magician/github"
	"math/rand"
	"strings"

	"github.com/trodge/magic-modules/tools/issue-labeler/labeler"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

// requestServiceReviewersCmd represents the requestServiceReviewers command
var requestServiceReviewersCmd = &cobra.Command{
	Use:   "request-service-reviewers PR_NUMBER",
	Short: "Assigns reviewers based on the PR's service labels.",
	Long: `This command requests (or re-requests) review based on the PR's service labels.

	If a PR has more than 3 service labels, the command will not do anything.
	`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		prNumber := args[0]
		fmt.Println("PR Number: ", prNumber)

		githubToken, ok := lookupGithubTokenOrFallback("GITHUB_TOKEN_MAGIC_MODULES")
		if !ok {
			return fmt.Errorf("did not provide GITHUB_TOKEN_MAGIC_MODULES or GITHUB_TOKEN environment variable")
		}
		gh := github.NewClient(githubToken)
		return execRequestServiceReviewers(prNumber, gh, labeler.EnrolledTeamsYaml)
	},
}

// TODO: Switch to labeler.LabelData after a soak period.
type LabelData struct {
	Team string `yaml:"team,omitempty"`
}

func execRequestServiceReviewers(prNumber string, gh GithubClient, enrolledTeamsYaml []byte) error {
	pullRequest, err := gh.GetPullRequest(prNumber)
	if err != nil {
		return err
	}

	enrolledTeams := make(map[string]LabelData)
	if err := yaml.Unmarshal(enrolledTeamsYaml, &enrolledTeams); err != nil {
		return fmt.Errorf("error unmarshalling enrolled teams yaml: %w", err)
	}

	requestedReviewers, err := gh.GetPullRequestRequestedReviewers(prNumber)
	if err != nil {
		return err
	}

	previousReviewers, err := gh.GetPullRequestPreviousReviewers(prNumber)
	if err != nil {
		return err
	}

	// If more than three service labels are impacted, don't request reviews.
	// Only request reviews from unique service teams.
	githubTeamsSet := make(map[string]struct{})
	teamCount := 0
	for _, label := range pullRequest.Labels {
		if !strings.HasPrefix(label.Name, "service/") || label.Name == "service/terraform" {
			continue
		}
		teamCount += 1
		if labelData, ok := enrolledTeams[label.Name]; ok && labelData.Team != "" {
			githubTeamsSet[labelData.Team] = struct{}{}
		}
	}

	if teamCount > 3 {
		fmt.Println("Provider-wide change (>3 services impacted); not requesting service team reviews")
		return nil
	}

	// For each service team, check if one of the team members is already a reviewer. Rerequest
	// review if there is and choose a random reviewer from the list if there isn't.
	reviewersToRequest := []string{}
	requestedReviewersSet := make(map[string]struct{})
	for _, reviewer := range requestedReviewers {
		requestedReviewersSet[reviewer.Login] = struct{}{}
	}

	previousReviewersSet := make(map[string]struct{})
	for _, reviewer := range previousReviewers {
		previousReviewersSet[reviewer.Login] = struct{}{}
	}

	exitCode := 0
	for githubTeam := range githubTeamsSet {
		members, err := gh.GetTeamMembers("trodge", githubTeam)
		if err != nil {
			fmt.Printf("Error fetching members for trodge/%s: %s", githubTeam, err)
			exitCode = 1
			continue
		}
		hasReviewer := false
		reviewerPool := []string{}
		for _, member := range members {
			// Skip PR author
			if member.Login == pullRequest.User.Login {
				continue
			}

			reviewerPool = append(reviewerPool, member.Login)
			// Don't re-request review if there's an active review request
			if _, ok := requestedReviewersSet[member.Login]; ok {
				hasReviewer = true
			}
			if _, ok := previousReviewersSet[member.Login]; ok {
				hasReviewer = true
				reviewersToRequest = append(reviewersToRequest, member.Login)
			}
		}

		if !hasReviewer && len(reviewerPool) > 0 {
			reviewersToRequest = append(reviewersToRequest, reviewerPool[rand.Intn(len(reviewerPool))])
		}
	}

	err = gh.RequestPullRequestReviewers(prNumber, reviewersToRequest)
	if err != nil {
		fmt.Println(err)
		exitCode = 1
	}
	if exitCode != 0 {
		return fmt.Errorf("exit code = %d", exitCode)
	}
	return nil
}

func init() {
	rootCmd.AddCommand(requestServiceReviewersCmd)
}
