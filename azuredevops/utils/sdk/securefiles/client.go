// --------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.
// --------------------------------------------------------------------------------------------

package securefiles

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"

	"github.com/google/uuid"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7"
)

// ResourceAreaId for distributedtask
const ResourceAreaId = "a85b8835-c1a1-4aac-ae97-1c3d0ba72dbd"

// Client defines the interface for secure file operations
type Client interface {
	// Upload a secure file
	UploadSecureFile(ctx context.Context, args UploadSecureFileArgs) (*SecureFile, error)
	// Get a secure file by ID
	GetSecureFile(ctx context.Context, args GetSecureFileArgs) (*SecureFile, error)
	// Get secure files
	GetSecureFiles(ctx context.Context, args GetSecureFilesArgs) (*[]SecureFile, error)
	// Delete a secure file
	DeleteSecureFile(ctx context.Context, args DeleteSecureFileArgs) error
	// Update secure file properties
	UpdateSecureFile(ctx context.Context, args UpdateSecureFileArgs) (*SecureFile, error)
	// Authorize secure file for pipelines
	AuthorizeSecureFile(ctx context.Context, args AuthorizeSecureFileArgs) error
}

// ClientImpl implements the Client interface
type ClientImpl struct {
	Client azuredevops.Client
}

// NewClient creates a new securefiles client
func NewClient(ctx context.Context, connection *azuredevops.Connection) (Client, error) {
	client, err := connection.GetClientByResourceAreaId(ctx, uuid.MustParse(ResourceAreaId))
	if err != nil {
		return nil, err
	}
	return &ClientImpl{
		Client: *client,
	}, nil
}

// SecureFile represents a secure file in Azure DevOps
type SecureFile struct {
	ID         *uuid.UUID         `json:"id,omitempty"`
	Name       *string            `json:"name,omitempty"`
	Properties *map[string]string `json:"properties,omitempty"`
	CreatedOn  *azuredevops.Time  `json:"createdOn,omitempty"`
	ModifiedOn *azuredevops.Time  `json:"modifiedOn,omitempty"`
}

// UploadSecureFileArgs are arguments for uploading a secure file
type UploadSecureFileArgs struct {
	// Project ID or name
	Project *string
	// Name of the secure file
	Name *string
	// Content of the file
	Content io.Reader
}

// GetSecureFileArgs are arguments for getting a secure file
type GetSecureFileArgs struct {
	// Project ID or name
	Project *string
	// Secure file ID
	SecureFileId *uuid.UUID
}

// GetSecureFilesArgs are arguments for listing secure files
type GetSecureFilesArgs struct {
	// Project ID or name
	Project *string
	// Names to filter by
	Names *[]string
}

// DeleteSecureFileArgs are arguments for deleting a secure file
type DeleteSecureFileArgs struct {
	// Project ID or name
	Project *string
	// Secure file ID
	SecureFileId *uuid.UUID
}

// UpdateSecureFileArgs are arguments for updating a secure file
type UpdateSecureFileArgs struct {
	// Project ID or name
	Project *string
	// Secure file ID
	SecureFileId *uuid.UUID
	// Secure file to update
	SecureFile *SecureFile
}

// UploadSecureFile uploads a secure file to Azure DevOps
func (client *ClientImpl) UploadSecureFile(ctx context.Context, args UploadSecureFileArgs) (*SecureFile, error) {
	if args.Project == nil {
		return nil, &azuredevops.ArgumentNilError{ArgumentName: "args.Project"}
	}
	if args.Name == nil {
		return nil, &azuredevops.ArgumentNilError{ArgumentName: "args.Name"}
	}
	if args.Content == nil {
		return nil, &azuredevops.ArgumentNilError{ArgumentName: "args.Content"}
	}

	routeValues := make(map[string]string)
	routeValues["project"] = *args.Project

	queryParams := url.Values{}
	queryParams.Set("name", *args.Name)

	// Read content into bytes
	content, err := io.ReadAll(args.Content)
	if err != nil {
		return nil, err
	}

	locationId, _ := uuid.Parse("adcfd8bc-b184-43ba-bd84-7c8c6a2ff421")
	resp, err := client.Client.Send(ctx, http.MethodPost, locationId, "7.1-preview.1", routeValues, queryParams, bytes.NewReader(content), "application/octet-stream", "application/json", nil)
	if err != nil {
		return nil, err
	}

	var responseValue SecureFile
	err = client.Client.UnmarshalBody(resp, &responseValue)
	return &responseValue, err
}

// GetSecureFile gets a secure file by ID
func (client *ClientImpl) GetSecureFile(ctx context.Context, args GetSecureFileArgs) (*SecureFile, error) {
	if args.Project == nil {
		return nil, &azuredevops.ArgumentNilError{ArgumentName: "args.Project"}
	}
	if args.SecureFileId == nil {
		return nil, &azuredevops.ArgumentNilError{ArgumentName: "args.SecureFileId"}
	}

	routeValues := make(map[string]string)
	routeValues["project"] = *args.Project
	routeValues["secureFileId"] = args.SecureFileId.String()

	queryParams := url.Values{}

	locationId, _ := uuid.Parse("adcfd8bc-b184-43ba-bd84-7c8c6a2ff421")
	resp, err := client.Client.Send(ctx, http.MethodGet, locationId, "7.1-preview.1", routeValues, queryParams, nil, "", "application/json", nil)
	if err != nil {
		return nil, err
	}

	var responseValue SecureFile
	err = client.Client.UnmarshalBody(resp, &responseValue)
	return &responseValue, err
}

