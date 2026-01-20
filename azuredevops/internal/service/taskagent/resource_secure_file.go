package taskagent

import (
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/microsoft/terraform-provider-azuredevops/azuredevops/internal/client"
	"github.com/microsoft/terraform-provider-azuredevops/azuredevops/internal/utils"
	"github.com/microsoft/terraform-provider-azuredevops/azuredevops/internal/utils/converter"
	"github.com/microsoft/terraform-provider-azuredevops/azuredevops/internal/utils/tfhelper"
	"github.com/microsoft/terraform-provider-azuredevops/azuredevops/utils/sdk/securefiles"
)

// ResourceSecureFile schema and implementation for secure file resource
func ResourceSecureFile() *schema.Resource {
	return &schema.Resource{
		Create: resourceSecureFileCreate,
		Read:   resourceSecureFileRead,
		Update: resourceSecureFileUpdate,
		Delete: resourceSecureFileDelete,
		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(10 * time.Minute),
			Read:   schema.DefaultTimeout(5 * time.Minute),
			Update: schema.DefaultTimeout(10 * time.Minute),
			Delete: schema.DefaultTimeout(10 * time.Minute),
		},
		Importer: tfhelper.ImportProjectQualifiedResourceUUID(),
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
				ForceNew:     true,
				ValidateFunc: validation.StringIsNotWhiteSpace,
				Description:  "The name of the secure file.",
			},
			"content": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				Sensitive:    true,
				ValidateFunc: validation.StringIsNotWhiteSpace,
				Description:  "The content of the secure file as a base64 encoded string.",
			},
			"authorize": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Whether the secure file is authorized for use by all pipelines in the project.",
			},
			"properties": {
				Type:        schema.TypeMap,
				Optional:    true,
				Description: "A map of properties to associate with the secure file.",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
		},
	}
}

func resourceSecureFileCreate(d *schema.ResourceData, m interface{}) error {
	clients := m.(*client.AggregatedClient)

	projectID := d.Get("project_id").(string)
	name := d.Get("name").(string)
	contentBase64 := d.Get("content").(string)

	// Decode base64 content
	content, err := base64.StdEncoding.DecodeString(contentBase64)
	if err != nil {
		return fmt.Errorf("Failed to decode base64 content: %+v", err)
	}

	// Upload the secure file
	secureFile, err := clients.SecureFilesClient.UploadSecureFile(clients.Ctx, securefiles.UploadSecureFileArgs{
		Project: &projectID,
		Name:    &name,
		Content: strings.NewReader(string(content)),
	})
	if err != nil {
		return fmt.Errorf("Failed to upload secure file: %+v", err)
	}

	d.SetId(secureFile.ID.String())

	// Update properties if specified
	if v, ok := d.GetOk("properties"); ok {
		propsMap := v.(map[string]interface{})
		if len(propsMap) > 0 {
			props := make(map[string]string)
			for k, val := range propsMap {
				props[k] = val.(string)
			}
			_, err := clients.SecureFilesClient.UpdateSecureFile(clients.Ctx, securefiles.UpdateSecureFileArgs{
				Project:      &projectID,
				SecureFileId: secureFile.ID,
				SecureFile: &securefiles.SecureFile{
					ID:         secureFile.ID,
					Name:       secureFile.Name,
					Properties: &props,
				},
			})
			if err != nil {
				return fmt.Errorf("Failed to update secure file properties: %+v", err)
			}
		}
	}

	// Handle authorization
	if d.Get("authorize").(bool) {
		err := clients.SecureFilesClient.AuthorizeSecureFile(clients.Ctx, securefiles.AuthorizeSecureFileArgs{
			Project:                  &projectID,
			SecureFileId:             secureFile.ID,
			AuthorizeForAllPipelines: converter.Bool(true),
		})
		if err != nil {
			return fmt.Errorf("Failed to authorize secure file: %+v", err)
		}
	}

	return resourceSecureFileRead(d, m)
}

func resourceSecureFileRead(d *schema.ResourceData, m interface{}) error {
	clients := m.(*client.AggregatedClient)

	projectID := d.Get("project_id").(string)
	secureFileId, err := uuid.Parse(d.Id())
	if err != nil {
		return fmt.Errorf("Error parsing secure file ID: %+v", err)
	}

	secureFile, err := clients.SecureFilesClient.GetSecureFile(clients.Ctx, securefiles.GetSecureFileArgs{
		Project:      &projectID,
		SecureFileId: &secureFileId,
	})
	if err != nil {
		if utils.ResponseWasNotFound(err) {
			d.SetId("")
			return nil
		}
		return fmt.Errorf("Error reading secure file: %+v", err)
	}

	if secureFile.ID == nil {
		d.SetId("")
		return nil
	}

	d.Set("name", secureFile.Name)
	d.Set("project_id", projectID)

	if secureFile.Properties != nil {
		d.Set("properties", *secureFile.Properties)
	}

	return nil
}

func resourceSecureFileUpdate(d *schema.ResourceData, m interface{}) error {
	clients := m.(*client.AggregatedClient)

	projectID := d.Get("project_id").(string)
	secureFileId, err := uuid.Parse(d.Id())
	if err != nil {
		return fmt.Errorf("Error parsing secure file ID: %+v", err)
	}

	// Update properties if changed
	if d.HasChange("properties") {
		propsMap := d.Get("properties").(map[string]interface{})
		props := make(map[string]string)
		for k, val := range propsMap {
			props[k] = val.(string)
		}

		name := d.Get("name").(string)
		_, err := clients.SecureFilesClient.UpdateSecureFile(clients.Ctx, securefiles.UpdateSecureFileArgs{
			Project:      &projectID,
			SecureFileId: &secureFileId,
			SecureFile: &securefiles.SecureFile{
				ID:         &secureFileId,
				Name:       &name,
				Properties: &props,
			},
		})
		if err != nil {
			return fmt.Errorf("Failed to update secure file properties: %+v", err)
		}
	}

	// Handle authorization changes
	if d.HasChange("authorize") {
		err := clients.SecureFilesClient.AuthorizeSecureFile(clients.Ctx, securefiles.AuthorizeSecureFileArgs{
			Project:                  &projectID,
			SecureFileId:             &secureFileId,
			AuthorizeForAllPipelines: converter.Bool(d.Get("authorize").(bool)),
		})
		if err != nil {
			return fmt.Errorf("Failed to update secure file authorization: %+v", err)
		}
	}

	return resourceSecureFileRead(d, m)
}

func resourceSecureFileDelete(d *schema.ResourceData, m interface{}) error {
	clients := m.(*client.AggregatedClient)

	projectID := d.Get("project_id").(string)
	secureFileId, err := uuid.Parse(d.Id())
	if err != nil {
		return fmt.Errorf("Error parsing secure file ID: %+v", err)
	}

	err = clients.SecureFilesClient.DeleteSecureFile(clients.Ctx, securefiles.DeleteSecureFileArgs{
		Project:      &projectID,
		SecureFileId: &secureFileId,
	})
	if err != nil {
		return fmt.Errorf("Error deleting secure file: %+v", err)
	}

	d.SetId("")
	return nil
}
