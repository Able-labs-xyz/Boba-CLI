package formatter

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/guptarohit/asciigraph"
	"github.com/tradeboba/boba-cli/internal/ui"
)

// FormatPnLChart renders a P&L chart with an ASCII graph, sparkline, and
// summary statistics. Handles MCP response format:
//
//	{
//	  "timeframe": "1W",
//	  "start_value_usd": 1825.00,
//	  "current_value_usd": 2150.50,
//	  "pnl_usd": 325.50,
//	  "pnl_percent": 17.8,
//	  "data_points": [ { "timestamp": "...", "value_usd": 1825.00 }, ... ]
//	}
func FormatPnLChart(data map[string]any) string {
	// Try extracting from data_points array of objects first
	values := extractValueFromObjects(data, "data_points", "value_usd")
	// Fall back to flat arrays
	if len(values) == 0 {
		values = extractFloatSlice(data, "chart", "data", "values", "points", "data_points")
	}
	if len(values) == 0 {
		return ui.DimStyle.Render("No chart data available.")
	}

	// Plot the chart
	plot := asciigraph.Plot(values,
		asciigraph.Height(10),
		asciigraph.Caption("P&L"),
	)

	// Sparkline of last 30 points
	sparkValues := values
	if len(sparkValues) > 30 {
		sparkValues = sparkValues[len(sparkValues)-30:]
	}
	sparkline := "Trend: " + Sparkline(sparkValues)

	// Summary statistics
	startVal := values[0]
	endVal := values[len(values)-1]
	change := 0.0
	if startVal != 0 {
		change = ((endVal - startVal) / math.Abs(startVal)) * 100
	}

	high := values[0]
	low := values[0]
	for _, v := range values {
		if v > high {
			high = v
		}
		if v < low {
			low = v
		}
	}

	labelStyle := lipgloss.NewStyle().Foreground(ui.ColorBright).Bold(true).Width(10)
	summary := strings.Join([]string{
		labelStyle.Render("Start") + FormatUSD(startVal),
		labelStyle.Render("End") + FormatUSD(endVal),
		labelStyle.Render("Change") + FormatPercent(change),
		labelStyle.Render("High") + FormatUSD(high),
		labelStyle.Render("Low") + FormatUSD(low),
	}, "\n")

	title := ui.TitleStyle.Render("P&L CHART")

	content := lipgloss.JoinVertical(lipgloss.Left,
		title,
		"",
		plot,
		"",
		sparkline,
		"",
		summary,
	)

	return ui.BoxBorder.Render(content)
}

