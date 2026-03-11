package canalbox

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html"
	"math"
	"strconv"
	"sync/atomic"
)

const (
	defaultHomePageURI        = "/PortailDistributeur/s/"
	defaultAccountReportID    = "00O6M000008qxjjUAA"
	defaultChartLoadQueryPart = "ui-analytics-platform-embeddedChart.EmbeddedReportChart.loadChart=1"
)

type AccountDetails struct {
	Balance         float64
	Currency        string
	DistributorName string
	AsOfDate        string
}

func (c *Client) GetAccountDetails(ctx context.Context) (*AccountDetails, error) {
	action := map[string]any{
		"id":         fmt.Sprintf("%d;a", atomic.AddInt64(&c.auraActionCounter, 1)),
		"descriptor": "serviceComponent://ui.analytics.platform.embeddedChart.EmbeddedReportChartController/ACTION$loadChart",
		"params": map[string]any{
			"reportIdOrDeveloperName": defaultAccountReportID,
			"embeddedChartInput": map[string]any{
				"size":          "auto",
				"isInteractive": true,
				"cacheAge":      86400000,
				"isInS1Context": false,
			},
		},
	}

	response, err := c.doAuraRequest(ctx, defaultHomePageURI, defaultChartLoadQueryPart, action, "")
	if err != nil {
		return nil, err
	}

	a, err := firstAction(response)
	if err != nil {
		return nil, err
	}

	return parseAccountDetails(a.ReturnValue)
}

func parseAccountDetails(raw json.RawMessage) (*AccountDetails, error) {
	payload := unwrapActionReturnValue(raw)

	var legacy struct {
		ChartData []struct {
			AsOfDate      string          `json:"asOfDate"`
			ReportResults json.RawMessage `json:"reportResults"`
		} `json:"chartData"`
	}
	if err := decodeReturnValue(payload, &legacy); err != nil {
		return nil, fmt.Errorf("parse account details: %w", err)
	}

	if len(legacy.ChartData) > 0 {
		return parseLegacyAccountDetails(legacy.ChartData[0].AsOfDate, legacy.ChartData[0].ReportResults)
	}

	var dashboard struct {
		ComponentData []struct {
			ReportResult json.RawMessage `json:"reportResult"`
		} `json:"componentData"`
	}
	if err := decodeReturnValue(payload, &dashboard); err != nil {
		return nil, fmt.Errorf("parse account details: %w", err)
	}
	if len(dashboard.ComponentData) == 0 {
		return nil, fmt.Errorf("account details response has no chart data")
	}

	for _, component := range dashboard.ComponentData {
		if len(component.ReportResult) == 0 {
			continue
		}
		report, err := decodeAccountReport(component.ReportResult)
		if err != nil {
			continue
		}
		balance, distributorName := selectAccountBalance(report)
		if balance == 0 {
			continue
		}

		return &AccountDetails{
			Balance:         balance,
			Currency:        report.ReportMetadata.Currency,
			DistributorName: distributorName,
		}, nil
	}

	return nil, fmt.Errorf("account details response has empty report results")
}

func unwrapActionReturnValue(raw json.RawMessage) json.RawMessage {
	var envelope struct {
		Actions []struct {
			ReturnValue json.RawMessage `json:"returnValue"`
		} `json:"actions"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return raw
	}
	if len(envelope.Actions) == 0 || len(envelope.Actions[0].ReturnValue) == 0 {
		return raw
	}
	return envelope.Actions[0].ReturnValue
}

func parseLegacyAccountDetails(asOfDate string, reportPayload json.RawMessage) (*AccountDetails, error) {
	if len(reportPayload) == 0 {
		return nil, fmt.Errorf("account details response has empty report results")
	}

	var reportString string
	if err := json.Unmarshal(reportPayload, &reportString); err == nil {
		reportString = html.UnescapeString(reportString)
		reportPayload = []byte(reportString)
	}

	report, err := decodeAccountReport(reportPayload)
	if err != nil {
		return nil, fmt.Errorf("parse report results: %w", err)
	}

	balance, distributorName := selectAccountBalance(report)

	return &AccountDetails{
		Balance:         balance,
		Currency:        report.ReportMetadata.Currency,
		DistributorName: distributorName,
		AsOfDate:        html.UnescapeString(asOfDate),
	}, nil
}

type accountReport struct {
	ReportMetadata struct {
		Currency string `json:"currency"`
	} `json:"reportMetadata"`
	GroupingsDown struct {
		Groupings []struct {
			Label string `json:"label"`
		} `json:"groupings"`
	} `json:"groupingsDown"`
	FactMap map[string]struct {
		Aggregates []struct {
			Value any `json:"value"`
		} `json:"aggregates"`
	} `json:"factMap"`
}

func decodeAccountReport(reportPayload []byte) (*accountReport, error) {
	var report accountReport
	if err := decodeJSON(reportPayload, &report); err != nil {
		return nil, err
	}
	return &report, nil
}

func selectAccountBalance(report *accountReport) (float64, string) {
	balance := 0.0
	name := ""

	if len(report.GroupingsDown.Groupings) > 0 {
		bestValue := math.MaxFloat64
		for i, grouping := range report.GroupingsDown.Groupings {
			key := fmt.Sprintf("%d!T", i)
			bucket, ok := report.FactMap[key]
			if !ok || len(bucket.Aggregates) == 0 {
				continue
			}

			value := toFloat(bucket.Aggregates[0].Value)
			if value <= 0 {
				continue
			}
			if value < bestValue {
				bestValue = value
				balance = value
				name = grouping.Label
			}
		}

		if balance > 0 {
			return balance, name
		}

		name = report.GroupingsDown.Groupings[0].Label
	}

	if total, ok := report.FactMap["T!T"]; ok && len(total.Aggregates) > 0 {
		balance = toFloat(total.Aggregates[0].Value)
	} else {
		for _, bucket := range report.FactMap {
			if len(bucket.Aggregates) > 0 {
				balance = toFloat(bucket.Aggregates[0].Value)
				break
			}
		}
	}

	return balance, name
}

func decodeJSON(payload []byte, target any) error {
	dec := json.NewDecoder(bytes.NewReader(payload))
	dec.UseNumber()
	return dec.Decode(target)
}

func toFloat(v any) float64 {
	switch n := v.(type) {
	case float64:
		return n
	case json.Number:
		f, err := n.Float64()
		if err == nil {
			return f
		}
	case string:
		f, err := strconv.ParseFloat(n, 64)
		if err == nil {
			return f
		}
	}
	return 0
}
