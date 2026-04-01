// SPDX-License-Identifier: Apache-2.0

package email

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	cf "github.com/cloudflare/cloudflare-go/v4"
	"github.com/cloudflare/cloudflare-go/v4/email_routing"
	"github.com/platform-engineering-labs/formae-plugin-cloudflare/pkg/resources/prov"
	"github.com/platform-engineering-labs/formae-plugin-cloudflare/pkg/resources/registry"
	"github.com/platform-engineering-labs/formae-plugin-cloudflare/pkg/transport"
	"github.com/platform-engineering-labs/formae/pkg/plugin/resource"
)

const RoutingRuleResourceType = "Cloudflare::Email::RoutingRule"

// routingRuleProvisioner handles email routing rule lifecycle operations
type routingRuleProvisioner struct {
	client *transport.Client
}

var _ prov.Provisioner = &routingRuleProvisioner{}

// routingRuleProps represents email routing rule properties
type routingRuleProps struct {
	ZoneId   string `json:"zone_id"`
	Name     string `json:"name,omitempty"`
	Enabled  bool   `json:"enabled"`
	Priority int    `json:"priority,omitempty"`
}

// parseNativeID extracts zone_id and rule_id from native ID (format: zone_id/rule_id)
func parseRuleNativeID(nativeID string) (zoneId, ruleId string, err error) {
	parts := strings.SplitN(nativeID, "/", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid native ID format, expected 'zone_id/rule_id': %s", nativeID)
	}
	return parts[0], parts[1], nil
}

// Create creates a new email routing rule
func (p *routingRuleProvisioner) Create(ctx context.Context, request *resource.CreateRequest) (*resource.CreateResult, error) {
	var props routingRuleProps
	if err := json.Unmarshal(request.Properties, &props); err != nil {
		return createRuleFailure(resource.OperationErrorCodeInvalidRequest,
			fmt.Sprintf("failed to parse properties: %v", err)), nil
	}

	if props.ZoneId == "" {
		return createRuleFailure(resource.OperationErrorCodeInvalidRequest, "zone_id is required"), nil
	}

	// Build params for a simple catch-all forward rule
	params := email_routing.RuleNewParams{
		ZoneID: cf.F(props.ZoneId),
	}

	if props.Name != "" {
		params.Name = cf.F(props.Name)
	}
	if props.Priority > 0 {
		params.Priority = cf.F(float64(props.Priority))
	}

	rule, err := p.client.CreateEmailRoutingRule(ctx, props.ZoneId, params)
	if err != nil {
		errCode := transport.ClassifyCloudflareError(err)
		return createRuleFailure(transport.ToResourceErrorCode(errCode), err.Error()), nil
	}

	nativeID := fmt.Sprintf("%s/%s", props.ZoneId, rule.ID)
	propsJSON, _ := json.Marshal(ruleToMap(rule, props.ZoneId))

	return &resource.CreateResult{
		ProgressResult: &resource.ProgressResult{
			Operation:          resource.OperationCreate,
			OperationStatus:    resource.OperationStatusSuccess,
			NativeID:           nativeID,
			ResourceProperties: propsJSON,
		},
	}, nil
}

// Read retrieves an email routing rule by ID
func (p *routingRuleProvisioner) Read(ctx context.Context, request *resource.ReadRequest) (*resource.ReadResult, error) {
	zoneId, ruleId, err := parseRuleNativeID(request.NativeID)
	if err != nil {
		return &resource.ReadResult{
			ErrorCode: resource.OperationErrorCodeInvalidRequest,
		}, nil
	}

	rule, err := p.client.GetEmailRoutingRule(ctx, zoneId, ruleId)
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

	propsJSON, _ := json.Marshal(ruleToMap(rule, zoneId))

	return &resource.ReadResult{
		Properties: string(propsJSON),
	}, nil
}

