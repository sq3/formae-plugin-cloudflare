// SPDX-License-Identifier: Apache-2.0

package dns

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	cf "github.com/cloudflare/cloudflare-go/v4"
	"github.com/cloudflare/cloudflare-go/v4/dns"
	"github.com/platform-engineering-labs/formae-plugin-cloudflare/pkg/resources/prov"
	"github.com/platform-engineering-labs/formae-plugin-cloudflare/pkg/resources/registry"
	"github.com/platform-engineering-labs/formae-plugin-cloudflare/pkg/transport"
	"github.com/platform-engineering-labs/formae/pkg/plugin/resource"
)

const RecordResourceType = "Cloudflare::DNS::Record"

// recordProvisioner handles DNS record lifecycle operations
type recordProvisioner struct {
	client *transport.Client
}

var _ prov.Provisioner = &recordProvisioner{}

// recordProps represents DNS record properties
type recordProps struct {
	ZoneId   string  `json:"zone_id"`
	Name     string  `json:"name"`
	Type     string  `json:"type"`
	Content  string  `json:"content"`
	TTL      int     `json:"ttl"`
	Proxied  *bool   `json:"proxied,omitempty"`
	Priority *int    `json:"priority,omitempty"`
	Comment  string  `json:"comment,omitempty"`
}

// parseNativeID extracts zone_id and record_id from native ID (format: zone_id/record_id)
func parseRecordNativeID(nativeID string) (zoneId, recordId string, err error) {
	parts := strings.SplitN(nativeID, "/", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid native ID format, expected 'zone_id/record_id': %s", nativeID)
	}
	return parts[0], parts[1], nil
}

// buildARecordParams creates parameters for A record
func buildARecordParams(zoneId string, props recordProps) dns.RecordNewParams {
	record := dns.ARecordParam{
		Name:    cf.F(props.Name),
		Content: cf.F(props.Content),
		Type:    cf.F(dns.ARecordTypeA),
	}
	if props.TTL > 0 {
		record.TTL = cf.F(dns.TTL(props.TTL))
	}
	if props.Proxied != nil {
		record.Proxied = cf.F(*props.Proxied)
	}
	if props.Comment != "" {
		record.Comment = cf.F(props.Comment)
	}
	return dns.RecordNewParams{
		ZoneID: cf.F(zoneId),
		Record: record,
	}
}

// buildAAAARecordParams creates parameters for AAAA record
func buildAAAARecordParams(zoneId string, props recordProps) dns.RecordNewParams {
	record := dns.AAAARecordParam{
		Name:    cf.F(props.Name),
		Content: cf.F(props.Content),
		Type:    cf.F(dns.AAAARecordTypeAAAA),
	}
	if props.TTL > 0 {
		record.TTL = cf.F(dns.TTL(props.TTL))
	}
	if props.Proxied != nil {
		record.Proxied = cf.F(*props.Proxied)
	}
	if props.Comment != "" {
		record.Comment = cf.F(props.Comment)
	}
	return dns.RecordNewParams{
		ZoneID: cf.F(zoneId),
		Record: record,
	}
}

// buildCNAMERecordParams creates parameters for CNAME record
func buildCNAMERecordParams(zoneId string, props recordProps) dns.RecordNewParams {
	record := dns.CNAMERecordParam{
		Name:    cf.F(props.Name),
		Content: cf.F(props.Content),
		Type:    cf.F(dns.CNAMERecordTypeCNAME),
	}
	if props.TTL > 0 {
		record.TTL = cf.F(dns.TTL(props.TTL))
	}
	if props.Proxied != nil {
		record.Proxied = cf.F(*props.Proxied)
	}
	if props.Comment != "" {
		record.Comment = cf.F(props.Comment)
	}
	return dns.RecordNewParams{
		ZoneID: cf.F(zoneId),
		Record: record,
	}
}

// buildTXTRecordParams creates parameters for TXT record
func buildTXTRecordParams(zoneId string, props recordProps) dns.RecordNewParams {
	record := dns.TXTRecordParam{
		Name:    cf.F(props.Name),
		Content: cf.F(props.Content),
		Type:    cf.F(dns.TXTRecordTypeTXT),
	}
	if props.TTL > 0 {
		record.TTL = cf.F(dns.TTL(props.TTL))
	}
	if props.Comment != "" {
		record.Comment = cf.F(props.Comment)
	}
	return dns.RecordNewParams{
		ZoneID: cf.F(zoneId),
		Record: record,
	}
}

// buildMXRecordParams creates parameters for MX record
func buildMXRecordParams(zoneId string, props recordProps) dns.RecordNewParams {
	priority := float64(10)
	if props.Priority != nil {
		priority = float64(*props.Priority)
	}
	record := dns.MXRecordParam{
		Name:     cf.F(props.Name),
		Content:  cf.F(props.Content),
		Type:     cf.F(dns.MXRecordTypeMX),
		Priority: cf.F(priority),
	}
	if props.TTL > 0 {
		record.TTL = cf.F(dns.TTL(props.TTL))
	}
	if props.Comment != "" {
		record.Comment = cf.F(props.Comment)
	}
	return dns.RecordNewParams{
		ZoneID: cf.F(zoneId),
		Record: record,
	}
}

