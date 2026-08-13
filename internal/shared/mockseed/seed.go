// Package mockseed holds the synthetic VINs shared by both mock upstreams (SPEC §11 Q4).
// Payloads stay in each mock package so the §5.3 shapes cannot accidentally converge.
package mockseed

// Kind says which upstreams hold documents for a seeded VIN.
type Kind string

const (
	// KindBoth means both upstreams return documents.
	KindBoth Kind = "both"
	// KindSalesOnly means only the sales mock returns documents.
	KindSalesOnly Kind = "sales"
	// KindServiceOnly means only the service mock returns attachments.
	KindServiceOnly Kind = "service"
	// KindNone means both upstreams return empty lists (FR6).
	KindNone Kind = "none"
)

// VIN is one seeded vehicle. Value is 10 characters (A1).
type VIN struct {
	Value string
	Kind  Kind
}

const (
	// WithDocumentsA has records in both Sales and Service (SYSTEM_DESIGN §5.3 example VIN).
	WithDocumentsA = "1HGCM82633"
	// WithDocumentsB has different document types on each upstream.
	WithDocumentsB = "2T1BURHE40"
	// WithNone has zero documents on both upstreams (FR6).
	WithNone = "3N1AB7AP1D"
	// SalesOnly has sales records and an empty service list (FR7 partial).
	SalesOnly = "JHMCM56557"
	// ServiceOnly has service attachments and an empty sales list (FR7 partial).
	ServiceOnly = "WBA3A5C59E"
)

// All is the shared seed set. Length is part of the contract: at least 20 VINs.
var All = []VIN{
	{Value: WithDocumentsA, Kind: KindBoth},
	{Value: WithDocumentsB, Kind: KindBoth},
	{Value: WithNone, Kind: KindNone},
	{Value: SalesOnly, Kind: KindSalesOnly},
	{Value: ServiceOnly, Kind: KindServiceOnly},
	{Value: "4T1BF1FK5C", Kind: KindBoth},
	{Value: "5YJSA1E14H", Kind: KindBoth},
	{Value: "JM1BL1SF3A", Kind: KindBoth},
	{Value: "KNDJP3A59E", Kind: KindSalesOnly},
	{Value: "NMTKHMBX1R", Kind: KindServiceOnly},
	{Value: "SALVP2BG1F", Kind: KindBoth},
	{Value: "VF3L1XHA0A", Kind: KindBoth},
	{Value: "WAUZZZ8K9A", Kind: KindSalesOnly},
	{Value: "YV1MS3901A", Kind: KindServiceOnly},
	{Value: "ZFF65LJA0A", Kind: KindBoth},
	{Value: "1FTFW1ET1E", Kind: KindBoth},
	{Value: "2G1FC3D33B", Kind: KindNone},
	{Value: "3FA6P0H76E", Kind: KindBoth},
	{Value: "6G2EC57Y09", Kind: KindSalesOnly},
	{Value: "7FARW2H89E", Kind: KindServiceOnly},
	{Value: "8AJBA3F59A", Kind: KindBoth},
}

// HasSales reports whether the sales mock should return documents.
func HasSales(k Kind) bool {
	return k == KindBoth || k == KindSalesOnly
}

// HasService reports whether the service mock should return attachments.
func HasService(k Kind) bool {
	return k == KindBoth || k == KindServiceOnly
}

// Suffix returns the last 4 characters for logs. Never log the full VIN (SPEC §8).
func Suffix(vin string) string {
	if len(vin) <= 4 {
		return vin
	}
	return vin[len(vin)-4:]
}