// GetSecureFiles lists secure files in a project
func (client *ClientImpl) GetSecureFiles(ctx context.Context, args GetSecureFilesArgs) (*[]SecureFile, error) {
	if args.Project == nil {
		return nil, &azuredevops.ArgumentNilError{ArgumentName: "args.Project"}
	}

	routeValues := make(map[string]string)
	routeValues["project"] = *args.Project

	queryParams := url.Values{}
	if args.Names != nil && len(*args.Names) > 0 {
		for _, name := range *args.Names {
			queryParams.Add("names", name)
		}
	}

	locationId, _ := uuid.Parse("adcfd8bc-b184-43ba-bd84-7c8c6a2ff421")
	resp, err := client.Client.Send(ctx, http.MethodGet, locationId, "7.1-preview.1", routeValues, queryParams, nil, "", "application/json", nil)
	if err != nil {
		return nil, err
	}

	var responseValue []SecureFile
	err = client.Client.UnmarshalCollectionBody(resp, &responseValue)
	return &responseValue, err
}

// DeleteSecureFile deletes a secure file
func (client *ClientImpl) DeleteSecureFile(ctx context.Context, args DeleteSecureFileArgs) error {
	if args.Project == nil {
		return &azuredevops.ArgumentNilError{ArgumentName: "args.Project"}
	}
	if args.SecureFileId == nil {
		return &azuredevops.ArgumentNilError{ArgumentName: "args.SecureFileId"}
	}

	routeValues := make(map[string]string)
	routeValues["project"] = *args.Project
	routeValues["secureFileId"] = args.SecureFileId.String()

	queryParams := url.Values{}

	locationId, _ := uuid.Parse("adcfd8bc-b184-43ba-bd84-7c8c6a2ff421")
	_, err := client.Client.Send(ctx, http.MethodDelete, locationId, "7.1-preview.1", routeValues, queryParams, nil, "", "application/json", nil)
	return err
}

// UpdateSecureFile updates a secure file's properties
func (client *ClientImpl) UpdateSecureFile(ctx context.Context, args UpdateSecureFileArgs) (*SecureFile, error) {
	if args.Project == nil {
		return nil, &azuredevops.ArgumentNilError{ArgumentName: "args.Project"}
	}
	if args.SecureFileId == nil {
		return nil, &azuredevops.ArgumentNilError{ArgumentName: "args.SecureFileId"}
	}
	if args.SecureFile == nil {
		return nil, &azuredevops.ArgumentNilError{ArgumentName: "args.SecureFile"}
	}

	routeValues := make(map[string]string)
	routeValues["project"] = *args.Project
	routeValues["secureFileId"] = args.SecureFileId.String()

	body, marshalErr := json.Marshal(args.SecureFile)
	if marshalErr != nil {
		return nil, marshalErr
	}

	queryParams := url.Values{}

	locationId, _ := uuid.Parse("adcfd8bc-b184-43ba-bd84-7c8c6a2ff421")
	resp, err := client.Client.Send(ctx, http.MethodPatch, locationId, "7.1-preview.1", routeValues, queryParams, bytes.NewReader(body), "application/json", "application/json", nil)
	if err != nil {
		return nil, err
	}

	var responseValue SecureFile
	err = client.Client.UnmarshalBody(resp, &responseValue)
	return &responseValue, err
}

// AuthorizeSecureFileArgs are arguments for authorizing a secure file for pipelines
type AuthorizeSecureFileArgs struct {
	// Project ID or name
	Project *string
	// Secure file ID
	SecureFileId *uuid.UUID
	// Whether to authorize for all pipelines
	AuthorizeForAllPipelines *bool
}

// AuthorizedResource represents an authorized resource
type AuthorizedResource struct {
	Authorized *bool   `json:"authorized,omitempty"`
	Id         *string `json:"id,omitempty"`
	Name       *string `json:"name,omitempty"`
	Type       *string `json:"type,omitempty"`
}

// AuthorizeSecureFile authorizes a secure file for pipelines
func (client *ClientImpl) AuthorizeSecureFile(ctx context.Context, args AuthorizeSecureFileArgs) error {
	if args.Project == nil {
		return &azuredevops.ArgumentNilError{ArgumentName: "args.Project"}
	}
	if args.SecureFileId == nil {
		return &azuredevops.ArgumentNilError{ArgumentName: "args.SecureFileId"}
	}

	routeValues := make(map[string]string)
	routeValues["project"] = *args.Project

	authorized := false
	if args.AuthorizeForAllPipelines != nil {
		authorized = *args.AuthorizeForAllPipelines
	}

	resources := []AuthorizedResource{
		{
			Authorized: &authorized,
			Id:         strToPtr(args.SecureFileId.String()),
			Type:       strToPtr("securefile"),
		},
	}

	body, marshalErr := json.Marshal(resources)
	if marshalErr != nil {
		return marshalErr
	}

	queryParams := url.Values{}

	locationId, _ := uuid.Parse("398c85bc-81aa-4822-947c-a194a05f0fef")
	_, err := client.Client.Send(ctx, http.MethodPatch, locationId, "7.1-preview.1", routeValues, queryParams, bytes.NewReader(body), "application/json", "application/json", nil)
	return err
}

func strToPtr(s string) *string {
	return &s
}
