package canalbox

import "testing"

func TestParseAccountDetails(t *testing.T) {
	raw := []byte(`{"chartData":[{"asOfDate":"As of Today at 23:&#8203;47","reportResults":"{\"groupingsDown\":{\"groupings\":[{\"label\":\"QUARKS GROUP Ltd\"}]},\"reportMetadata\":{\"currency\":\"XOF\"},\"factMap\":{\"T!T\":{\"aggregates\":[{\"value\":12345}]}}}"}]}`)

	details, err := parseAccountDetails(raw)
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}
	if details.Balance != 12345 {
		t.Fatalf("expected balance 12345, got %f", details.Balance)
	}
	if details.Currency != "XOF" {
		t.Fatalf("expected currency XOF, got %s", details.Currency)
	}
	if details.DistributorName != "QUARKS GROUP Ltd" {
		t.Fatalf("unexpected distributor name: %s", details.DistributorName)
	}
	if details.AsOfDate == "As of Today at 23:&#8203;47" {
		t.Fatal("expected as-of date html entities to be decoded")
	}
}

func TestParseAccountDetailsNoChartData(t *testing.T) {
	_, err := parseAccountDetails([]byte(`{"chartData":[]}`))
	if err == nil {
		t.Fatal("expected error for empty chartData")
	}
}

func TestParseAccountDetailsWrappedReturnValue(t *testing.T) {
	raw := []byte(`{"returnValue":{"chartData":[{"asOfDate":"As of Today at 20:00","reportResults":"{\"reportMetadata\":{\"currency\":\"XOF\"},\"factMap\":{\"T!T\":{\"aggregates\":[{\"value\":\"25000\"}]}}}"}]}}`)

	details, err := parseAccountDetails(raw)
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}
	if details.Balance != 25000 {
		t.Fatalf("expected balance 25000, got %f", details.Balance)
	}
}
