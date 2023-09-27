package tests

import (
	"fmt"
	"os"
	"os/exec"

	tfjson "github.com/hashicorp/terraform-json"

	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclwrite"
)

const (
	DependencyTemplates = "dependencies/"
)

type Features struct{}

type AzureRm struct {
	Name     string   `hcl:"name,label"`
	Features Features `hcl:"features,block"`
}

type Provider struct {
	path     *os.File
	Provider AzureRm `hcl:"provider,block"`
}

func (p Provider) Create() error {
	defer p.path.Close()

	f := hclwrite.NewEmptyFile()
	gohcl.EncodeIntoBody(&p, f.Body())

	p.path.Write(f.Bytes())
	return nil
}

func (p Provider) Delete() {
	_ = os.Remove(p.path.Name())
}

func NewProvider(path string) (Provider, error) {

	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return Provider{}, fmt.Errorf("could not open provided path: %s", err)
	}
	return Provider{
		path: f,
		Provider: AzureRm{
			Name:     "azurerm",
			Features: Features{},
		},
	}, nil
}

func LocateTerraformExec() string {
	tfPath, err := exec.LookPath("terraform")
	if err != nil {
		fmt.Printf("lookup terraform binary: %s\n", err)
		os.Exit(1)
	}
	return tfPath
}

func ParseResourceAddresses(plan *tfjson.Plan) []string {
	var addreses []string
	for _, resource := range plan.ResourceChanges {
		addreses = append(addreses, resource.Address)
	}
	return addreses
}
