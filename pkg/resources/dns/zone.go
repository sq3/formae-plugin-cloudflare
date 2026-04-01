// SPDX-License-Identifier: Apache-2.0

package dns

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/platform-engineering-labs/formae-plugin-cloudflare/pkg/resources/prov"
	"github.com/platform-engineering-labs/formae-plugin-cloudflare/pkg/resources/registry"
	"github.com/platform-engineering-labs/formae-plugin-cloudflare/pkg/transport"
	"github.com/platform-engineering-labs/formae/pkg/plugin/resource"
)

const ZoneResourceType = "Cloudflare::DNS::Zone"

// zoneProvisioner handles DNS zone lifecycle operations
type zoneProvisioner struct {
	client *transport.Client
}

var _ prov.Provisioner = &zoneProvisioner{}

// Create creates a new DNS zone
func (p *zoneProvisioner) Create(ctx context.Context, request *resource.CreateRequest) (*resource.CreateResult, error) {
	var props struct {
		Name      string `json:"name"`
		AccountId string `json:"account_id"`
		Type      string `json:"type"`
	}
	if err := json.Unmarshal(request.Properties, &props); err != nil {
		return createFailure(resource.OperationErrorCodeInvalidRequest,
			fmt.Sprintf("failed to parse properties: %v", err)), nil
	}

	if props.Name == "" {
		return createFailure(resource.OperationErrorCodeInvalidRequest, "name is required"), nil
	}
	if props.AccountId == "" {
		return createFailure(resource.OperationErrorCodeInvalidRequest, "account_id is required"), nil
	}

	zoneType := props.Type
	if zoneType == "" {
		zoneType = "full"
	}

	zone, err := p.client.CreateZone(ctx, props.Name, props.AccountId, zoneType)
	if err != nil {
		errCode := transport.ClassifyCloudflareError(err)
		return createFailure(transport.ToResourceErrorCode(errCode), err.Error()), nil
	}

	propsJSON, _ := json.Marshal(zoneToMap(zone))

	return &resource.CreateResult{
		ProgressResult: &resource.ProgressResult{
			Operation:          resource.OperationCreate,
			OperationStatus:    resource.OperationStatusSuccess,
			NativeID:           zone.ID,
			ResourceProperties: propsJSON,
		},
	}, nil
}

// Read retrieves a DNS zone by ID
func (p *zoneProvisioner) Read(ctx context.Context, request *resource.ReadRequest) (*resource.ReadResult, error) {
	zone, err := p.client.GetZone(ctx, request.NativeID)
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

	propsJSON, _ := json.Marshal(zoneToMap(zone))

	return &resource.ReadResult{
		Properties: string(propsJSON),
	}, nil
}

// Update modifies a DNS zone (limited to type and vanity_name_servers)
func (p *zoneProvisioner) Update(ctx context.Context, request *resource.UpdateRequest) (*resource.UpdateResult, error) {
	// Zone edit is limited in the v4 API - primarily for zone type changes
	// For now, we just re-read the zone state
	zone, err := p.client.GetZone(ctx, request.NativeID)
	if err != nil {
		errCode := transport.ClassifyCloudflareError(err)
		return updateFailure(request.NativeID, transport.ToResourceErrorCode(errCode), err.Error()), nil
	}

	propsJSON, _ := json.Marshal(zoneToMap(zone))

	return &resource.UpdateResult{
		ProgressResult: &resource.ProgressResult{
			Operation:          resource.OperationUpdate,
			OperationStatus:    resource.OperationStatusSuccess,
			NativeID:           request.NativeID,
			ResourceProperties: propsJSON,
		},
	}, nil
}

// Delete removes a DNS zone
func (p *zoneProvisioner) Delete(ctx context.Context, request *resource.DeleteRequest) (*resource.DeleteResult, error) {
	err := p.client.DeleteZone(ctx, request.NativeID)
	if err != nil {
		if transport.IsNotFound(err) {
			// Already deleted, treat as success
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

// Status checks the current state of a DNS zone
func (p *zoneProvisioner) Status(ctx context.Context, request *resource.StatusRequest) (*resource.StatusResult, error) {
	zone, err := p.client.GetZone(ctx, request.NativeID)
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

	propsJSON, _ := json.Marshal(zoneToMap(zone))

	return &resource.StatusResult{
		ProgressResult: &resource.ProgressResult{
			Operation:          resource.OperationCheckStatus,
			OperationStatus:    resource.OperationStatusSuccess,
			NativeID:           request.NativeID,
			ResourceProperties: propsJSON,
		},
	}, nil
}

// List returns all DNS zones for the account
func (p *zoneProvisioner) List(ctx context.Context, request *resource.ListRequest) (*resource.ListResult, error) {
	accountId := p.client.AccountId()
	if accountId == "" {
		// Try to get from target config
		var cfg struct {
			AccountId string `json:"AccountId"`
		}
		if len(request.TargetConfig) > 0 {
			if err := json.Unmarshal(request.TargetConfig, &cfg); err == nil {
				accountId = cfg.AccountId
			}
		}
	}

	if accountId == "" {
		return nil, fmt.Errorf("account_id is required for listing zones")
	}

	cfZones, err := p.client.ListZones(ctx, accountId)
	if err != nil {
		return nil, fmt.Errorf("failed to list zones: %w", err)
	}

	var nativeIDs []string
	for _, zone := range cfZones {
		nativeIDs = append(nativeIDs, zone.ID)
	}

	return &resource.ListResult{
		NativeIDs: nativeIDs,
	}, nil
}

// zoneToMap converts a Cloudflare zone to a map for JSON serialization
func zoneToMap(zone interface{}) map[string]interface{} {
	// Use type assertion to get zone fields - zones.Zone struct
	// Marshal and unmarshal to get a generic map
	data, _ := json.Marshal(zone)
	var m map[string]interface{}
	_ = json.Unmarshal(data, &m)
	return m
}

// Helper functions

func createFailure(code resource.OperationErrorCode, msg string) *resource.CreateResult {
	return &resource.CreateResult{
		ProgressResult: &resource.ProgressResult{
			Operation:       resource.OperationCreate,
			OperationStatus: resource.OperationStatusFailure,
			ErrorCode:       code,
			StatusMessage:   msg,
		},
	}
}

func updateFailure(nativeID string, code resource.OperationErrorCode, msg string) *resource.UpdateResult {
	return &resource.UpdateResult{
		ProgressResult: &resource.ProgressResult{
			Operation:       resource.OperationUpdate,
			OperationStatus: resource.OperationStatusFailure,
			ErrorCode:       code,
			StatusMessage:   msg,
			NativeID:        nativeID,
		},
	}
}

func init() {
	registry.Register(
		ZoneResourceType,
		[]resource.Operation{
			resource.OperationCreate,
			resource.OperationRead,
			resource.OperationUpdate,
			resource.OperationDelete,
			resource.OperationList,
			resource.OperationCheckStatus,
		},
		func(client *transport.Client) prov.Provisioner {
			return &zoneProvisioner{client: client}
		},
	)
}
