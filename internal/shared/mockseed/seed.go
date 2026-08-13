// Package mockseed holds the synthetic VINs shared by both mock upstreams (SPEC §11 Q4).
// Payloads stay in each mock package so the §5.3 shapes cannot accidentally converge.
package mockseed

import "github.com/ducnd58233/unified-document-viewer/internal/shared/common"

// Kind is which upstreams hold documents for a seeded VIN.
type Kind string

// KindNone is FR6 (both empty). SalesOnly/ServiceOnly are FR7 partial.
const (
	KindBoth        Kind = "both"
	KindSalesOnly   Kind = "sales"
	KindServiceOnly Kind = "service"
	KindNone        Kind = "none"
)

// VIN Value is 10 characters (A1).
type VIN struct {
	Value string
	Kind  Kind
}

// Named VINs are fixtures: §5.3 example, FR6 empty, FR7 one-sided.
const (
	WithDocumentsA = "1HGCM82633"
	WithDocumentsB = "2T1BURHE40"
	WithNone       = "3N1AB7AP1D"
	SalesOnly      = "JHMCM56557"
	ServiceOnly    = "WBA3A5C59E"
)

// All must stay at least 20 entries (seed contract).
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

// HasSales is true for KindBoth and KindSalesOnly.
func HasSales(k Kind) bool {
	return k == KindBoth || k == KindSalesOnly
}

// HasService is true for KindBoth and KindServiceOnly.
func HasService(k Kind) bool {
	return k == KindBoth || k == KindServiceOnly
}

// Suffix is the last 4 characters. Never log the full VIN (SPEC §8).
func Suffix(vin string) string {
	if len(vin) <= common.VinSuffixLen {
		return vin
	}
	return vin[len(vin)-common.VinSuffixLen:]
}
