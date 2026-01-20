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

// TestAccIteration_basic verifies that an iteration can be created and updated
func TestAccIteration_basic(t *testing.T) {
	projectName := testutils.GenerateResourceName()
	iterationNameFirst := testutils.GenerateResourceName()
	iterationNameSecond := testutils.GenerateResourceName()
	tfNode := "azuredevops_iteration.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testutils.PreCheck(t, nil) },
		Providers:    testutils.GetProviders(),
		CheckDestroy: checkIterationDestroyed,
		Steps: []resource.TestStep{
			{
				Config: hclIterationResource(projectName, iterationNameFirst),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(tfNode, "name", iterationNameFirst),
					resource.TestCheckResourceAttrSet(tfNode, "project_id"),
					checkIterationExists(iterationNameFirst),
				),
			},
			{
				Config: hclIterationResource(projectName, iterationNameSecond),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(tfNode, "name", iterationNameSecond),
					checkIterationExists(iterationNameSecond),
				),
			},
		},
	})
}

// TestAccIteration_withDates verifies that an iteration with dates can be created
func TestAccIteration_withDates(t *testing.T) {
	projectName := testutils.GenerateResourceName()
	iterationName := testutils.GenerateResourceName()
	tfNode := "azuredevops_iteration.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testutils.PreCheck(t, nil) },
		Providers:    testutils.GetProviders(),
		CheckDestroy: checkIterationDestroyed,
		Steps: []resource.TestStep{
			{
				Config: hclIterationResourceWithDates(projectName, iterationName, "2024-01-01T00:00:00Z", "2024-01-14T00:00:00Z"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(tfNode, "name", iterationName),
					resource.TestCheckResourceAttr(tfNode, "start_date", "2024-01-01T00:00:00Z"),
					resource.TestCheckResourceAttr(tfNode, "finish_date", "2024-01-14T00:00:00Z"),
					checkIterationExists(iterationName),
				),
			},
		},
	})
}

// TestAccIteration_nested verifies that a nested iteration can be created
func TestAccIteration_nested(t *testing.T) {
	projectName := testutils.GenerateResourceName()
	parentIterationName := testutils.GenerateResourceName()
	childIterationName := testutils.GenerateResourceName()
	tfNodeParent := "azuredevops_iteration.parent"
	tfNodeChild := "azuredevops_iteration.child"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testutils.PreCheck(t, nil) },
		Providers:    testutils.GetProviders(),
		CheckDestroy: checkIterationDestroyed,
		Steps: []resource.TestStep{
			{
				Config: hclIterationResourceNested(projectName, parentIterationName, childIterationName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(tfNodeParent, "name", parentIterationName),
					resource.TestCheckResourceAttr(tfNodeChild, "name", childIterationName),
					checkIterationExists(parentIterationName),
				),
			},
		},
	})
}

func checkIterationExists(expectedName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		for _, res := range s.RootModule().Resources {
			if res.Type != "azuredevops_iteration" {
				continue
			}
			if res.Primary.Attributes["name"] != expectedName {
				continue
			}

			clients := testutils.GetProvider().Meta().(*client.AggregatedClient)
			projectID := res.Primary.Attributes["project_id"]

			structureGroup := workitemtracking.TreeStructureGroupValues.Iterations
			_, err := clients.WorkItemTrackingClient.GetClassificationNode(clients.Ctx, workitemtracking.GetClassificationNodeArgs{
				Project:        &projectID,
				StructureGroup: &structureGroup,
				Path:           converter.String(expectedName),
				Depth:          converter.Int(0),
			})
			if err != nil {
				return fmt.Errorf("Iteration with name=%s cannot be found. Error=%v", expectedName, err)
			}

			return nil
		}

		return fmt.Errorf("Did not find an iteration with name=%s in the TF state", expectedName)
	}
}

func checkIterationDestroyed(s *terraform.State) error {
	clients := testutils.GetProvider().Meta().(*client.AggregatedClient)

	for _, res := range s.RootModule().Resources {
		if res.Type != "azuredevops_iteration" {
			continue
		}

		projectID := res.Primary.Attributes["project_id"]
		iterationName := res.Primary.Attributes["name"]

		structureGroup := workitemtracking.TreeStructureGroupValues.Iterations
		if _, err := clients.WorkItemTrackingClient.GetClassificationNode(clients.Ctx, workitemtracking.GetClassificationNodeArgs{
			Project:        &projectID,
			StructureGroup: &structureGroup,
			Path:           &iterationName,
			Depth:          converter.Int(0),
		}); err == nil {
			return fmt.Errorf("Iteration %s should not exist", iterationName)
		}
	}

	return nil
}

func hclIterationResource(projectName string, iterationName string) string {
	projectResource := testutils.HclProjectResource(projectName)
	return fmt.Sprintf(`
%s

resource "azuredevops_iteration" "test" {
  project_id = azuredevops_project.project.id
  name       = "%s"
}
`, projectResource, iterationName)
}

func hclIterationResourceWithDates(projectName string, iterationName string, startDate string, finishDate string) string {
	projectResource := testutils.HclProjectResource(projectName)
	return fmt.Sprintf(`
%s

resource "azuredevops_iteration" "test" {
  project_id  = azuredevops_project.project.id
  name        = "%s"
  start_date  = "%s"
  finish_date = "%s"
}
`, projectResource, iterationName, startDate, finishDate)
}

func hclIterationResourceNested(projectName string, parentIterationName string, childIterationName string) string {
	projectResource := testutils.HclProjectResource(projectName)
	return fmt.Sprintf(`
%s

resource "azuredevops_iteration" "parent" {
  project_id = azuredevops_project.project.id
  name       = "%s"
}

resource "azuredevops_iteration" "child" {
  project_id = azuredevops_project.project.id
  name       = "%s"
  path       = "/${azuredevops_iteration.parent.name}"

  depends_on = [azuredevops_iteration.parent]
}
`, projectResource, parentIterationName, childIterationName)
}
