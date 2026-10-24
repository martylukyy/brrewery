package vnstat

import (
	"context"
	_ "embed"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//go:embed testdata/days.json
var sampleDays []byte

//go:embed testdata/months.json
var sampleMonths []byte

//go:embed testdata/hours.json
var sampleHours []byte

type mockRunner struct {
	lookPathErr error
	outputs     map[string][]byte
	outputErr   error
	limits      map[string]string
}

func (m *mockRunner) LookPath(string) (string, error) {
	if m.lookPathErr != nil {
		return "", m.lookPathErr
	}
	return "/usr/bin/vnstat", nil
}

func (m *mockRunner) Output(_ context.Context, _ string, args ...string) ([]byte, error) {
	if m.outputErr != nil {
		return nil, m.outputErr
	}
	key := ""
	if len(args) >= 2 {
		key = args[1]
	}
	if len(args) >= 3 {
		if m.limits == nil {
			m.limits = map[string]string{}
		}
		m.limits[key] = args[2]
	}
	data, ok := m.outputs[key]
	if !ok {
		return nil, errors.New("unexpected args")
	}
	return data, nil
}

func TestCollector_Collect_notInstalled(t *testing.T) {
	t.Parallel()

	c := &Collector{runner: &mockRunner{lookPathErr: errors.New("missing")}}
	report, err := c.Collect(context.Background(), 14, 12)
	require.NoError(t, err)
	assert.False(t, report.Installed)
	assert.Contains(t, report.Message, "not installed")
}

func TestCollector_Collect_parsesHistory(t *testing.T) {
	t.Parallel()

	runner := &mockRunner{
		outputs: map[string][]byte{
			"d": sampleDays,
			"m": sampleMonths,
			"h": sampleHours,
		},
	}
	c := &Collector{runner: runner}

	report, err := c.Collect(context.Background(), 14, 12)
	require.NoError(t, err)
	require.True(t, report.Installed)
	require.Len(t, report.Days, 2)
	require.Len(t, report.Months, 2)

	assert.Equal(t, "14", runner.limits["d"])
	assert.Equal(t, "12", runner.limits["m"])
	assert.Equal(t, "2", runner.limits["h"])
	require.NotNil(t, report.Summary)

	assert.Equal(t, "2026-05-29", report.Days[0].Label)
	assert.Equal(t, uint64(1_000_000), report.Days[0].RxBytes)
	assert.Equal(t, uint64(500_000), report.Days[0].TxBytes)

	assert.Equal(t, "2026-05", report.Months[1].Label)
	assert.Equal(t, uint64(3_000_000), report.Months[1].RxBytes)
}

func TestParseReport_aggregatesInterfaces(t *testing.T) {
	t.Parallel()

	report, err := parseReport(sampleDays, sampleMonths, sampleHours, time.Now())
	require.NoError(t, err)
	assert.True(t, report.Installed)

	day := report.Days[len(report.Days)-1]
	assert.Equal(t, uint64(4_000_000), day.RxBytes)
	assert.Equal(t, uint64(2_000_000), day.TxBytes)
}

func TestParseReport_summary(t *testing.T) {
	t.Parallel()

	allTime := Traffic{RxBytes: 55_000_000, TxBytes: 22_000_000}

	tests := []struct {
		name string
		now  time.Time
		want Summary
	}{
		{
			name: "current periods",
			now:  time.Date(2026, 5, 30, 14, 25, 0, 0, time.UTC),
			want: Summary{
				ThisHour:  Traffic{RxBytes: 210_000, TxBytes: 105_000},
				LastHour:  Traffic{RxBytes: 100_000, TxBytes: 50_000},
				ThisDay:   Traffic{RxBytes: 4_000_000, TxBytes: 2_000_000},
				ThisMonth: Traffic{RxBytes: 3_000_000, TxBytes: 1_500_000},
				AllTime:   allTime,
			},
		},
		{
			name: "no entries for current periods yet",
			now:  time.Date(2026, 6, 1, 0, 10, 0, 0, time.UTC),
			want: Summary{AllTime: allTime},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			report, err := parseReport(sampleDays, sampleMonths, sampleHours, tt.now)
			require.NoError(t, err)
			require.NotNil(t, report.Summary)
			assert.Equal(t, tt.want, *report.Summary)
		})
	}
}
