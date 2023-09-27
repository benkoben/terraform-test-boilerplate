package tests

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/hashicorp/terraform-exec/tfexec"
)

// Testing scenarios are defined with functions such as this one.
// Usually TerraformDir, Reconfigure and Upgrade can be left as is. Reference docs can be found here: https://pkg.go.dev/github.com/gruntwork-io/terratest@v0.44.0/modules/terraform#Options
// Vars is the input that the module expects. These are equivalent for a tfvars file or using TF_VAR environment variables.
func mockModuleInput(t *testing.T) *terraform.Options {
	return &terraform.Options{
		TerraformDir: "../.",
		Reconfigure:  true,
		Upgrade:      true,
		Vars: map[string]interface{}{
			"location":            "northeurope",
			"resource_group_name": "rg-test",
		},
	}
}

// --- Dry-runs
// The following function is designed to only perform dry-runs on isolated modules.
//
// The below example performs terraform init, validate & plan on each test in tests[].
// the plan is saved before all resources are parsed, which are then in turn compared with test.want.
// if the plan matches test.want then tests pass.
func TestDry_Example(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   *terraform.Options
		want    []string
		options struct {
			planOut string
		}
	}{
		{
			name:  "simple-isolated-network-eun",
			input: mockModuleInput(t),
			// The order of the elements is important
			// Tip: Run a test to see the order of the resources
			want: []string{},
			options: struct{ planOut string }{
				planOut: "mockModule.tfplan",
			},
		},
	}

	for _, test := range tests {
		// Runs each test in the tests table as a subset of the unit test.
		// Each test is run as an individual goroutine.
		t.Run(test.name, func(t *testing.T) {
			provider, err := NewProvider(test.input.TerraformDir + "/provider.tf")
			if err != nil {
				t.Fatal(err)
			}
			defer provider.Delete()
			provider.Create()

			tf, err := tfexec.NewTerraform(test.input.TerraformDir, LocateTerraformExec())
			if err != nil {
				t.Fatal(err)
			}
			terraform.Init(t, test.input)

			// Run
			validateJson, err := tf.Validate(context.Background())
			if err != nil {
				t.Fatal(err)
			}

			if !validateJson.Valid || validateJson.WarningCount > 0 || validateJson.ErrorCount > 0 {
				for _, diagnostic := range validateJson.Diagnostics {
					msg, err := json.Marshal(diagnostic)
					if err != nil {
						t.Fatal(err)
					}
					t.Log(fmt.Printf("%s", msg))
				}
				t.Fatalf("configuration is not valid")
			}

			// Create plan outfile
			_, err = terraform.RunTerraformCommandE(t, test.input, terraform.FormatArgs(test.input, "plan", "-out="+test.options.planOut)...)
			if err != nil {
				t.Fatal(err)
			}

			// Read plan file as json
			planJson, err := tf.ShowPlanFile(context.Background(), test.options.planOut)
			if err != nil {
				t.Fatal(err)
			}
			got := ParseResourceAddresses(planJson)

			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Fatalf("%s = Unexpected result, (-want, +got)\n%s\n", test.name, diff)
			}
		})
	}
}

// --- Unit tests
// Test_UT stands for unit test. Which tests modules isolated from each other.
// Some modules might not support unit tests because of infrastructure dependencies (deploying multiple modules), feel free to use the TestIT_Example function instead.
//
// The following example function runs isolated functionality tests on each test case in tests[]. Notice that the same mockModule function is used as test.input.
// For each iteration of tests, terraform init, apply x 2 (idempotency checking) and terraform destroy is run. The destroy is deffered to make sure that destroy always runs
// even if the apply failed.
func TestUT_Example(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input *terraform.Options
	}{
		{
			name:  "simple-unittest-example",
			input: mockModuleInput(t),
		},
	}

	for _, test := range tests {
		provider, err := NewProvider(test.input.TerraformDir + "/provider.tf")
		if err != nil {
			t.Fatal(err)
		}
		defer provider.Delete()
		provider.Create()

		t.Run(test.name, func(t *testing.T) {
			defer terraform.Destroy(t, test.input)
			terraform.Init(t, test.input)
			terraform.ApplyAndIdempotent(t, test.input)
		})
	}
}

// --- Integration tests
// Below two new mockings are created to demonstrate how an integration test can be composed.
// Lets image that mockModuleB depends on mockModuleA (think of a virtual machine module depending on a network module)
//
// First define the input for all modules that are required for an integration test.
// Notice the value of terraform.Options.TerraformDir. It will read from another terraform module directory
func mockModuleA(t *testing.T) *terraform.Options {
	return &terraform.Options{
		TerraformDir: "../../../terraform-azurerm-network-isolated",
		Reconfigure:  true,
		Upgrade:      true,
		Vars: map[string]interface{}{
			"location":            "northeurope",
			"resource_group_name": "rg-network",
		},
	}
}

// Then create the next inputs for mockModuleB.
// Notice the value of terraform.Options.TerraformDir. It will read from the current terraform module directory
func mockModuleB(t *testing.T) *terraform.Options {
	return &terraform.Options{
		TerraformDir: "../.",
		Reconfigure:  true,
		Upgrade:      true,
		Vars: map[string]interface{}{
			"location":            "northeurope",
			"resource_group_name": "rg-vm",
			"subnet_id":           "xxx",
		},
	}
}

// Below we define all unit tests. The following example only shows one single test.
// However, notice how test[0].input is contains both mockModuleA and mockModuleB
func TestIT_Example(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input []*terraform.Options
	}{
		{
			name:  "basic-integration-test",
			input: []*terraform.Options{mockModuleA(t), mockModuleB(t)},
		},
	}

	for _, test := range tests {

		t.Run(test.name, func(t *testing.T) {

			// Create provider.tf files in each module involved in this test.
			// This is because azurerm provider requires a provider azurerm block containing with the features{} block to be present.
			networkProvider, err := NewProvider(test.input[0].TerraformDir + "/provider.tf")
			if err != nil {
				t.Fatal(err)
			}
			defer networkProvider.Delete()
			networkProvider.Create()

			aksProvider, err := NewProvider(test.input[1].TerraformDir + "/provider.tf")
			if err != nil {
				t.Fatal(err)
			}
			defer aksProvider.Delete()
			aksProvider.Create()

			// Define the variables used to to handover outputs/inputs
			var mockModuleAOutput string
			var mockModuleBInput map[string]interface{}

			// Deploy mockModuleA resources
			defer terraform.Destroy(t, test.input[0])
			terraform.InitAndApply(t, test.input[0])

			// Fetch output from  module. The JSON from mockModuleA is converted into a go native datatype map[string]interface{}
			mockModuleAOutput = terraform.OutputJson(t, test.input[0], "")
			err = json.Unmarshal([]byte(mockModuleAOutput), &mockModuleBInput)
			if err != nil {
				t.Fatal("error - could not unmarshal output from dependencies deployment")
			}

			// Gather the subnet_id from the variable mockModuleB input of type map[string]interface{}
			vnetId := mockModuleBInput["subnet_id"].(map[string]interface{})["value"].(string)

			// Replace the value of mockModuleB.Vars["subnet_id"] with the subnet_id received from mockModuleA
			test.input[1].Vars["virtual_network_id"] = vnetId

			// Deploy AKS to network
			defer terraform.Destroy(t, test.input[1])
			terraform.Init(t, test.input[1])
			terraform.ApplyAndIdempotent(t, test.input[1])

		})
	}
}
