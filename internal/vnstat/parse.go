package vnstat

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
)

// Layouts of the labels periodLabel builds, used to look up the current
// periods for the summary.
const (
	hourLayout  = "2006-01-02 15:00"
	dayLayout   = "2006-01-02"
	monthLayout = "2006-01"
)

type jsonReport struct {
	VnstatVersion string          `json:"vnstatversion"`
	Interfaces    []jsonInterface `json:"interfaces"`
}

type jsonInterface struct {
	Name    string      `json:"name"`
	Traffic jsonTraffic `json:"traffic"`
}

type jsonTraffic struct {
	Total jsonTotal    `json:"total"`
	Hour  []jsonSample `json:"hour"`
	Day   []jsonSample `json:"day"`
	Month []jsonSample `json:"month"`
}

type jsonTotal struct {
	Rx uint64 `json:"rx"`
	Tx uint64 `json:"tx"`
}

type jsonSample struct {
	Date jsonDate `json:"date"`
	Time jsonTime `json:"time"`
	Rx   uint64   `json:"rx"`
	Tx   uint64   `json:"tx"`
}

type jsonDate struct {
	Year  int `json:"year"`
	Month int `json:"month"`
	Day   int `json:"day"`
}

type jsonTime struct {
	Hour int `json:"hour"`
}

func parseReport(dayJSON, monthJSON, hourJSON []byte, now time.Time) (Report, error) {
	days, err := parsePeriods(dayJSON, periodDay)
	if err != nil {
		return Report{}, err
	}

	months, err := parsePeriods(monthJSON, periodMonth)
	if err != nil {
		return Report{}, err
	}

	hours, err := parsePeriods(hourJSON, periodHour)
	if err != nil {
		return Report{}, err
	}

	version := ""
	var allTime Traffic
	if len(dayJSON) > 0 {
		var meta jsonReport
		if err := json.Unmarshal(dayJSON, &meta); err == nil {
			version = meta.VnstatVersion
			allTime = totalTraffic(meta.Interfaces)
		}
	}

	return Report{
		Installed: true,
		Version:   version,
		Days:      days,
		Months:    months,
		Summary: &Summary{
			ThisHour:  findTraffic(hours, now.Format(hourLayout)),
			LastHour:  findTraffic(hours, now.Add(-time.Hour).Format(hourLayout)),
			ThisDay:   findTraffic(days, now.Format(dayLayout)),
			ThisMonth: findTraffic(months, now.Format(monthLayout)),
			AllTime:   allTime,
		},
	}, nil
}

type periodKind int

const (
	periodDay periodKind = iota
	periodMonth
	periodHour
)

func parsePeriods(data []byte, kind periodKind) ([]TrafficPeriod, error) {
	var raw jsonReport
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse vnstat json: %w", err)
	}

	totals := map[string]TrafficPeriod{}

	for _, iface := range raw.Interfaces {
		if skipInterface(iface.Name) {
			continue
		}

		var samples []jsonSample
		switch kind {
		case periodDay:
			samples = iface.Traffic.Day
		case periodMonth:
			samples = iface.Traffic.Month
		case periodHour:
			samples = iface.Traffic.Hour
		}

		for _, sample := range samples {
			label, err := periodLabel(sample, kind)
			if err != nil {
				continue
			}

			existing := totals[label]
			existing.Label = label
			existing.RxBytes += sample.Rx
			existing.TxBytes += sample.Tx
			totals[label] = existing
		}
	}

	periods := make([]TrafficPeriod, 0, len(totals))
	for _, period := range totals {
		periods = append(periods, period)
	}

	sort.Slice(periods, func(i, j int) bool {
		return periods[i].Label < periods[j].Label
	})

	return periods, nil
}

func periodLabel(sample jsonSample, kind periodKind) (string, error) {
	date := sample.Date
	if date.Year == 0 || date.Month == 0 {
		return "", fmt.Errorf("invalid date")
	}
	if kind == periodMonth {
		return fmt.Sprintf("%04d-%02d", date.Year, date.Month), nil
	}
	if date.Day == 0 {
		return "", fmt.Errorf("invalid day")
	}
	if kind == periodHour {
		return fmt.Sprintf("%04d-%02d-%02d %02d:00", date.Year, date.Month, date.Day, sample.Time.Hour), nil
	}
	return fmt.Sprintf("%04d-%02d-%02d", date.Year, date.Month, date.Day), nil
}

// totalTraffic sums the all-time counters vnstat includes in every --json
// output.
func totalTraffic(interfaces []jsonInterface) Traffic {
	var total Traffic
	for _, iface := range interfaces {
		if skipInterface(iface.Name) {
			continue
		}
		total.RxBytes += iface.Traffic.Total.Rx
		total.TxBytes += iface.Traffic.Total.Tx
	}
	return total
}

// findTraffic returns the traffic of the period with the given label, or zero
// when vnstat has no entry for it yet.
func findTraffic(periods []TrafficPeriod, label string) Traffic {
	for _, period := range periods {
		if period.Label == label {
			return Traffic{RxBytes: period.RxBytes, TxBytes: period.TxBytes}
		}
	}
	return Traffic{}
}

func skipInterface(name string) bool {
	switch name {
	case "lo":
		return true
	default:
		return strings.HasPrefix(name, "docker") ||
			strings.HasPrefix(name, "veth") ||
			strings.HasPrefix(name, "br-") ||
			strings.HasPrefix(name, "virbr")
	}
}
