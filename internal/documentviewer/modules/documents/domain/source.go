package domain

// SourceName identifies which back-office system produced a document (FR4).
type SourceName string

// Source names are FR4 provenance, not upstream hostnames.
const (
	SourceSales   SourceName = "SALES"
	SourceService SourceName = "SERVICE"
)

// SourceStatus is per-source health on the aggregate (FR7), not HTTP status.
type SourceStatus string

// Source status values feed FR7 sources[], not upstream hostnames.
const (
	SourceStatusOK      SourceStatus = "OK"
	SourceStatusTimeout SourceStatus = "TIMEOUT"
	SourceStatusError   SourceStatus = "ERROR"
)

// SourceReport is FR7 per-source status on every aggregate.
type SourceReport struct {
	Name          SourceName
	Status        SourceStatus
	ErrorCode     string
	LatencyMs     int
	DocumentCount int
}
