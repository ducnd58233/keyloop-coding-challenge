// Package http holds Sales and Service upstream clients and FR5 normalisers.
package http

import (
	"context"
	"errors"
	nethttp "net/http"
	"net/url"
	"strings"

	"github.com/ducnd58233/unified-document-viewer/internal/documentviewer/modules/documents/app"
	"github.com/ducnd58233/unified-document-viewer/internal/documentviewer/modules/documents/domain"
	"github.com/ducnd58233/unified-document-viewer/internal/shared/infra/httpclient"
)

var _ app.DocumentSource = (*SalesClient)(nil)

// SalesClient is the Sales System adapter (FR5). Per-source budget lives on ctx.
type SalesClient struct {
	base   string
	client *httpclient.Client
}

// NewSalesClient talks to baseURL. Timeouts are not set on the client; Aggregate owns NFR2.
func NewSalesClient(baseURL string) *SalesClient {
	return &SalesClient{
		base:   strings.TrimRight(baseURL, "/"),
		client: httpclient.New(),
	}
}

// Name is FR4 provenance, not a hostname.
func (c *SalesClient) Name() domain.SourceName { return domain.SourceSales }

// Fetch never returns hostnames or the VIN in error text.
func (c *SalesClient) Fetch(ctx context.Context, vin string) ([]domain.Document, error) {
	u, err := url.Parse(c.base + "/sales/v1/documents")
	if err != nil {
		return nil, errors.New("sales url invalid")
	}
	q := u.Query()
	q.Set("vin", vin)
	u.RawQuery = q.Encode()

	req, err := nethttp.NewRequestWithContext(ctx, nethttp.MethodGet, u.String(), nil)
	if err != nil {
		return nil, errors.New("sales request failed")
	}
	var payload salesList
	if err := c.client.DoJSON(ctx, req, &payload); err != nil {
		return nil, err
	}
	return normalizeSales(c.base, payload.Records), nil
}