// FormatTokenChart renders a token price chart from OHLC/candle data with
// ASCII graph, sparkline, and price statistics.
func FormatTokenChart(data map[string]any) string {
	// Try multiple possible keys for candle data
	var rawCandles []any
	for _, key := range []string{"candles", "ohlc", "data", "chart", "bars"} {
		if v, ok := data[key].([]any); ok {
			rawCandles = v
			break
		}
	}

	if len(rawCandles) == 0 {
		return ui.DimStyle.Render("No chart data available.")
	}

	// Extract close prices from candle data
	var values []float64
	for _, c := range rawCandles {
		switch candle := c.(type) {
		case map[string]any:
			// Try "close", "c", "price" keys
			closePrice := getFloat(candle, "close")
			if closePrice == 0 {
				closePrice = getFloat(candle, "c")
			}
			if closePrice == 0 {
				closePrice = getFloat(candle, "price")
			}
			if closePrice != 0 {
				values = append(values, closePrice)
			}
		case []any:
			// OHLCV array format: [open, high, low, close, volume]
			if len(candle) > 4 {
				if closeVal, ok := toFloat64(candle[4]); ok && closeVal != 0 {
					values = append(values, closeVal)
				}
			} else if len(candle) > 3 {
				if closeVal, ok := toFloat64(candle[3]); ok && closeVal != 0 {
					values = append(values, closeVal)
				}
			}
		case float64:
			// Plain array of numbers
			if candle != 0 {
				values = append(values, candle)
			}
		case string:
			// Plain array of string numbers
			if f, err := strconv.ParseFloat(candle, 64); err == nil && f != 0 {
				values = append(values, f)
			}
		}
	}

	if len(values) == 0 {
		return ui.DimStyle.Render("No price data available.")
	}

	// Plot the chart
	plot := asciigraph.Plot(values, asciigraph.Height(12))

	// Sparkline
	sparkValues := values
	if len(sparkValues) > 30 {
		sparkValues = sparkValues[len(sparkValues)-30:]
	}
	sparkline := "Trend: " + Sparkline(sparkValues)

	// Statistics
	openPrice := values[0]
	closePrice := values[len(values)-1]
	change := 0.0
	if openPrice != 0 {
		change = ((closePrice - openPrice) / math.Abs(openPrice)) * 100
	}

	high := values[0]
	low := values[0]
	for _, v := range values {
		if v > high {
			high = v
		}
		if v < low {
			low = v
		}
	}

	labelStyle := lipgloss.NewStyle().Foreground(ui.ColorBright).Bold(true).Width(10)

	summary := strings.Join([]string{
		labelStyle.Render("Open") + smartFormatPrice(openPrice),
		labelStyle.Render("Close") + smartFormatPrice(closePrice),
		labelStyle.Render("Change") + FormatPercent(change),
		labelStyle.Render("High") + smartFormatPrice(high),
		labelStyle.Render("Low") + smartFormatPrice(low),
	}, "\n")

	title := ui.TitleStyle.Render("PRICE CHART")

	content := lipgloss.JoinVertical(lipgloss.Left,
		title,
		"",
		plot,
		"",
		sparkline,
		"",
		summary,
	)

	return ui.BoxBorder.Render(content)
}

// smartFormatPrice formats a price with appropriate decimal places based on
// magnitude: >= $1 uses 2 decimals, $0.01-$1 uses 4 decimals, < $0.01 uses
// 8 decimals.
func smartFormatPrice(price float64) string {
	style := lipgloss.NewStyle().Foreground(ui.ColorGold)
	abs := math.Abs(price)

	var formatted string
	switch {
	case abs >= 1:
		formatted = fmt.Sprintf("$%.2f", price)
	case abs >= 0.01:
		formatted = fmt.Sprintf("$%.4f", price)
	default:
		formatted = fmt.Sprintf("$%.8f", price)
	}

	return style.Render(formatted)
}

// extractValueFromObjects extracts a float field from an array of objects.
// e.g., extractValueFromObjects(data, "data_points", "value_usd") extracts
// value_usd from each object in the data_points array.
func extractValueFromObjects(data map[string]any, arrayKey, fieldKey string) []float64 {
	arr, ok := data[arrayKey].([]any)
	if !ok || len(arr) == 0 {
		return nil
	}
	var result []float64
	for _, item := range arr {
		obj, ok := item.(map[string]any)
		if !ok {
			continue
		}
		val := getFloat(obj, fieldKey)
		if val != 0 {
			result = append(result, val)
		}
	}
	return result
}

// toFloat64 converts an interface value to float64, handling both numeric types
// and string representations.
func toFloat64(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	case string:
		f, err := strconv.ParseFloat(n, 64)
		return f, err == nil
	default:
		return 0, false
	}
}

// extractFloatSlice tries multiple keys to find a slice of float64 values
// in the data map. Handles both numeric and string values.
func extractFloatSlice(data map[string]any, keys ...string) []float64 {
	for _, key := range keys {
		if raw, ok := data[key]; ok {
			switch v := raw.(type) {
			case []any:
				var result []float64
				for _, item := range v {
					if f, ok := toFloat64(item); ok {
						result = append(result, f)
					}
				}
				if len(result) > 0 {
					return result
				}
			case []float64:
				return v
			}
		}
	}
	return nil
}
