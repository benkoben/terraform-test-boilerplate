#!/usr/bin/env bash

set -o nounset
set -o pipefail

blue='\033[0;33m'
nc='\033[0m'

# Settings regarding the environment
script_dir=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" &>/dev/null && pwd)
target_module_dir=$(realpath "${script_dir}/..")
go_executable=$(which go)

# Edit the following variables to change the behaviour of the tests
dry_tests_prefix='TestDry_'
dry_tests_timeout='15m'

unit_tests_prefix='TestUT_'
unit_tests_timeout='15m'

integration_tests_prefix='TestIT_'
integration_tests_timeout='30m'

# When a test fails the script needs to take care of cleaning up terraform resources
# This is primarily useful when running this script locally. 
handle_error(){
  echo "Test failed"
  echo "cleaning up before exiting"
  cleanup
  
  exit 1
}

usage() {
  echo 'Usage: ./test.sh -m MODE <full|unit|integration|dry>'
  exit 1
}

full() {
  dry
  unit
  integration
}

dry() {
  # This build step performs a Unit test. Which implied an init and plan.
  # The plan file is compared to the expected output of all tests. If no diffs are detected
  # the test succeeds.
  format
  printf "\nRunning%btests....\n" "${blue} dry-run ${nc}"
  "${go_executable}" test -count=1 "${script_dir}" -run "${dry_tests_prefix}" -v -timeout "${dry_tests_timeout}"
}


unit() {
  # This build step performs a Unit test. Which implied an init and plan.
  # The plan file is compared to the expected output of all tests. If no diffs are detected
  # the test succeeds.
  format
  printf "\nRunning%btests....\n" "${blue} unit ${nc}"
  "${go_executable}" test -count=1 "${script_dir}" -run "${unit_tests_prefix}" -v -timeout "${unit_tests_timeout}"
}

integration() {
  # Build setp that performs an integration test. This implies that a terraform apply will be exected.
  format
  printf "\nRunning%btests....\n" "${blue} integration ${nc}"
  "${go_executable}" test -count=1 "${script_dir}" -run "${integration_tests_prefix}" -v -timeout "${integration_tests_timeout}"
}

format() {
  # Build step that format both Terraform code and Go code.
  # This will catch any code that has invalid content.
  printf "\nFormatting...\n"

  if ! terraform fmt "${target_module_dir}"; then
    printf "could not format terraform templates correclty. exiting...\n"
    exit 1
  fi

  if ! go fmt "${script_dir}"; then
    printf "could not format tests correclty. exiting...\n"
    exit 1
  fi
}

cleanup() {
  # Cleanup any files created by terraform
  printf "\nCleaning up...\n"
  find "${target_module_dir}" | while read -r file; do
    basename_obj=$(basename "${file}")
    # remove directories created by terraform
    if [[ -d ${file} ]] && [[ ${basename_obj} == ".terraform" ]]; then
      rm -rf "${file}"
      printf "removed %s\n" "${file}"
    fi

    # remove files created by terraform
    if [[ -f ${file} ]] && [[ ${basename_obj} == "terraform.tfstate" || ${basename_obj} == *.tfplan || ${basename_obj} == "terraform.tfstate.backup" ]]; then
      rm -rf "${file}"
      printf "removed %s\n" "${file}"
    fi

    # remove files created by the tests
    if [[ -f ${file} ]] && [[ ${basename_obj} == "features.tf" || ${basename_obj} == "provider.tf" ]]; then
      rm -rf "${file}"
      printf "removed %s\n" "${file}"
    fi
  done
}

if [[ ${TRACE-0} == "1" ]]; then
  set -o xtrace
fi

# Detect if first argument is '-h'/'help' OR if first argument is empty
# then print usage and exit
if [[ ${1-} =~ ^-*h(elp)?$ ]] || [[ -z ${1-} ]]; then
  usage
fi

if [[ -z ${go_executable} ]]; then
  printf "unable to locate go binary, make sure go is installed. Exiting...\n"
  exit 1
fi

trap "handle_error" ERR

while getopts ":m:" o; do
  case "${o}" in
  m)
    m=${OPTARG}
    case ${m} in
    'full')
      full
      ;;
    'unit')
      unit
      ;;
    'integration')
      integration
      ;;
    'dry')
      dry
      ;;
    *)
      usage
      ;;
    esac
    ;;
  *)
    usage
    ;;
  esac
done

printf "\nFinished :)\n"

exit 0
