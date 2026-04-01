// SPDX-License-Identifier: Apache-2.0

package transport

import (
	"context"
	"fmt"

	cf "github.com/cloudflare/cloudflare-go/v4"
	"github.com/cloudflare/cloudflare-go/v4/dns"
	"github.com/cloudflare/cloudflare-go/v4/email_routing"
	"github.com/cloudflare/cloudflare-go/v4/option"
	"github.com/cloudflare/cloudflare-go/v4/zones"
)

// Client wraps the Cloudflare SDK client
type Client struct {
	cf        *cf.Client
	accountId string
}

// ClientConfig holds connection parameters for the Cloudflare API
type ClientConfig struct {
	ApiToken  string
	AccountId string
}

// NewClient creates a new Cloudflare API client
func NewClient(cfg *ClientConfig) (*Client, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is nil")
	}
	if cfg.ApiToken == "" {
		return nil, fmt.Errorf("API token is required")
	}

	client := cf.NewClient(
		option.WithAPIToken(cfg.ApiToken),
	)

	return &Client{
		cf:        client,
		accountId: cfg.AccountId,
	}, nil
}

// AccountId returns the configured account ID
func (c *Client) AccountId() string {
	return c.accountId
}

// =============================================================================
// Zone Operations
// =============================================================================

// CreateZone creates a new DNS zone
func (c *Client) CreateZone(ctx context.Context, name, accountId, zoneType string) (*zones.Zone, error) {
	zone, err := c.cf.Zones.New(ctx, zones.ZoneNewParams{
		Account: cf.F(zones.ZoneNewParamsAccount{
			ID: cf.F(accountId),
		}),
		Name: cf.F(name),
		Type: cf.F(zones.Type(zoneType)),
	})
	if err != nil {
		return nil, err
	}
	return zone, nil
}

// GetZone retrieves a zone by ID
func (c *Client) GetZone(ctx context.Context, zoneId string) (*zones.Zone, error) {
	zone, err := c.cf.Zones.Get(ctx, zones.ZoneGetParams{
		ZoneID: cf.F(zoneId),
	})
	if err != nil {
		return nil, err
	}
	return zone, nil
}

// UpdateZone updates a zone's settings (limited to type and vanity_name_servers in v4 API)
func (c *Client) UpdateZone(ctx context.Context, zoneId string, params zones.ZoneEditParams) (*zones.Zone, error) {
	params.ZoneID = cf.F(zoneId)
	zone, err := c.cf.Zones.Edit(ctx, params)
	if err != nil {
		return nil, err
	}
	return zone, nil
}

// DeleteZone removes a zone
func (c *Client) DeleteZone(ctx context.Context, zoneId string) error {
	_, err := c.cf.Zones.Delete(ctx, zones.ZoneDeleteParams{
		ZoneID: cf.F(zoneId),
	})
	return err
}

// ListZones returns all zones for an account
func (c *Client) ListZones(ctx context.Context, accountId string) ([]*zones.Zone, error) {
	var allZones []*zones.Zone

	iter := c.cf.Zones.ListAutoPaging(ctx, zones.ZoneListParams{
		Account: cf.F(zones.ZoneListParamsAccount{
			ID: cf.F(accountId),
		}),
	})
	for iter.Next() {
		zone := iter.Current()
		allZones = append(allZones, &zone)
	}
	if err := iter.Err(); err != nil {
		return nil, err
	}

	return allZones, nil
}

// =============================================================================
// DNS Record Operations
// =============================================================================

// CreateDNSRecord creates a new DNS record
func (c *Client) CreateDNSRecord(ctx context.Context, zoneId string, params dns.RecordNewParams) (*dns.RecordResponse, error) {
	record, err := c.cf.DNS.Records.New(ctx, params)
	if err != nil {
		return nil, err
	}
	return record, nil
}

// GetDNSRecord retrieves a DNS record by ID
func (c *Client) GetDNSRecord(ctx context.Context, zoneId, recordId string) (*dns.RecordResponse, error) {
	record, err := c.cf.DNS.Records.Get(ctx, recordId, dns.RecordGetParams{
		ZoneID: cf.F(zoneId),
	})
	if err != nil {
		return nil, err
	}
	return record, nil
}

// UpdateDNSRecord updates a DNS record using the Update method
func (c *Client) UpdateDNSRecord(ctx context.Context, zoneId, recordId string, params dns.RecordUpdateParams) (*dns.RecordResponse, error) {
	record, err := c.cf.DNS.Records.Update(ctx, recordId, params)
	if err != nil {
		return nil, err
	}
	return record, nil
}

// UpdateDNSRecordEdit updates a DNS record using the Edit method
func (c *Client) UpdateDNSRecordEdit(ctx context.Context, zoneId, recordId string, params dns.RecordEditParams) (*dns.RecordResponse, error) {
	record, err := c.cf.DNS.Records.Edit(ctx, recordId, params)
	if err != nil {
		return nil, err
	}
	return record, nil
}

