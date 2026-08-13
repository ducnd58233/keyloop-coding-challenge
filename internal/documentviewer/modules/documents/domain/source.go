package domain

// SourceName identifies which back-office system produced a document (FR4).
type SourceName string

// Source names are FR4 provenance, not upstream hostnames.
const (
	SourceSales   SourceName = "SALES"
	SourceService SourceName = "SERVICE"
)
