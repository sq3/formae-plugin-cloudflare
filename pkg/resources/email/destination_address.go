// SPDX-License-Identifier: Apache-2.0

package email

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/platform-engineering-labs/formae-plugin-cloudflare/pkg/resources/prov"
	"github.com/platform-engineering-labs/formae-plugin-cloudflare/pkg/resources/registry"
	"github.com/platform-engineering-labs/formae-plugin-cloudflare/pkg/transport"
	"github.com/platform-engineering-labs/formae/pkg/plugin/resource"
)

const DestinationAddressResourceType = "Cloudflare::Email::DestinationAddress"

// destinationAddressProvisioner handles email destination address lifecycle operations
type destinationAddressProvisioner struct {
	client *transport.Client
}

var _ prov.Provisioner = &destinationAddressProvisioner{}

// destinationAddressProps represents destination address properties
type destinationAddressProps struct {
	AccountId string `json:"account_id"`
	Email     string `json:"email"`
}

// parseNativeID extracts account_id and address_id from native ID (format: account_id/address_id)
func parseAddressNativeID(nativeID string) (accountId, addressId string, err error) {
	parts := strings.SplitN(nativeID, "/", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid native ID format, expected 'account_id/address_id': %s", nativeID)
	}
	return parts[0], parts[1], nil
}

// Create creates a new destination address
func (p *destinationAddressProvisioner) Create(ctx context.Context, request *resource.CreateRequest) (*resource.CreateResult, error) {
	var props destinationAddressProps
	if err := json.Unmarshal(request.Properties, &props); err != nil {
		return createAddressFailure(resource.OperationErrorCodeInvalidRequest,
			fmt.Sprintf("failed to parse properties: %v", err)), nil
	}

	if props.AccountId == "" {
		return createAddressFailure(resource.OperationErrorCodeInvalidRequest, "account_id is required"), nil
	}
	if props.Email == "" {
		return createAddressFailure(resource.OperationErrorCodeInvalidRequest, "email is required"), nil
	}

	addr, err := p.client.CreateDestinationAddress(ctx, props.AccountId, props.Email)
	if err != nil {
		errCode := transport.ClassifyCloudflareError(err)
		return createAddressFailure(transport.ToResourceErrorCode(errCode), err.Error()), nil
	}

	nativeID := fmt.Sprintf("%s/%s", props.AccountId, addr.ID)
	propsJSON, _ := json.Marshal(map[string]interface{}{
		"id":         addr.ID,
		"account_id": props.AccountId,
		"email":      addr.Email,
		"verified":   addr.Verified,
		"created":    addr.Created.String(),
		"modified":   addr.Modified.String(),
	})

	return &resource.CreateResult{
		ProgressResult: &resource.ProgressResult{
			Operation:          resource.OperationCreate,
			OperationStatus:    resource.OperationStatusSuccess,
			NativeID:           nativeID,
			ResourceProperties: propsJSON,
		},
	}, nil
}

// Read retrieves a destination address by ID
func (p *destinationAddressProvisioner) Read(ctx context.Context, request *resource.ReadRequest) (*resource.ReadResult, error) {
	accountId, addressId, err := parseAddressNativeID(request.NativeID)
	if err != nil {
		return &resource.ReadResult{
			ErrorCode: resource.OperationErrorCodeInvalidRequest,
		}, nil
	}

	addr, err := p.client.GetDestinationAddress(ctx, accountId, addressId)
	if err != nil {
		if transport.IsNotFound(err) {
			return &resource.ReadResult{
				ErrorCode: resource.OperationErrorCodeNotFound,
			}, nil
		}
		errCode := transport.ClassifyCloudflareError(err)
		return &resource.ReadResult{
			ErrorCode: transport.ToResourceErrorCode(errCode),
		}, nil
	}

	propsJSON, _ := json.Marshal(map[string]interface{}{
		"id":         addr.ID,
		"account_id": accountId,
		"email":      addr.Email,
		"verified":   addr.Verified,
		"created":    addr.Created.String(),
		"modified":   addr.Modified.String(),
	})

	return &resource.ReadResult{
		Properties: string(propsJSON),
	}, nil
}

// Update - destination addresses cannot be updated, only created/deleted
func (p *destinationAddressProvisioner) Update(ctx context.Context, request *resource.UpdateRequest) (*resource.UpdateResult, error) {
	return &resource.UpdateResult{
		ProgressResult: &resource.ProgressResult{
			Operation:       resource.OperationUpdate,
			OperationStatus: resource.OperationStatusFailure,
			ErrorCode:       resource.OperationErrorCodeInvalidRequest,
			StatusMessage:   "destination addresses cannot be updated, only created or deleted",
			NativeID:        request.NativeID,
		},
	}, nil
}