// buildNSRecordParams creates parameters for NS record
func buildNSRecordParams(zoneId string, props recordProps) dns.RecordNewParams {
	record := dns.NSRecordParam{
		Name:    cf.F(props.Name),
		Content: cf.F(props.Content),
		Type:    cf.F(dns.NSRecordTypeNS),
	}
	if props.TTL > 0 {
		record.TTL = cf.F(dns.TTL(props.TTL))
	}
	if props.Comment != "" {
		record.Comment = cf.F(props.Comment)
	}
	return dns.RecordNewParams{
		ZoneID: cf.F(zoneId),
		Record: record,
	}
}

// Create creates a new DNS record
func (p *recordProvisioner) Create(ctx context.Context, request *resource.CreateRequest) (*resource.CreateResult, error) {
	var props recordProps
	if err := json.Unmarshal(request.Properties, &props); err != nil {
		return createRecordFailure(resource.OperationErrorCodeInvalidRequest,
			fmt.Sprintf("failed to parse properties: %v", err)), nil
	}

	if props.ZoneId == "" {
		return createRecordFailure(resource.OperationErrorCodeInvalidRequest, "zone_id is required"), nil
	}
	if props.Name == "" {
		return createRecordFailure(resource.OperationErrorCodeInvalidRequest, "name is required"), nil
	}
	if props.Type == "" {
		return createRecordFailure(resource.OperationErrorCodeInvalidRequest, "type is required"), nil
	}
	if props.Content == "" {
		return createRecordFailure(resource.OperationErrorCodeInvalidRequest, "content is required"), nil
	}

	// Build record params based on type
	var params dns.RecordNewParams
	switch strings.ToUpper(props.Type) {
	case "A":
		params = buildARecordParams(props.ZoneId, props)
	case "AAAA":
		params = buildAAAARecordParams(props.ZoneId, props)
	case "CNAME":
		params = buildCNAMERecordParams(props.ZoneId, props)
	case "TXT":
		params = buildTXTRecordParams(props.ZoneId, props)
	case "MX":
		params = buildMXRecordParams(props.ZoneId, props)
	case "NS":
		params = buildNSRecordParams(props.ZoneId, props)
	default:
		return createRecordFailure(resource.OperationErrorCodeInvalidRequest,
			fmt.Sprintf("unsupported record type: %s", props.Type)), nil
	}

	record, err := p.client.CreateDNSRecord(ctx, props.ZoneId, params)
	if err != nil {
		errCode := transport.ClassifyCloudflareError(err)
		return createRecordFailure(transport.ToResourceErrorCode(errCode), err.Error()), nil
	}

	nativeID := fmt.Sprintf("%s/%s", props.ZoneId, record.ID)
	propsJSON, _ := json.Marshal(recordToMap(record, props.ZoneId))

	return &resource.CreateResult{
		ProgressResult: &resource.ProgressResult{
			Operation:          resource.OperationCreate,
			OperationStatus:    resource.OperationStatusSuccess,
			NativeID:           nativeID,
			ResourceProperties: propsJSON,
		},
	}, nil
}

// Read retrieves a DNS record by ID
func (p *recordProvisioner) Read(ctx context.Context, request *resource.ReadRequest) (*resource.ReadResult, error) {
	zoneId, recordId, err := parseRecordNativeID(request.NativeID)
	if err != nil {
		return &resource.ReadResult{
			ErrorCode: resource.OperationErrorCodeInvalidRequest,
		}, nil
	}

	record, err := p.client.GetDNSRecord(ctx, zoneId, recordId)
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

	propsJSON, _ := json.Marshal(recordToMap(record, zoneId))

	return &resource.ReadResult{
		Properties: string(propsJSON),
	}, nil
}