// DeleteDNSRecord removes a DNS record
func (c *Client) DeleteDNSRecord(ctx context.Context, zoneId, recordId string) error {
	_, err := c.cf.DNS.Records.Delete(ctx, recordId, dns.RecordDeleteParams{
		ZoneID: cf.F(zoneId),
	})
	return err
}

// ListDNSRecords returns all DNS records for a zone
func (c *Client) ListDNSRecords(ctx context.Context, zoneId string) ([]*dns.RecordResponse, error) {
	var allRecords []*dns.RecordResponse

	iter := c.cf.DNS.Records.ListAutoPaging(ctx, dns.RecordListParams{
		ZoneID: cf.F(zoneId),
	})
	for iter.Next() {
		record := iter.Current()
		allRecords = append(allRecords, &record)
	}
	if err := iter.Err(); err != nil {
		return nil, err
	}

	return allRecords, nil
}

// =============================================================================
// Email Routing Operations
// =============================================================================

// CreateEmailRoutingRule creates a new email routing rule
func (c *Client) CreateEmailRoutingRule(ctx context.Context, zoneId string, params email_routing.RuleNewParams) (*email_routing.EmailRoutingRule, error) {
	rule, err := c.cf.EmailRouting.Rules.New(ctx, params)
	if err != nil {
		return nil, err
	}
	return rule, nil
}

// GetEmailRoutingRule retrieves an email routing rule by ID
func (c *Client) GetEmailRoutingRule(ctx context.Context, zoneId, ruleId string) (*email_routing.EmailRoutingRule, error) {
	rule, err := c.cf.EmailRouting.Rules.Get(ctx, ruleId, email_routing.RuleGetParams{
		ZoneID: cf.F(zoneId),
	})
	if err != nil {
		return nil, err
	}
	return rule, nil
}

// UpdateEmailRoutingRule updates an email routing rule
func (c *Client) UpdateEmailRoutingRule(ctx context.Context, zoneId, ruleId string, params email_routing.RuleUpdateParams) (*email_routing.EmailRoutingRule, error) {
	rule, err := c.cf.EmailRouting.Rules.Update(ctx, ruleId, params)
	if err != nil {
		return nil, err
	}
	return rule, nil
}

// DeleteEmailRoutingRule removes an email routing rule
func (c *Client) DeleteEmailRoutingRule(ctx context.Context, zoneId, ruleId string) error {
	_, err := c.cf.EmailRouting.Rules.Delete(ctx, ruleId, email_routing.RuleDeleteParams{
		ZoneID: cf.F(zoneId),
	})
	return err
}

// ListEmailRoutingRules returns all email routing rules for a zone
func (c *Client) ListEmailRoutingRules(ctx context.Context, zoneId string) ([]*email_routing.EmailRoutingRule, error) {
	var allRules []*email_routing.EmailRoutingRule

	iter := c.cf.EmailRouting.Rules.ListAutoPaging(ctx, email_routing.RuleListParams{
		ZoneID: cf.F(zoneId),
	})
	for iter.Next() {
		rule := iter.Current()
		allRules = append(allRules, &rule)
	}
	if err := iter.Err(); err != nil {
		return nil, err
	}

	return allRules, nil
}

// CreateDestinationAddress creates a new destination address for email routing
func (c *Client) CreateDestinationAddress(ctx context.Context, accountId, email string) (*email_routing.Address, error) {
	addr, err := c.cf.EmailRouting.Addresses.New(ctx, email_routing.AddressNewParams{
		AccountID: cf.F(accountId),
		Email:     cf.F(email),
	})
	if err != nil {
		return nil, err
	}
	return addr, nil
}

// GetDestinationAddress retrieves a destination address by ID
func (c *Client) GetDestinationAddress(ctx context.Context, accountId, addressId string) (*email_routing.Address, error) {
	addr, err := c.cf.EmailRouting.Addresses.Get(ctx, addressId, email_routing.AddressGetParams{
		AccountID: cf.F(accountId),
	})
	if err != nil {
		return nil, err
	}
	return addr, nil
}

// DeleteDestinationAddress removes a destination address
func (c *Client) DeleteDestinationAddress(ctx context.Context, accountId, addressId string) error {
	_, err := c.cf.EmailRouting.Addresses.Delete(ctx, addressId, email_routing.AddressDeleteParams{
		AccountID: cf.F(accountId),
	})
	return err
}

// ListDestinationAddresses returns all destination addresses for an account
func (c *Client) ListDestinationAddresses(ctx context.Context, accountId string) ([]*email_routing.Address, error) {
	var allAddresses []*email_routing.Address

	iter := c.cf.EmailRouting.Addresses.ListAutoPaging(ctx, email_routing.AddressListParams{
		AccountID: cf.F(accountId),
	})
	for iter.Next() {
		addr := iter.Current()
		allAddresses = append(allAddresses, &addr)
	}
	if err := iter.Err(); err != nil {
		return nil, err
	}

	return allAddresses, nil
}
