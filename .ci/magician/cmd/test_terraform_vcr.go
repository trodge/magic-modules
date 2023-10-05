package cmd

import (
	"fmt"
	"magician/exec"
	"magician/github"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

type vtGithub interface {
	PostBuildStatus(prNumber, title, state, targetURL, commitSha string) error
	PostComment(prNumber, comment string) error
}

type vtStorage interface{}

type vtRunner interface {
	Chdir(path string)
	WriteFile(path, data string) error
	Run(name string, args, env []string) (string, error)
	MustRun(name string, args, env []string) string
}

const accTestParalellism = "32"
const replayingTimeout = "240m"

var vcrTestCmd = &cobra.Command{
	Use:   "vcr-test",
	Short: "Run vcr tests",
	Long: `This command processes pull requests and runs VCR tests first in replaying then in recording if replaying fails.

	The following details are expected as environment variables:
	1. BUILD_ID
	2. PROJECT_ID
	3. BUILD_STEP
	4. COMMIT_SHA
	5. PR_NUMBER
	6. BASE_BRANCH (optional, default main)
	7. GOPATH
	8. GITHUB_TOKEN

	The command performs the following steps:
	`,
	Run: func(cmd *cobra.Command, args []string) {
		var opt execVCRTestOptions
		opt.buildID = os.Getenv("BUILD_ID")
		fmt.Println("Build ID: ", opt.buildID)

		opt.projectID = os.Getenv("PROJECT_ID")
		fmt.Println("Project ID: ", opt.projectID)

		opt.buildStep = os.Getenv("BUILD_STEP")
		fmt.Println("Build Step: ", opt.buildStep)

		opt.commit = os.Getenv("COMMIT_SHA")
		fmt.Println("Commit SHA: ", opt.commit)

		opt.pr = os.Getenv("PR_NUMBER")
		fmt.Println("PR Number: ", opt.pr)

		opt.baseBranch = os.Getenv("BASE_BRANCH")
		if opt.baseBranch == "" {
			opt.baseBranch = "main"
		}
		fmt.Println("Base Branch: ", opt.baseBranch)

		opt.saKey = os.Getenv("SA_KEY")
		if opt.saKey == "" {
			fmt.Println("SA_KEY environment variable not set")
		}

		opt.googleServiceAccount = os.Getenv("GOOGLE_SERVICE_ACCOUNT")
		if opt.googleServiceAccount == "" {
			fmt.Println("GOOGLE_SERVICE_ACCOUNT environment variable not set")
		}

		opt.googleProject = os.Getenv("GOOGLE_PROJECT")
		if opt.googleProject == "" {
			fmt.Println("GOOGLE_PROJECT environment variable not set")
		}

		opt.goPath = os.Getenv("GOPATH")
		fmt.Println("GOPATH:", opt.goPath)

		gh := github.NewGithubService()
		execVCRTest(opt, gh, exec.NewRunner())
	},
}

type execVCRTestOptions struct {
	buildID              string
	projectID            string
	buildStep            string
	commit               string
	pr                   string
	baseBranch           string
	saKey                string
	googleServiceAccount string
	googleProject        string
	goPath               string
}

func execVCRTest(opt execVCRTestOptions, gh vtGithub, r vtRunner) {
	githubUsername := "modular-magician"
	repo := "terraform-provider-google-beta"
	newBranch := "auto-pr-" + opt.pr
	gitRemote := fmt.Sprintf("https://github.com/%s/%s", githubUsername, repo)
	localPathParent := opt.goPath + "/src/github.com/hashicorp"
	localPath := fmt.Sprintf("%s/%s", localPathParent, repo)

	if _, err := r.Run("mkdir", []string{"-p", localPathParent}, nil); err != nil {
		fmt.Printf("Error creating parent directory for repo: %v\n", repo)
		os.Exit(1)
	}

	if _, err := r.Run("git", []string{"clone", gitRemote, localPath, "--branch", newBranch, "--depth", "2"}, nil); err != nil {
		fmt.Printf("Error cloning %s: %v\n", gitRemote, err)
		os.Exit(1)
	}

	r.Chdir(localPath)
	// Only skip tests if we can tell for sure that no go files were changed.
	fmt.Println("Checking for modified go files")
	diffs, err := r.Run("git", []string{"diff", "--name-only", "HEAD~1"}, nil)
	if err != nil {
		fmt.Println("Error running git diff: ", err)
		os.Exit(1)
	}
	goFilesChanged := false
	for _, diff := range strings.Split(diffs, "\n") {
		if strings.HasSuffix(diff, ".go") || diff == "go.mod" || diff == "go.sum" {
			goFilesChanged = true
			break
		}
	}
	if !goFilesChanged() {
		fmt.Println("Skipping tests: No go files changed")
		os.Exit(0)
	}
	fmt.Println("Running tests: Go files changed")

	// cassette retrieval
	if _, err := r.Run("mkdir", []string{"fixtures"}, nil); err != nil {
		fmt.Println("Error making fixtures dir: ", err)
	}

	if opt.baseBranch != "FEATURE-BRANCH-major-release-5.0.0" {
		// Pull main cassettes (major release uses branch specific casssettes as primary ones).
		if _, err := r.Run("gsutil", []string{"-m", "-q", "cp", "gs://ci-vcr-cassettes/beta/fixtures/*", "fixtures/"}, nil); err != nil {
			fmt.Println(err)
		}
	}

	fixturesPattern := "gs://ci-vcr-cassettes/beta/refs/branches/%s/fixtures/*"
	if opt.baseBranch != "main" {
		// Copy feature branch specific cassettes over main. This might fail but that's ok if the folder doesnt exist
		if _, err := r.Run("gsutil", []string{"-m", "-q", "cp", fmt.Sprintf(fixturesPattern, opt.baseBranch), "fixtures/"}, nil); err != nil {
			fmt.Println(err)
		}
	}
	// Copy PR branch specific cassettes over main. This might fail but that's ok if the folder doesnt exist.
	if _, err := r.Run("gsutil", []string{"-m", "-q", "cp", fmt.Sprintf(fixturesPattern, newBranch), "fixtures/"}, nil); err != nil {
		fmt.Println(err)
	}

	if err := r.WriteFile("sa_key.json", opt.saKey); err != nil {
		fmt.Println("Error writing sa_key.json: ", err)
	}

	if _, err := r.Run("gcloud", []string{
		"auth",
		"activate-service-account",
		opt.googleServiceAccount,
		fmt.Sprintf("--key-file=%s/sa_key.json", localPath),
		"--project=" + opt.googleProject,
	}, nil); err != nil {
		fmt.Println("Error from gcloud auth: ", err)
	}

	for _, dir := range []string{
		"testlog",
		"testlog/replaying",
		"testlog/recording",
		"testlog/recording_build",
		"testlog/replaying_after_recording",
		"testlog/replaying_build_after_recording",
	} {
		if _, err := r.Run("mkdir", []string{dir}, nil); err != nil {
			fmt.Printf("Error making dir %s: %v", dir, err)
		}
	}

	var googleTestDirectory string
	if allPackages, err := r.Run("go", []string{"list", "./..."}, nil); err != nil {
		fmt.Println("Error listing go modules: ", err)
	} else {
		for _, dir := range strings.Split(allPackages, "\n") {
			if !strings.Contains(dir, "github.com/hashicorp/terraform-provider-google-beta/scripts") {
				googleTestDirectory += dir + "\n"
			}
		}
	}

	fmt.Println("Checking terraform version")
	fmt.Println(r.MustRun("terraform", []string{"version"}, nil))

	// Build the tests.
	if _, err := r.Run("go", []string{"build", googleTestDirectory}, nil); err != nil {
		fmt.Println("Skipping tests: Build failure detected")
		os.Exit(1)
	}

	// Update build status to pending.
	targetURL := fmt.Sprintf("https://console.cloud.google.com/cloud-build/builds;region=global/%s;step=%s?project=%s", opt.buildID, opt.buildStep, opt.projectID)
	title := "VCR-test"
	if err := gh.PostBuildStatus(opt.pr, title, "pending", targetURL, opt.commit); err != nil {
		fmt.Println("Error posting build status: ", err)
	}

	env := []string{
		"GOOGLE_REGION=us-central1",
		"GOOGLE_ZONE=us-central1-a",
		fmt.Sprintf("VCR_PATH=%s/fixtures", localPath),
		"VCR_MODE=REPLAYING",
		"ACCTEST_PARALLELISM=" + accTestParalellism,
		"GOOGLE_CREDENTIALS=" + opt.saKey,
		fmt.Sprintf("GOOGLE_APPLICATION_CREDENTIALS=%s/sa_key.json", localPath),
		"GOOGLE_TEST_DIRECTORY=" + googleTestDirectory,
		"TF_LOG=DEBUG",
		fmt.Sprintf("TF_LOG_PATH_MASK=%s/testlog/replaying/%%s.log", localPath),
		"TF_ACC=1",
		"TF_SCHEMA_PANIC_ON_ERROR=1",
	}

	args := []string{"test", googleTestDirectory, "-parallel", accTestParalellism, "-v", "-run=TestAcc", "-timeout", replayingTimeout, `-ldflags="-X=github.com/hashicorp/terraform-provider-google-beta/version.ProviderVersion=acc"`}

	replayingOutput, replayingErr := r.Run("go", args, env)

	if replayingErr != nil {
		if err := r.WriteFile("replaying_test.log", fmt.Sprintf("Error replaying tests: %v", replayingErr)); err != nil {
			fmt.Printf("Error writing replaying log: %v\nError replaying tests: %v\n", err, replayingErr)
		}
	} else {
		if err := r.WriteFile("replaying_test.log", replayingOutput); err != nil {
			fmt.Printf("Error writing replaying log: %v\nTest output: %v\n", err, replayingOutput)
		}
	}

	// Store replaying build log.
	logPath := fmt.Sprintf("gs://ci-vcr-logs/beta/refs/heads/%s/artifacts/%s/", newBranch, opt.buildID)
	if _, err := r.Run("gsutil", []string{"-h", `"Content-Type:text/plain"`, "-q", "cp", "replaying_test.log", logPath + "build-log/"}, nil); err != nil {
		fmt.Println(err)
	}

	// Store replaying test logs.
	if _, err := r.Run("gsutil", []string{"-h", `"Content-Type:text/plain"`, "-m", "-q", "cp", "testlog/replaying/*", logPath + "replaying/"}, nil); err != nil {
		fmt.Println(err)
	}

	var failedTests, passedTests, skippedTests []string
	for _, line := range strings.Split(replayingOutput, "\n") {
		// Handle provider crash.
		if strings.HasPrefix(line, "panic: ") {
			comment := fmt.Sprintf(`$\textcolor{red}{\textsf{The provider crashed while running the VCR tests in REPLAYING mode}}$
$\textcolor{red}{\textsf{Please fix it to complete your PR}}$
View the [build log](https://storage.cloud.google.com/ci-vcr-logs/beta/refs/heads/%s/artifacts/%s/build-log/replaying_test.log)
`, newBranch, opt.buildID)
			if err := gh.PostComment(opt.pr, comment); err != nil {
				fmt.Println("Error posting comment: ", err)
				os.Exit(1)
			}
			if err := gh.PostBuildStatus(opt.pr, title, "failure", targetURL, opt.commit); err != nil {
				fmt.Println("Error posting build status: ", err)
				os.Exit(1)
			}
			os.Exit(0)
		}
		if strings.HasPrefix(line, "--- FAIL: TestAcc") {
			failedTests = append(failedTests, line)
		}
		if strings.HasPrefix(line, "--- PASS: TestAcc") {
			passedTests = append(passedTests, line)
		}
		if strings.HasPrefix(line, "--- SKIP: TestAcc") {
			skippedTests = append(skippedTests, line)
		}
	}

	for _, test := range failedTests {
		testParts := strings.Split(test, " ")
		if len(testParts) < 3 {
			fmt.Println("Failed to parse failed test: ", test)
			continue
		}

	}
	failedTestsPattern := strings.Join(failedTests, "|")

	/*
		function add_comment {
		  curl -H "Authorization: token ${GITHUB_TOKEN}" \
		    -d "$(jq -r --arg comment "${1}" -n "{body: \$comment}")" \
		    "https://api.github.com/repos/GoogleCloudPlatform/magic-modules/issues/${pr_number}/comments"
		}

		function update_status {
		  post_body=$( jq -n \
		    --arg context "VCR-test" \
		    --arg target_url "https://console.cloud.google.com/cloud-build/builds;region=global/${build_id};step=${build_step}?project=${project_id}" \
		    --arg state "${1}" \
		    '{context: $context, target_url: $target_url, state: $state}')

		  curl \
		    -X POST \
		    -u "$github_username:$GITHUB_TOKEN" \
		    -H "Accept: application/vnd.github.v3+json" \
		    "https://api.github.com/repos/GoogleCloudPlatform/magic-modules/statuses/$mm_commit_sha" \
		    -d "$post_body"
		}

		set +e




		TF_LOG=DEBUG TF_LOG_PATH_MASK=$local_path/testlog/replaying/%s.log TF_ACC=1 TF_SCHEMA_PANIC_ON_ERROR=1 go test $GOOGLE_TEST_DIRECTORY -parallel $ACCTEST_PARALLELISM -v -run=TestAcc -timeout 240m -ldflags="-X=github.com/hashicorp/terraform-provider-google-beta/version.ProviderVersion=acc" > replaying_test.log

		test_exit_code=$?




		FAILED_TESTS_PATTERN=$(grep "^--- FAIL: TestAcc" replaying_test$test_suffix.log | awk '{print $3}' | awk -v d="|" '{s=(NR==1?s:s d)$0}END{print s}')

		comment="#### Tests analytics ${NEWLINE}"
		comment+="Total tests: \`$(($FAILED_TESTS_COUNT+$PASSED_TESTS_COUNT+$SKIPPED_TESTS_COUNT))\` ${NEWLINE}"
		comment+="Passed tests \`$PASSED_TESTS_COUNT\` ${NEWLINE}"
		comment+="Skipped tests: \`$SKIPPED_TESTS_COUNT\` ${NEWLINE}"
		comment+="Affected tests: \`$FAILED_TESTS_COUNT\` ${NEWLINE}${NEWLINE}"

		if [[ -n $FAILED_TESTS_PATTERN ]]; then
		  comment+="#### Action taken ${NEWLINE}"
		  comment+="<details> <summary>Found $FAILED_TESTS_COUNT affected test(s) by replaying old test recordings. Starting RECORDING based on the most recent commit. Click here to see the affected tests</summary><blockquote>$FAILED_TESTS_PATTERN </blockquote></details> ${NEWLINE}${NEWLINE}"
		  comment+="[Get to know how VCR tests work](https://googlecloudplatform.github.io/magic-modules/docs/getting-started/contributing/#general-contributing-steps)"
		  add_comment "${comment}"
		  # Clear fixtures folder
		  rm $VCR_PATH/*

		  # Clear replaying-log folder
		  rm testlog/replaying/*

		  # RECORDING mode
		  export VCR_MODE=RECORDING
		  FAILED_TESTS=$(grep "^--- FAIL: TestAcc" replaying_test$test_suffix.log | awk '{print $3}')
		  # test_exit_code=0
		  parallel --jobs 16 TF_LOG=DEBUG TF_LOG_PATH_MASK=$local_path/testlog/recording/%s.log TF_ACC=1 TF_SCHEMA_PANIC_ON_ERROR=1 go test {1} -parallel 1 -v -run="{2}$" -timeout 240m -ldflags="-X=github.com/hashicorp/terraform-provider-google-beta/version.ProviderVersion=acc" ">>" testlog/recording_build/{2}_recording_test.log ::: $GOOGLE_TEST_DIRECTORY ::: $FAILED_TESTS

		  test_exit_code=$?

		  # Concatenate recording build logs to one file
		  # Note: build logs are different from debug logs
		  for failed_test in $FAILED_TESTS
		  do
		    cat testlog/recording_build/${failed_test}_recording_test.log >> recording_test.log
		  done

		  # store cassettes
		  gsutil -m -q cp fixtures/* gs://ci-vcr-cassettes/beta/refs/heads/auto-pr-$pr_number/fixtures/

		  # store recording build log
		  gsutil -h "Content-Type:text/plain" -q cp recording_test.log gs://ci-vcr-logs/beta/refs/heads/auto-pr-$pr_number/artifacts/$build_id/build-log/

		  # store recording individual build logs
		  gsutil -h "Content-Type:text/plain" -m -q cp testlog/recording_build/* gs://ci-vcr-logs/beta/refs/heads/auto-pr-$pr_number/artifacts/$build_id/build-log/recording_build/

		  # store recording test logs
		  gsutil -h "Content-Type:text/plain" -m -q cp testlog/recording/* gs://ci-vcr-logs/beta/refs/heads/auto-pr-$pr_number/artifacts/$build_id/recording/

		  # handle provider crash
		  RECORDING_TESTS_PANIC=$(grep "^panic: " recording_test.log)

		  if [[ -n $RECORDING_TESTS_PANIC ]]; then

		    comment="$\textcolor{red}{\textsf{The provider crashed while running the VCR tests in RECORDING mode}}$ ${NEWLINE}"
		    comment+="$\textcolor{red}{\textsf{Please fix it to complete your PR}}$ ${NEWLINE}"
		    comment+="View the [build log](https://storage.cloud.google.com/ci-vcr-logs/beta/refs/heads/auto-pr-$pr_number/artifacts/$build_id/build-log/recording_test.log)"
		    add_comment "${comment}"
		    update_status "failure"
		    exit 0
		  fi


		  RECORDING_FAILED_TESTS=$(grep "^--- FAIL: TestAcc" recording_test.log | awk -v pr_number=$pr_number -v build_id=$build_id '{print "`"$3"`[[Error message](https://storage.cloud.google.com/ci-vcr-logs/beta/refs/heads/auto-pr-"pr_number"/artifacts/"build_id"/build-log/recording_build/"$3"_recording_test.log)] [[Debug log](https://storage.cloud.google.com/ci-vcr-logs/beta/refs/heads/auto-pr-"pr_number"/artifacts/"build_id"/recording/"$3".log)]"}')
		  RECORDING_PASSED_TESTS=$(grep "^--- PASS: TestAcc" recording_test.log | awk -v pr_number=$pr_number -v build_id=$build_id '{print "`"$3"`[[Debug log](https://storage.cloud.google.com/ci-vcr-logs/beta/refs/heads/auto-pr-"pr_number"/artifacts/"build_id"/recording/"$3".log)]"}')
		  RECORDING_PASSED_TEST_LIST=$(grep "^--- PASS: TestAcc" recording_test.log | awk '{print $3}')

		  comment=""
		  RECORDING_PASSED_TESTS_COUNT=0
		  RECORDING_FAILED_TESTS_COUNT=0
		  if [[ -n $RECORDING_PASSED_TESTS ]]; then
		    comment+="$\textcolor{green}{\textsf{Tests passed during RECORDING mode:}}$ ${NEWLINE} $RECORDING_PASSED_TESTS ${NEWLINE}${NEWLINE}"
		    RECORDING_PASSED_TESTS_COUNT=$(echo "$RECORDING_PASSED_TESTS" | wc -l)
		    comment+="##### Rerun these tests in REPLAYING mode to catch issues ${NEWLINE}${NEWLINE}"

		    # Rerun passed tests in REPLAYING mode 3 times to catch issues
		    export VCR_MODE=REPLAYING
		    count=3
		    parallel --jobs 16 TF_LOG=DEBUG TF_LOG_PATH_MASK=$local_path/testlog/replaying_after_recording/%s.log TF_ACC=1 TF_SCHEMA_PANIC_ON_ERROR=1 go test {1} -parallel 1 -count=$count -v -run="{2}$" -timeout 120m -ldflags="-X=github.com/hashicorp/terraform-provider-google-beta/version.ProviderVersion=acc" ">>" testlog/replaying_build_after_recording/{2}_replaying_test.log ::: $GOOGLE_TEST_DIRECTORY ::: $RECORDING_PASSED_TEST_LIST

		    test_exit_code=$(($test_exit_code || $?))

		    # Concatenate recording build logs to one file
		    for test in $RECORDING_PASSED_TEST_LIST
		    do
		      cat testlog/replaying_build_after_recording/${test}_replaying_test.log >> replaying_build_after_recording.log
		    done

		    # store replaying individual build logs
		    gsutil -h "Content-Type:text/plain" -m -q cp testlog/replaying_build_after_recording/* gs://ci-vcr-logs/beta/refs/heads/auto-pr-$pr_number/artifacts/$build_id/build-log/replaying_build_after_recording/

		    # store replaying test logs
		    gsutil -h "Content-Type:text/plain" -m -q cp testlog/replaying_after_recording/* gs://ci-vcr-logs/beta/refs/heads/auto-pr-$pr_number/artifacts/$build_id/replaying_after_recording/

		    REPLAYING_FAILED_TESTS=$(grep "^--- FAIL: TestAcc" replaying_build_after_recording.log | sort -u -t' ' -k3,3 | awk -v pr_number=$pr_number -v build_id=$build_id '{print "`"$3"`[[Error message](https://storage.cloud.google.com/ci-vcr-logs/beta/refs/heads/auto-pr-"pr_number"/artifacts/"build_id"/build-log/replaying_build_after_recording/"$3"_replaying_test.log)] [[Debug log](https://storage.cloud.google.com/ci-vcr-logs/beta/refs/heads/auto-pr-"pr_number"/artifacts/"build_id"/replaying_after_recording/"$3".log)]"}')
		    if [[ -n $REPLAYING_FAILED_TESTS ]]; then
		      comment+="$\textcolor{red}{\textsf{Tests failed when rerunning REPLAYING mode:}}$ ${NEWLINE} $REPLAYING_FAILED_TESTS ${NEWLINE}${NEWLINE}"
		      comment+="Tests failed due to non-determinism or randomness when the VCR replayed the response after the HTTP request was made.${NEWLINE}${NEWLINE}"
		      comment+="Please fix these to complete your PR. If you believe these test failures to be incorrect or unrelated to your change, or if you have any questions, please raise the concern with your reviewer.${NEWLINE}"
		    else
		      comment+="$\textcolor{green}{\textsf{No issues found for passed tests after REPLAYING rerun.}}$ ${NEWLINE}"
		    fi
		    comment+="${NEWLINE}---${NEWLINE}"

		    # Clear replaying-log folder
		    rm testlog/replaying_after_recording/*
		    rm testlog/replaying_build_after_recording/*
		  fi

		  if [[ -n $RECORDING_FAILED_TESTS ]]; then
		    comment+="$\textcolor{red}{\textsf{Tests failed during RECORDING mode:}}$ ${NEWLINE} $RECORDING_FAILED_TESTS ${NEWLINE}${NEWLINE}"
		    RECORDING_FAILED_TESTS_COUNT=$(echo "$RECORDING_FAILED_TESTS" | wc -l)
		    if [[ $RECORDING_PASSED_TESTS_COUNT+$RECORDING_FAILED_TESTS_COUNT -lt $FAILED_TESTS_COUNT ]]; then
		      comment+="$\textcolor{red}{\textsf{Several tests got terminated during RECORDING mode.}}$ ${NEWLINE}"
		    fi
		    comment+="$\textcolor{red}{\textsf{Please fix these to complete your PR.}}$ ${NEWLINE}"
		  else
		    if [[ $RECORDING_PASSED_TESTS_COUNT+$RECORDING_FAILED_TESTS_COUNT -lt $FAILED_TESTS_COUNT ]]; then
		      comment+="$\textcolor{red}{\textsf{Several tests got terminated during RECORDING mode.}}$ ${NEWLINE}"
		    elif [[ $test_exit_code -ne 0 ]]; then
		      # check for any uncaught errors in RECORDING mode
		      comment+="$\textcolor{red}{\textsf{Errors occurred during RECORDING mode. Please fix them to complete your PR.}}$ ${NEWLINE}"
		    else
		      comment+="$\textcolor{green}{\textsf{All tests passed!}}$ ${NEWLINE}"
		    fi
		  fi

		  comment+="View the [build log](https://storage.cloud.google.com/ci-vcr-logs/beta/refs/heads/auto-pr-$pr_number/artifacts/$build_id/build-log/recording_test.log) or the [debug log](https://console.cloud.google.com/storage/browser/ci-vcr-logs/beta/refs/heads/auto-pr-$pr_number/artifacts/$build_id/recording) for each test"
		  add_comment "${comment}"

		else
		  if [[ $test_exit_code -ne 0 ]]; then
		    # check for any uncaught errors errors in REPLAYING mode
		    comment+="$\textcolor{red}{\textsf{Errors occurred during REPLAYING mode. Please fix them to complete your PR}}$ ${NEWLINE}"
		  else
		    comment+="$\textcolor{green}{\textsf{All tests passed in REPLAYING mode.}}$ ${NEWLINE}"
		  fi
		  comment+="View the [build log](https://storage.cloud.google.com/ci-vcr-logs/beta/refs/heads/auto-pr-$pr_number/artifacts/$build_id/build-log/replaying_test$test_suffix.log)"
		  add_comment "${comment}"
		fi


		if [[ $test_exit_code -ne 0 ]]; then
		  test_state="failure"
		else
		  test_state="success"
		fi

		set -e

		update_status ${test_state}
	*/
}

func goFilesChanged() bool {
}
