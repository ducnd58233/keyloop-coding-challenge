package domain

// SourceName identifies which back-office system produced a document (FR4).
type SourceName string

const (
	// SourceSales is the Sales System.
	SourceSales SourceName = "SALES"
	// SourceService is the Service System.
	SourceService SourceName = "SERVICE"
)
