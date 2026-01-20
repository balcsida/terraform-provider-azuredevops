package acceptancetests

import (
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/microsoft/terraform-provider-azuredevops/azuredevops/internal/acceptancetests/testutils"
	"github.com/microsoft/terraform-provider-azuredevops/azuredevops/internal/client"
	"github.com/microsoft/terraform-provider-azuredevops/azuredevops/utils/sdk/securefiles"
)

// TestAccSecureFile_basic verifies that a secure file can be created and updated
func TestAccSecureFile_basic(t *testing.T) {
	projectName := testutils.GenerateResourceName()
	secureFileName := testutils.GenerateResourceName()
	tfNode := "azuredevops_secure_file.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testutils.PreCheck(t, nil) },
		Providers:    testutils.GetProviders(),
		CheckDestroy: checkSecureFileDestroyed,
		Steps: []resource.TestStep{
			{
				Config: hclSecureFileResource(projectName, secureFileName, false),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(tfNode, "name", secureFileName),
					resource.TestCheckResourceAttr(tfNode, "authorize", "false"),
					resource.TestCheckResourceAttrSet(tfNode, "project_id"),
					checkSecureFileExists(secureFileName),
				),
			},
			{
				// Update authorize to true
				Config: hclSecureFileResource(projectName, secureFileName, true),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(tfNode, "name", secureFileName),
					resource.TestCheckResourceAttr(tfNode, "authorize", "true"),
					checkSecureFileExists(secureFileName),
				),
			},
		},
	})
}

// TestAccSecureFile_withProperties verifies that a secure file with properties can be created
func TestAccSecureFile_withProperties(t *testing.T) {
	projectName := testutils.GenerateResourceName()
	secureFileName := testutils.GenerateResourceName()
	tfNode := "azuredevops_secure_file.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testutils.PreCheck(t, nil) },
		Providers:    testutils.GetProviders(),
		CheckDestroy: checkSecureFileDestroyed,
		Steps: []resource.TestStep{
			{
				Config: hclSecureFileResourceWithProperties(projectName, secureFileName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(tfNode, "name", secureFileName),
					resource.TestCheckResourceAttr(tfNode, "properties.key1", "value1"),
					checkSecureFileExists(secureFileName),
				),
			},
		},
	})
}

func checkSecureFileExists(expectedName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		res, ok := s.RootModule().Resources["azuredevops_secure_file.test"]
		if !ok {
			return fmt.Errorf("Did not find a secure file in the TF state")
		}

		clients := testutils.GetProvider().Meta().(*client.AggregatedClient)
		secureFileId, err := uuid.Parse(res.Primary.ID)
		if err != nil {
			return fmt.Errorf("Parse ID error, ID: %v. Error= %v", res.Primary.ID, err)
		}
		projectID := res.Primary.Attributes["project_id"]

		secureFile, err := clients.SecureFilesClient.GetSecureFile(clients.Ctx, securefiles.GetSecureFileArgs{
			Project:      &projectID,
			SecureFileId: &secureFileId,
		})
		if err != nil {
			return fmt.Errorf("Secure file with ID=%s cannot be found. Error=%v", res.Primary.ID, err)
		}

		if *secureFile.Name != expectedName {
			return fmt.Errorf("Secure file with ID=%s has Name=%s, but expected Name=%s", res.Primary.ID, *secureFile.Name, expectedName)
		}

		return nil
	}
}

func checkSecureFileDestroyed(s *terraform.State) error {
	clients := testutils.GetProvider().Meta().(*client.AggregatedClient)

	for _, res := range s.RootModule().Resources {
		if res.Type != "azuredevops_secure_file" {
			continue
		}

		secureFileId, err := uuid.Parse(res.Primary.ID)
		if err != nil {
			return fmt.Errorf("Secure file ID=%s cannot be parsed. Error=%v", res.Primary.ID, err)
		}
		projectID := res.Primary.Attributes["project_id"]

		if _, err := clients.SecureFilesClient.GetSecureFile(clients.Ctx, securefiles.GetSecureFileArgs{
			Project:      &projectID,
			SecureFileId: &secureFileId,
		}); err == nil {
			return fmt.Errorf("Secure file ID %s should not exist", res.Primary.ID)
		}
	}

	return nil
}

func hclSecureFileResource(projectName string, secureFileName string, authorize bool) string {
	projectResource := testutils.HclProjectResource(projectName)
	return fmt.Sprintf(`
%s

resource "azuredevops_secure_file" "test" {
  project_id = azuredevops_project.project.id
  name       = "%s"
  content    = base64encode("test content for secure file")
  authorize  = %t
}
`, projectResource, secureFileName, authorize)
}

func hclSecureFileResourceWithProperties(projectName string, secureFileName string) string {
	projectResource := testutils.HclProjectResource(projectName)
	return fmt.Sprintf(`
%s

resource "azuredevops_secure_file" "test" {
  project_id = azuredevops_project.project.id
  name       = "%s"
  content    = base64encode("test content for secure file")
  properties = {
    key1 = "value1"
  }
}
`, projectResource, secureFileName)
}
