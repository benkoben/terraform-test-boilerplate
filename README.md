# terraform-test-boilerplate

This repository contains templates to use when creating new tests for terraform modules. It contains:
1. Example tests written that show example of table driven tests for dry-running, unit testing and integration testing terraform modules.
2. Helper functions that could be useful for locating terraform binaries, parsing plan outputs and generating provider blocks.
3. A shell script that wraps the tests into a friendly use format.
4. Github actions workflows

**Note**
> This repository contains a workflow file that can execute *_test.go files. However, should use these boilerplates in an pre-existing repository there is a good chance that a workflow is already in place. Should that be the case then feel free to remove the .github folder from the project directory.

# Getting started

To create a new project based on the one of the templates:

1. Install `gonew` (if is not already installed):

```
go install golang.org/x/tools/cmd/gonew@latest
```

2. Download the template and create the new project

```
# Assuming the project will be hosted in GitHub.
# If not replace github.com/<owner>/<repo> with the correct path.
gonew github.com/benkoben/terraform-test-boilerplate github.com/<owner>/<repo>
```

# Shell script

The shell script is used by the github action workflow but can however also be used locally.

Use the `-h` to list options

Tests can be customized by modifiying the following variables:
```
unit_tests_prefix='TestDry_'
unit_tests_timeout='15m'

unit_tests_prefix='TestUT_'
unit_tests_timeout='15m'

integration_tests_prefix='TestIT_'
integration_tests_timeout='30m'
```

### Running locally

Useful if you dont want to wait for pipelines executing tests. The script does by default not use caching for go tests and all terraform files created during tests will be cleaned up between runs no matter if the test succeeds or fails.