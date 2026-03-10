package canalbox

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html"
	"strconv"
	"sync/atomic"
)

const (
	defaultHomePageURI        = "/PortailDistributeur/s/"
	defaultAccountReportID    = "00O6M000008qxjjUAA"
	defaultChartInstanceID    = "0LGIV00001B3YSz"
	defaultChartComponentID   = "2:1148;a"
	defaultChartLoadQueryPart = "ui-analytics-platform-embeddedChart.EmbeddedReportChart.loadChartInstance=1"
)

type AccountDetails struct {
	Balance         float64
	Currency        string
	DistributorName string
	AsOfDate        string
}

func (c *Client) GetAccountDetails(ctx context.Context) (*AccountDetails, error) {
	action := map[string]any{
		"id":                fmt.Sprintf("%d;a", atomic.AddInt64(&c.auraActionCounter, 1)),
		"descriptor":        "serviceComponent://ui.analytics.platform.embeddedChart.EmbeddedReportChartController/ACTION$loadChartInstance",
		"callingDescriptor": "UNKNOWN",
		"params": map[string]any{
			"reportIdOrDeveloperName": defaultAccountReportID,
			"embeddedChartInput": map[string]any{
				"instanceId":        defaultChartInstanceID,
				"componentId":       defaultChartComponentID,
				"size":              "auto",
				"widthOverrides":    []float64{302, 846.9},
				"isInteractive":     true,
				"cacheAge":          86400000,
				"origin":            "earc",
				"isInS1Context":     false,
				"renderEclairChart": true,
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
	var wrapper struct {
		ChartData []struct {
			AsOfDate      string          `json:"asOfDate"`
			ReportResults json.RawMessage `json:"reportResults"`
		} `json:"chartData"`
	}
	if err := decodeReturnValue(raw, &wrapper); err != nil {
		return nil, fmt.Errorf("parse account details: %w", err)
	}
	if len(wrapper.ChartData) == 0 {
		return nil, fmt.Errorf("account details response has no chart data")
	}
	if len(wrapper.ChartData[0].ReportResults) == 0 {
		return nil, fmt.Errorf("account details response has empty report results")
	}

	reportPayload := wrapper.ChartData[0].ReportResults
	var reportString string
	if err := json.Unmarshal(wrapper.ChartData[0].ReportResults, &reportString); err == nil {
		reportString = html.UnescapeString(reportString)
		reportPayload = []byte(reportString)
	}

	var report struct {
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
	if err := decodeJSON(reportPayload, &report); err != nil {
		return nil, fmt.Errorf("parse report results: %w", err)
	}

	balance := 0.0
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

	name := ""
	if len(report.GroupingsDown.Groupings) > 0 {
		name = report.GroupingsDown.Groupings[0].Label
	}

	return &AccountDetails{
		Balance:         balance,
		Currency:        report.ReportMetadata.Currency,
		DistributorName: name,
		AsOfDate:        html.UnescapeString(wrapper.ChartData[0].AsOfDate),
	}, nil
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
