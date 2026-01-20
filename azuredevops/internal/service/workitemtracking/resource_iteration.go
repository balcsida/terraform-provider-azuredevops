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

// ResourceIteration schema and implementation for iteration resource
func ResourceIteration() *schema.Resource {
	return &schema.Resource{
		Create: resourceIterationCreate,
		Read:   resourceIterationRead,
		Update: resourceIterationUpdate,
		Delete: resourceIterationDelete,
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
				Description:  "The name of the iteration.",
			},
			"path": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "The path of the iteration. The path should be relative to the root iteration. For example, to create an iteration under 'Sprint 1', use '/Sprint 1'.",
			},
			"start_date": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.IsRFC3339Time,
				Description:  "The start date of the iteration in RFC3339 format.",
			},
			"finish_date": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.IsRFC3339Time,
				Description:  "The finish date of the iteration in RFC3339 format.",
			},
			"has_children": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Indicates if the iteration has child iterations.",
			},
		},
	}
}

func resourceIterationCreate(d *schema.ResourceData, m interface{}) error {
	clients := m.(*client.AggregatedClient)

	projectID := d.Get("project_id").(string)
	name := d.Get("name").(string)

	node := &workitemtracking.WorkItemClassificationNode{
		Name: &name,
	}

	// Set attributes for start and finish dates
	attributes := make(map[string]interface{})
	if v, ok := d.GetOk("start_date"); ok {
		startDate, err := time.Parse(time.RFC3339, v.(string))
		if err != nil {
			return fmt.Errorf("Error parsing start_date: %+v", err)
		}
		attributes["startDate"] = startDate.Format(time.RFC3339)
	}
	if v, ok := d.GetOk("finish_date"); ok {
		finishDate, err := time.Parse(time.RFC3339, v.(string))
		if err != nil {
			return fmt.Errorf("Error parsing finish_date: %+v", err)
		}
		attributes["finishDate"] = finishDate.Format(time.RFC3339)
	}
	if len(attributes) > 0 {
		node.Attributes = &attributes
	}

	// Get the parent path if specified
	var path *string
	if v, ok := d.GetOk("path"); ok {
		pathStr := v.(string)
		path = &pathStr
	}

	structureGroup := workitemtracking.TreeStructureGroupValues.Iterations
	createdNode, err := clients.WorkItemTrackingClient.CreateOrUpdateClassificationNode(clients.Ctx, workitemtracking.CreateOrUpdateClassificationNodeArgs{
		PostedNode:     node,
		Project:        &projectID,
		StructureGroup: &structureGroup,
		Path:           path,
	})
	if err != nil {
		return fmt.Errorf("Error creating iteration: %+v", err)
	}

	d.SetId(createdNode.Identifier.String())

	return resourceIterationRead(d, m)
}

func resourceIterationRead(d *schema.ResourceData, m interface{}) error {
	clients := m.(*client.AggregatedClient)

	projectID := d.Get("project_id").(string)

	// Construct the path to read the iteration
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
		// If no path, it's a root-level iteration
		readPath = d.Get("name").(string)
	}

	structureGroup := workitemtracking.TreeStructureGroupValues.Iterations
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
		return fmt.Errorf("Error reading iteration: %+v", err)
	}

	if node.Identifier == nil {
		d.SetId("")
		return nil
	}

	flattenIteration(d, node, &projectID)
	return nil
}

func resourceIterationUpdate(d *schema.ResourceData, m interface{}) error {
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

	// Set attributes for start and finish dates
	attributes := make(map[string]interface{})
	if v, ok := d.GetOk("start_date"); ok {
		startDate, err := time.Parse(time.RFC3339, v.(string))
		if err != nil {
			return fmt.Errorf("Error parsing start_date: %+v", err)
		}
		attributes["startDate"] = startDate.Format(time.RFC3339)
	}
	if v, ok := d.GetOk("finish_date"); ok {
		finishDate, err := time.Parse(time.RFC3339, v.(string))
		if err != nil {
			return fmt.Errorf("Error parsing finish_date: %+v", err)
		}
		attributes["finishDate"] = finishDate.Format(time.RFC3339)
	}
	if len(attributes) > 0 {
		node.Attributes = &attributes
	}

	structureGroup := workitemtracking.TreeStructureGroupValues.Iterations
	_, err := clients.WorkItemTrackingClient.UpdateClassificationNode(clients.Ctx, workitemtracking.UpdateClassificationNodeArgs{
		PostedNode:     node,
		Project:        &projectID,
		StructureGroup: &structureGroup,
		Path:           &currentPath,
	})
	if err != nil {
		return fmt.Errorf("Error updating iteration: %+v", err)
	}

	return resourceIterationRead(d, m)
}

func resourceIterationDelete(d *schema.ResourceData, m interface{}) error {
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

	structureGroup := workitemtracking.TreeStructureGroupValues.Iterations
	err := clients.WorkItemTrackingClient.DeleteClassificationNode(clients.Ctx, workitemtracking.DeleteClassificationNodeArgs{
		Project:        &projectID,
		StructureGroup: &structureGroup,
		Path:           &deletePath,
	})
	if err != nil {
		return fmt.Errorf("Error deleting iteration: %+v", err)
	}

	d.SetId("")
	return nil
}

func flattenIteration(d *schema.ResourceData, node *workitemtracking.WorkItemClassificationNode, projectID *string) {
	d.SetId(node.Identifier.String())
	d.Set("project_id", projectID)
	d.Set("name", converter.ToString(node.Name, ""))
	d.Set("has_children", converter.ToBool(node.HasChildren, false))

	// Parse the path - remove the project prefix and Iterations prefix
	if node.Path != nil {
		// Path format: \ProjectName\Iteration\ParentPath\Name
		// We want to return the parent path
		itemPath := convertIterationNodePath(node.Path, node.Name)
		d.Set("path", itemPath)
	}

	// Parse attributes for start and finish dates
	if node.Attributes != nil {
		attrs := *node.Attributes
		if startDate, ok := attrs["startDate"]; ok {
			if startDateStr, ok := startDate.(string); ok {
				d.Set("start_date", startDateStr)
			}
		}
		if finishDate, ok := attrs["finishDate"]; ok {
			if finishDateStr, ok := finishDate.(string); ok {
				d.Set("finish_date", finishDateStr)
			}
		}
	}
}

// convertIterationNodePath converts the node path to a relative path
// Input: \ProjectName\Iteration\Sprint 1\Sprint 1.1
// Name: Sprint 1.1
// Output: /Sprint 1 (the parent path)
func convertIterationNodePath(path *string, name *string) string {
	if path == nil {
		return "/"
	}

	itemPath := strings.ReplaceAll(*path, "\\", "/")
	parts := strings.Split(itemPath, "/")

	// Find the "Iteration" part and extract the parent path
	// Format: /ProjectName/Iteration/path/to/name
	iterationIndex := -1
	for i, part := range parts {
		if part == "Iteration" {
			iterationIndex = i
			break
		}
	}

	if iterationIndex == -1 || iterationIndex >= len(parts)-1 {
		return "/"
	}

	// Get everything after "Iteration" except the last part (which is the name)
	pathParts := parts[iterationIndex+1:]
	if len(pathParts) <= 1 {
		return "/"
	}

	// Return the parent path (everything except the name at the end)
	parentPath := "/" + strings.Join(pathParts[:len(pathParts)-1], "/")
	return parentPath
}
