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

var _ app.DocumentSource = (*ServiceClient)(nil)

// ServiceClient is the Service System adapter (FR5). Per-source budget lives on ctx.
type ServiceClient struct {
	base   string
	client *httpclient.Client
}

// NewServiceClient talks to baseURL. Timeouts are not set on the client; Aggregate owns NFR2.
func NewServiceClient(baseURL string) *ServiceClient {
	return &ServiceClient{
		base:   strings.TrimRight(baseURL, "/"),
		client: httpclient.New(),
	}
}

// Name is FR4 provenance.
func (c *ServiceClient) Name() domain.SourceName { return domain.SourceService }

// Fetch never returns hostnames or the VIN in error text.
func (c *ServiceClient) Fetch(ctx context.Context, vin string) ([]domain.Document, error) {
	path, err := url.JoinPath(c.base, "service", "v1", "vehicles", vin, "attachments")
	if err != nil {
		return nil, errors.New("service url invalid")
	}
	req, err := nethttp.NewRequestWithContext(ctx, nethttp.MethodGet, path, nil)
	if err != nil {
		return nil, errors.New("service request failed")
	}
	var payload serviceList
	if err := c.client.DoJSON(ctx, req, &payload); err != nil {
		return nil, err
	}
	return normalizeService(payload.Attachments), nil
}
