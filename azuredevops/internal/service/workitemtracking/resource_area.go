package workitemtracking

import (
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/workitemtracking"
	"github.com/microsoft/terraform-provider-azuredevops/azuredevops/internal/client"
	"github.com/microsoft/terraform-provider-azuredevops/azuredevops/internal/utils"
	"github.com/microsoft/terraform-provider-azuredevops/azuredevops/internal/utils/converter"
	"github.com/microsoft/terraform-provider-azuredevops/azuredevops/internal/utils/tfhelper"
)

// ResourceArea schema and implementation for area resource
func ResourceArea() *schema.Resource {
	return &schema.Resource{
		Create: resourceAreaCreate,
		Read:   resourceAreaRead,
		Update: resourceAreaUpdate,
		Delete: resourceAreaDelete,
		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(10 * time.Minute),
			Read:   schema.DefaultTimeout(5 * time.Minute),
			Update: schema.DefaultTimeout(10 * time.Minute),
			Delete: schema.DefaultTimeout(10 * time.Minute),
		},
		Importer: tfhelper.ImportProjectQualifiedResource(),
		Schema: map[string]*schema.Schema{
			"project_id": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.IsUUID,
				Description:  "The ID of the project.",
			},
			"name": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.StringIsNotWhiteSpace,
				Description:  "The name of the area.",
			},
			"path": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "The path of the area. The path should be relative to the root area. For example, to create an area under 'Team A', use '/Team A'.",
			},
			"has_children": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Indicates if the area has child areas.",
			},
		},
	}
}

func resourceAreaCreate(d *schema.ResourceData, m interface{}) error {
	clients := m.(*client.AggregatedClient)

	projectID := d.Get("project_id").(string)
	name := d.Get("name").(string)

	node := &workitemtracking.WorkItemClassificationNode{
		Name: &name,
	}

	// Get the parent path if specified
	var path *string
	if v, ok := d.GetOk("path"); ok {
		pathStr := v.(string)
		path = &pathStr
	}

	structureGroup := workitemtracking.TreeStructureGroupValues.Areas
	createdNode, err := clients.WorkItemTrackingClient.CreateOrUpdateClassificationNode(clients.Ctx, workitemtracking.CreateOrUpdateClassificationNodeArgs{
		PostedNode:     node,
		Project:        &projectID,
		StructureGroup: &structureGroup,
		Path:           path,
	})
	if err != nil {
		return fmt.Errorf("Error creating area: %+v", err)
	}

	d.SetId(createdNode.Identifier.String())

	return resourceAreaRead(d, m)
}

func resourceAreaRead(d *schema.ResourceData, m interface{}) error {
	clients := m.(*client.AggregatedClient)

	projectID := d.Get("project_id").(string)

	// Construct the path to read the area
	// The path in the state is the parent path, so we need to append the name
	var readPath string
	if v, ok := d.GetOk("path"); ok {
		parentPath := v.(string)
		name := d.Get("name").(string)
		if parentPath == "" || parentPath == "/" {
			readPath = name
		} else {
			readPath = strings.TrimPrefix(parentPath, "/") + "/" + name
		}
	} else {
		// If no path, it's a root-level area
		readPath = d.Get("name").(string)
	}

	structureGroup := workitemtracking.TreeStructureGroupValues.Areas
	node, err := clients.WorkItemTrackingClient.GetClassificationNode(clients.Ctx, workitemtracking.GetClassificationNodeArgs{
		Project:        &projectID,
		StructureGroup: &structureGroup,
		Path:           &readPath,
		Depth:          converter.Int(0),
	})
	if err != nil {
		if utils.ResponseWasNotFound(err) {
			d.SetId("")
			return nil
		}
		return fmt.Errorf("Error reading area: %+v", err)
	}

	if node.Identifier == nil {
		d.SetId("")
		return nil
	}

	flattenArea(d, node, &projectID)
	return nil
}

