package acceptancetests

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/workitemtracking"
	"github.com/microsoft/terraform-provider-azuredevops/azuredevops/internal/acceptancetests/testutils"
	"github.com/microsoft/terraform-provider-azuredevops/azuredevops/internal/client"
	"github.com/microsoft/terraform-provider-azuredevops/azuredevops/internal/utils/converter"
)

// TestAccArea_basic verifies that an area can be created and updated
func TestAccArea_basic(t *testing.T) {
	projectName := testutils.GenerateResourceName()
	areaNameFirst := testutils.GenerateResourceName()
	areaNameSecond := testutils.GenerateResourceName()
	tfNode := "azuredevops_area.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testutils.PreCheck(t, nil) },
		Providers:    testutils.GetProviders(),
		CheckDestroy: checkAreaDestroyed,
		Steps: []resource.TestStep{
			{
				Config: hclAreaResource(projectName, areaNameFirst),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(tfNode, "name", areaNameFirst),
					resource.TestCheckResourceAttrSet(tfNode, "project_id"),
					checkAreaExists(areaNameFirst),
				),
			},
			{
				Config: hclAreaResource(projectName, areaNameSecond),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(tfNode, "name", areaNameSecond),
					checkAreaExists(areaNameSecond),
				),
			},
		},
	})
}

// TestAccArea_nested verifies that a nested area can be created
func TestAccArea_nested(t *testing.T) {
	projectName := testutils.GenerateResourceName()
	parentAreaName := testutils.GenerateResourceName()
	childAreaName := testutils.GenerateResourceName()
	tfNodeParent := "azuredevops_area.parent"
	tfNodeChild := "azuredevops_area.child"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testutils.PreCheck(t, nil) },
		Providers:    testutils.GetProviders(),
		CheckDestroy: checkAreaDestroyed,
		Steps: []resource.TestStep{
			{
				Config: hclAreaResourceNested(projectName, parentAreaName, childAreaName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(tfNodeParent, "name", parentAreaName),
					resource.TestCheckResourceAttr(tfNodeChild, "name", childAreaName),
					checkAreaExists(parentAreaName),
				),
			},
		},
	})
}

func checkAreaExists(expectedName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		for _, res := range s.RootModule().Resources {
			if res.Type != "azuredevops_area" {
				continue
			}
			if res.Primary.Attributes["name"] != expectedName {
				continue
			}

			clients := testutils.GetProvider().Meta().(*client.AggregatedClient)
			projectID := res.Primary.Attributes["project_id"]

			structureGroup := workitemtracking.TreeStructureGroupValues.Areas
			_, err := clients.WorkItemTrackingClient.GetClassificationNode(clients.Ctx, workitemtracking.GetClassificationNodeArgs{
				Project:        &projectID,
				StructureGroup: &structureGroup,
				Path:           converter.String(expectedName),
				Depth:          converter.Int(0),
			})
			if err != nil {
				return fmt.Errorf("Area with name=%s cannot be found. Error=%v", expectedName, err)
			}

			return nil
		}

		return fmt.Errorf("Did not find an area with name=%s in the TF state", expectedName)
	}
}

func checkAreaDestroyed(s *terraform.State) error {
	clients := testutils.GetProvider().Meta().(*client.AggregatedClient)

	for _, res := range s.RootModule().Resources {
		if res.Type != "azuredevops_area" {
			continue
		}

		projectID := res.Primary.Attributes["project_id"]
		areaName := res.Primary.Attributes["name"]

		structureGroup := workitemtracking.TreeStructureGroupValues.Areas
		if _, err := clients.WorkItemTrackingClient.GetClassificationNode(clients.Ctx, workitemtracking.GetClassificationNodeArgs{
			Project:        &projectID,
			StructureGroup: &structureGroup,
			Path:           &areaName,
			Depth:          converter.Int(0),
		}); err == nil {
			return fmt.Errorf("Area %s should not exist", areaName)
		}
	}

	return nil
}

func hclAreaResource(projectName string, areaName string) string {
	projectResource := testutils.HclProjectResource(projectName)
	return fmt.Sprintf(`
%s

resource "azuredevops_area" "test" {
  project_id = azuredevops_project.project.id
  name       = "%s"
}
`, projectResource, areaName)
}

func hclAreaResourceNested(projectName string, parentAreaName string, childAreaName string) string {
	projectResource := testutils.HclProjectResource(projectName)
	return fmt.Sprintf(`
%s

resource "azuredevops_area" "parent" {
  project_id = azuredevops_project.project.id
  name       = "%s"
}

resource "azuredevops_area" "child" {
  project_id = azuredevops_project.project.id
  name       = "%s"
  path       = "/${azuredevops_area.parent.name}"

  depends_on = [azuredevops_area.parent]
}
`, projectResource, parentAreaName, childAreaName)
}