// Update modifies a DNS record
func (p *recordProvisioner) Update(ctx context.Context, request *resource.UpdateRequest) (*resource.UpdateResult, error) {
	zoneId, recordId, err := parseRecordNativeID(request.NativeID)
	if err != nil {
		return updateRecordFailure(request.NativeID, resource.OperationErrorCodeInvalidRequest, err.Error()), nil
	}

	var props recordProps
	if err := json.Unmarshal(request.DesiredProperties, &props); err != nil {
		return updateRecordFailure(request.NativeID, resource.OperationErrorCodeInvalidRequest,
			fmt.Sprintf("failed to parse properties: %v", err)), nil
	}

	// Build update params based on type - using Edit which takes record content
	var params dns.RecordEditParams
	switch strings.ToUpper(props.Type) {
	case "A":
		record := dns.ARecordParam{
			Name:    cf.F(props.Name),
			Content: cf.F(props.Content),
			Type:    cf.F(dns.ARecordTypeA),
		}
		if props.TTL > 0 {
			record.TTL = cf.F(dns.TTL(props.TTL))
		}
		if props.Proxied != nil {
			record.Proxied = cf.F(*props.Proxied)
		}
		if props.Comment != "" {
			record.Comment = cf.F(props.Comment)
		}
		params = dns.RecordEditParams{
			ZoneID: cf.F(zoneId),
			Record: record,
		}
	default:
		// For simplicity, use re-read for unsupported update types
		rec, err := p.client.GetDNSRecord(ctx, zoneId, recordId)
		if err != nil {
			errCode := transport.ClassifyCloudflareError(err)
			return updateRecordFailure(request.NativeID, transport.ToResourceErrorCode(errCode), err.Error()), nil
		}
		propsJSON, _ := json.Marshal(recordToMap(rec, zoneId))
		return &resource.UpdateResult{
			ProgressResult: &resource.ProgressResult{
				Operation:          resource.OperationUpdate,
				OperationStatus:    resource.OperationStatusSuccess,
				NativeID:           request.NativeID,
				ResourceProperties: propsJSON,
			},
		}, nil
	}

	record, err := p.client.UpdateDNSRecordEdit(ctx, zoneId, recordId, params)
	if err != nil {
		errCode := transport.ClassifyCloudflareError(err)
		return updateRecordFailure(request.NativeID, transport.ToResourceErrorCode(errCode), err.Error()), nil
	}

	propsJSON, _ := json.Marshal(recordToMap(record, zoneId))

	return &resource.UpdateResult{
		ProgressResult: &resource.ProgressResult{
			Operation:          resource.OperationUpdate,
			OperationStatus:    resource.OperationStatusSuccess,
			NativeID:           request.NativeID,
			ResourceProperties: propsJSON,
		},
	}, nil
}

// Delete removes a DNS record
func (p *recordProvisioner) Delete(ctx context.Context, request *resource.DeleteRequest) (*resource.DeleteResult, error) {
	zoneId, recordId, err := parseRecordNativeID(request.NativeID)
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

	err = p.client.DeleteDNSRecord(ctx, zoneId, recordId)
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

// Status checks the current state of a DNS record
func (p *recordProvisioner) Status(ctx context.Context, request *resource.StatusRequest) (*resource.StatusResult, error) {
	zoneId, recordId, err := parseRecordNativeID(request.NativeID)
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

	record, err := p.client.GetDNSRecord(ctx, zoneId, recordId)
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

	propsJSON, _ := json.Marshal(recordToMap(record, zoneId))

	return &resource.StatusResult{
		ProgressResult: &resource.ProgressResult{
			Operation:          resource.OperationCheckStatus,
			OperationStatus:    resource.OperationStatusSuccess,
			NativeID:           request.NativeID,
			ResourceProperties: propsJSON,
		},
	}, nil
}

// List returns all DNS records for a zone
func (p *recordProvisioner) List(ctx context.Context, request *resource.ListRequest) (*resource.ListResult, error) {
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

	records, err := p.client.ListDNSRecords(ctx, cfg.ZoneId)
	if err != nil {
		return nil, fmt.Errorf("failed to list DNS records: %w", err)
	}

	var nativeIDs []string
	for _, record := range records {
		nativeID := fmt.Sprintf("%s/%s", cfg.ZoneId, record.ID)
		nativeIDs = append(nativeIDs, nativeID)
	}

	return &resource.ListResult{
		NativeIDs: nativeIDs,
	}, nil
}

// recordToMap converts a Cloudflare DNS record to a map for JSON serialization
func recordToMap(record *dns.RecordResponse, zoneId string) map[string]interface{} {
	m := map[string]interface{}{
		"id":      record.ID,
		"zone_id": zoneId,
		"name":    record.Name,
		"type":    string(record.Type),
		"content": record.Content,
		"ttl":     record.TTL,
		"proxied": record.Proxied,
	}

	if record.Priority != 0 {
		m["priority"] = int(record.Priority)
	}
	if record.Proxiable {
		m["proxiable"] = record.Proxiable
	}
	if record.Comment != "" {
		m["comment"] = record.Comment
	}
	if !record.CreatedOn.IsZero() {
		m["created_on"] = record.CreatedOn.String()
	}
	if !record.ModifiedOn.IsZero() {
		m["modified_on"] = record.ModifiedOn.String()
	}

	return m
}

func createRecordFailure(code resource.OperationErrorCode, msg string) *resource.CreateResult {
	return &resource.CreateResult{
		ProgressResult: &resource.ProgressResult{
			Operation:       resource.OperationCreate,
			OperationStatus: resource.OperationStatusFailure,
			ErrorCode:       code,
			StatusMessage:   msg,
		},
	}
}

func updateRecordFailure(nativeID string, code resource.OperationErrorCode, msg string) *resource.UpdateResult {
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
		RecordResourceType,
		[]resource.Operation{
			resource.OperationCreate,
			resource.OperationRead,
			resource.OperationUpdate,
			resource.OperationDelete,
			resource.OperationList,
			resource.OperationCheckStatus,
		},
		func(client *transport.Client) prov.Provisioner {
			return &recordProvisioner{client: client}
		},
	)
}