func resourceAreaUpdate(d *schema.ResourceData, m interface{}) error {
	clients := m.(*client.AggregatedClient)

	projectID := d.Get("project_id").(string)
	name := d.Get("name").(string)

	// Construct the current path
	var currentPath string
	if v, ok := d.GetOk("path"); ok {
		parentPath := v.(string)
		oldName := name
		if d.HasChange("name") {
			oldNameVal, _ := d.GetChange("name")
			oldName = oldNameVal.(string)
		}
		if parentPath == "" || parentPath == "/" {
			currentPath = oldName
		} else {
			currentPath = strings.TrimPrefix(parentPath, "/") + "/" + oldName
		}
	} else {
		if d.HasChange("name") {
			oldNameVal, _ := d.GetChange("name")
			currentPath = oldNameVal.(string)
		} else {
			currentPath = name
		}
	}

	node := &workitemtracking.WorkItemClassificationNode{
		Name: &name,
	}

	structureGroup := workitemtracking.TreeStructureGroupValues.Areas
	_, err := clients.WorkItemTrackingClient.UpdateClassificationNode(clients.Ctx, workitemtracking.UpdateClassificationNodeArgs{
		PostedNode:     node,
		Project:        &projectID,
		StructureGroup: &structureGroup,
		Path:           &currentPath,
	})
	if err != nil {
		return fmt.Errorf("Error updating area: %+v", err)
	}

	return resourceAreaRead(d, m)
}

func resourceAreaDelete(d *schema.ResourceData, m interface{}) error {
	clients := m.(*client.AggregatedClient)

	projectID := d.Get("project_id").(string)
	name := d.Get("name").(string)

	// Construct the path
	var deletePath string
	if v, ok := d.GetOk("path"); ok {
		parentPath := v.(string)
		if parentPath == "" || parentPath == "/" {
			deletePath = name
		} else {
			deletePath = strings.TrimPrefix(parentPath, "/") + "/" + name
		}
	} else {
		deletePath = name
	}

	structureGroup := workitemtracking.TreeStructureGroupValues.Areas
	err := clients.WorkItemTrackingClient.DeleteClassificationNode(clients.Ctx, workitemtracking.DeleteClassificationNodeArgs{
		Project:        &projectID,
		StructureGroup: &structureGroup,
		Path:           &deletePath,
	})
	if err != nil {
		return fmt.Errorf("Error deleting area: %+v", err)
	}

	d.SetId("")
	return nil
}

func flattenArea(d *schema.ResourceData, node *workitemtracking.WorkItemClassificationNode, projectID *string) {
	d.SetId(node.Identifier.String())
	d.Set("project_id", projectID)
	d.Set("name", converter.ToString(node.Name, ""))
	d.Set("has_children", converter.ToBool(node.HasChildren, false))

	// Parse the path - remove the project prefix and Area prefix
	if node.Path != nil {
		// Path format: \ProjectName\Area\ParentPath\Name
		// We want to return the parent path
		itemPath := convertAreaNodePath(node.Path, node.Name)
		d.Set("path", itemPath)
	}
}

// convertAreaNodePath converts the node path to a relative path
// Input: \ProjectName\Area\Team A\Team A.1
// Name: Team A.1
// Output: /Team A (the parent path)
func convertAreaNodePath(path *string, name *string) string {
	if path == nil {
		return "/"
	}

	itemPath := strings.ReplaceAll(*path, "\\", "/")
	parts := strings.Split(itemPath, "/")

	// Find the "Area" part and extract the parent path
	// Format: /ProjectName/Area/path/to/name
	areaIndex := -1
	for i, part := range parts {
		if part == "Area" {
			areaIndex = i
			break
		}
	}

	if areaIndex == -1 || areaIndex >= len(parts)-1 {
		return "/"
	}

	// Get everything after "Area" except the last part (which is the name)
	pathParts := parts[areaIndex+1:]
	if len(pathParts) <= 1 {
		return "/"
	}

	// Return the parent path (everything except the name at the end)
	parentPath := "/" + strings.Join(pathParts[:len(pathParts)-1], "/")
	return parentPath
}