// Delete removes a destination address
func (p *destinationAddressProvisioner) Delete(ctx context.Context, request *resource.DeleteRequest) (*resource.DeleteResult, error) {
	accountId, addressId, err := parseAddressNativeID(request.NativeID)
	if err != nil {
		return &resource.DeleteResult{
			ProgressResult: &resource.ProgressResult{
				Operation:       resource.OperationDelete,
				OperationStatus: resource.OperationStatusFailure,
				ErrorCode:       resource.OperationErrorCodeInvalidRequest,
				StatusMessage:   err.Error(),
			},
		}, nil
	}

	err = p.client.DeleteDestinationAddress(ctx, accountId, addressId)
	if err != nil {
		if transport.IsNotFound(err) {
			return &resource.DeleteResult{
				ProgressResult: &resource.ProgressResult{
					Operation:       resource.OperationDelete,
					OperationStatus: resource.OperationStatusSuccess,
					NativeID:        request.NativeID,
				},
			}, nil
		}
		errCode := transport.ClassifyCloudflareError(err)
		return &resource.DeleteResult{
			ProgressResult: &resource.ProgressResult{
				Operation:       resource.OperationDelete,
				OperationStatus: resource.OperationStatusFailure,
				ErrorCode:       transport.ToResourceErrorCode(errCode),
				StatusMessage:   err.Error(),
				NativeID:        request.NativeID,
			},
		}, nil
	}

	return &resource.DeleteResult{
		ProgressResult: &resource.ProgressResult{
			Operation:       resource.OperationDelete,
			OperationStatus: resource.OperationStatusSuccess,
			NativeID:        request.NativeID,
		},
	}, nil
}

// Status checks the current state of a destination address
func (p *destinationAddressProvisioner) Status(ctx context.Context, request *resource.StatusRequest) (*resource.StatusResult, error) {
	accountId, addressId, err := parseAddressNativeID(request.NativeID)
	if err != nil {
		return &resource.StatusResult{
			ProgressResult: &resource.ProgressResult{
				Operation:       resource.OperationCheckStatus,
				OperationStatus: resource.OperationStatusFailure,
				ErrorCode:       resource.OperationErrorCodeInvalidRequest,
				StatusMessage:   err.Error(),
			},
		}, nil
	}

	addr, err := p.client.GetDestinationAddress(ctx, accountId, addressId)
	if err != nil {
		if transport.IsNotFound(err) {
			return &resource.StatusResult{
				ProgressResult: &resource.ProgressResult{
					Operation:       resource.OperationCheckStatus,
					OperationStatus: resource.OperationStatusFailure,
					ErrorCode:       resource.OperationErrorCodeNotFound,
					NativeID:        request.NativeID,
				},
			}, nil
		}
		errCode := transport.ClassifyCloudflareError(err)
		return &resource.StatusResult{
			ProgressResult: &resource.ProgressResult{
				Operation:       resource.OperationCheckStatus,
				OperationStatus: resource.OperationStatusFailure,
				ErrorCode:       transport.ToResourceErrorCode(errCode),
				StatusMessage:   err.Error(),
				NativeID:        request.NativeID,
			},
		}, nil
	}

	propsJSON, _ := json.Marshal(map[string]interface{}{
		"id":         addr.ID,
		"account_id": accountId,
		"email":      addr.Email,
		"verified":   addr.Verified,
		"created":    addr.Created.String(),
		"modified":   addr.Modified.String(),
	})

	return &resource.StatusResult{
		ProgressResult: &resource.ProgressResult{
			Operation:          resource.OperationCheckStatus,
			OperationStatus:    resource.OperationStatusSuccess,
			NativeID:           request.NativeID,
			ResourceProperties: propsJSON,
		},
	}, nil
}

// List returns all destination addresses for an account
func (p *destinationAddressProvisioner) List(ctx context.Context, request *resource.ListRequest) (*resource.ListResult, error) {
	accountId := p.client.AccountId()
	if accountId == "" {
		var cfg struct {
			AccountId string `json:"AccountId"`
		}
		if len(request.TargetConfig) > 0 {
			_ = json.Unmarshal(request.TargetConfig, &cfg)
			accountId = cfg.AccountId
		}
	}

	if accountId == "" {
		return &resource.ListResult{
			NativeIDs: []string{},
		}, nil
	}

	addresses, err := p.client.ListDestinationAddresses(ctx, accountId)
	if err != nil {
		return nil, fmt.Errorf("failed to list destination addresses: %w", err)
	}

	var nativeIDs []string
	for _, addr := range addresses {
		nativeID := fmt.Sprintf("%s/%s", accountId, addr.ID)
		nativeIDs = append(nativeIDs, nativeID)
	}

	return &resource.ListResult{
		NativeIDs: nativeIDs,
	}, nil
}

func createAddressFailure(code resource.OperationErrorCode, msg string) *resource.CreateResult {
	return &resource.CreateResult{
		ProgressResult: &resource.ProgressResult{
			Operation:       resource.OperationCreate,
			OperationStatus: resource.OperationStatusFailure,
			ErrorCode:       code,
			StatusMessage:   msg,
		},
	}
}

func init() {
	registry.Register(
		DestinationAddressResourceType,
		[]resource.Operation{
			resource.OperationCreate,
			resource.OperationRead,
			resource.OperationDelete,
			resource.OperationList,
			resource.OperationCheckStatus,
		},
		func(client *transport.Client) prov.Provisioner {
			return &destinationAddressProvisioner{client: client}
		},
	)
}