// Update modifies an email routing rule
func (p *routingRuleProvisioner) Update(ctx context.Context, request *resource.UpdateRequest) (*resource.UpdateResult, error) {
	zoneId, ruleId, err := parseRuleNativeID(request.NativeID)
	if err != nil {
		return updateRuleFailure(request.NativeID, resource.OperationErrorCodeInvalidRequest, err.Error()), nil
	}

	var props routingRuleProps
	if err := json.Unmarshal(request.DesiredProperties, &props); err != nil {
		return updateRuleFailure(request.NativeID, resource.OperationErrorCodeInvalidRequest,
			fmt.Sprintf("failed to parse properties: %v", err)), nil
	}

	params := email_routing.RuleUpdateParams{
		ZoneID: cf.F(zoneId),
	}

	if props.Name != "" {
		params.Name = cf.F(props.Name)
	}
	if props.Priority > 0 {
		params.Priority = cf.F(float64(props.Priority))
	}

	rule, err := p.client.UpdateEmailRoutingRule(ctx, zoneId, ruleId, params)
	if err != nil {
		errCode := transport.ClassifyCloudflareError(err)
		return updateRuleFailure(request.NativeID, transport.ToResourceErrorCode(errCode), err.Error()), nil
	}

	propsJSON, _ := json.Marshal(ruleToMap(rule, zoneId))

	return &resource.UpdateResult{
		ProgressResult: &resource.ProgressResult{
			Operation:          resource.OperationUpdate,
			OperationStatus:    resource.OperationStatusSuccess,
			NativeID:           request.NativeID,
			ResourceProperties: propsJSON,
		},
	}, nil
}

// Delete removes an email routing rule
func (p *routingRuleProvisioner) Delete(ctx context.Context, request *resource.DeleteRequest) (*resource.DeleteResult, error) {
	zoneId, ruleId, err := parseRuleNativeID(request.NativeID)
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

	err = p.client.DeleteEmailRoutingRule(ctx, zoneId, ruleId)
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

// Status checks the current state of an email routing rule
func (p *routingRuleProvisioner) Status(ctx context.Context, request *resource.StatusRequest) (*resource.StatusResult, error) {
	zoneId, ruleId, err := parseRuleNativeID(request.NativeID)
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

	rule, err := p.client.GetEmailRoutingRule(ctx, zoneId, ruleId)
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

	propsJSON, _ := json.Marshal(ruleToMap(rule, zoneId))

	return &resource.StatusResult{
		ProgressResult: &resource.ProgressResult{
			Operation:          resource.OperationCheckStatus,
			OperationStatus:    resource.OperationStatusSuccess,
			NativeID:           request.NativeID,
			ResourceProperties: propsJSON,
		},
	}, nil
}

// List returns all email routing rules for a zone
func (p *routingRuleProvisioner) List(ctx context.Context, request *resource.ListRequest) (*resource.ListResult, error) {
	var cfg struct {
		ZoneId string `json:"ZoneId"`
	}
	if len(request.TargetConfig) > 0 {
		_ = json.Unmarshal(request.TargetConfig, &cfg)
	}

	if cfg.ZoneId == "" {
		return &resource.ListResult{
			NativeIDs: []string{},
		}, nil
	}

	rules, err := p.client.ListEmailRoutingRules(ctx, cfg.ZoneId)
	if err != nil {
		return nil, fmt.Errorf("failed to list email routing rules: %w", err)
	}

	var nativeIDs []string
	for _, rule := range rules {
		nativeID := fmt.Sprintf("%s/%s", cfg.ZoneId, rule.ID)
		nativeIDs = append(nativeIDs, nativeID)
	}

	return &resource.ListResult{
		NativeIDs: nativeIDs,
	}, nil
}

// ruleToMap converts a Cloudflare email routing rule to a map for JSON serialization
func ruleToMap(rule *email_routing.EmailRoutingRule, zoneId string) map[string]interface{} {
	m := map[string]interface{}{
		"id":      rule.ID,
		"tag":     rule.Tag,
		"zone_id": zoneId,
		"name":    rule.Name,
		"enabled": rule.Enabled,
	}
	if rule.Priority != 0 {
		m["priority"] = int(rule.Priority)
	}
	return m
}

func createRuleFailure(code resource.OperationErrorCode, msg string) *resource.CreateResult {
	return &resource.CreateResult{
		ProgressResult: &resource.ProgressResult{
			Operation:       resource.OperationCreate,
			OperationStatus: resource.OperationStatusFailure,
			ErrorCode:       code,
			StatusMessage:   msg,
		},
	}
}

func updateRuleFailure(nativeID string, code resource.OperationErrorCode, msg string) *resource.UpdateResult {
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
		RoutingRuleResourceType,
		[]resource.Operation{
			resource.OperationCreate,
			resource.OperationRead,
			resource.OperationUpdate,
			resource.OperationDelete,
			resource.OperationList,
			resource.OperationCheckStatus,
		},
		func(client *transport.Client) prov.Provisioner {
			return &routingRuleProvisioner{client: client}
		},
	)
}
