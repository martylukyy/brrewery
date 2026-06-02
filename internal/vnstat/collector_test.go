package vnstat

import (
	"context"
	_ "embed"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//go:embed testdata/days.json
var sampleDays []byte

//go:embed testdata/months.json
var sampleMonths []byte

type mockRunner struct {
	lookPathErr error
	outputs     map[string][]byte
	outputErr   error
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
	data, ok := m.outputs[key]
	if !ok {
		return nil, errors.New("unexpected args")
	}
	return data, nil
}

func TestCollector_Collect_notInstalled(t *testing.T) {
	t.Parallel()

	c := &Collector{runner: &mockRunner{lookPathErr: errors.New("missing")}}
	report, err := c.Collect(context.Background())
	require.NoError(t, err)
	assert.False(t, report.Installed)
	assert.Contains(t, report.Message, "not installed")
}

func TestCollector_Collect_parsesHistory(t *testing.T) {
	t.Parallel()

	c := &Collector{runner: &mockRunner{
		outputs: map[string][]byte{
			"d": sampleDays,
			"m": sampleMonths,
		},
	}}

	report, err := c.Collect(context.Background())
	require.NoError(t, err)
	require.True(t, report.Installed)
	require.Len(t, report.Days, 2)
	require.Len(t, report.Months, 2)

	assert.Equal(t, "2026-05-29", report.Days[0].Label)
	assert.Equal(t, uint64(1_000_000), report.Days[0].RxBytes)
	assert.Equal(t, uint64(500_000), report.Days[0].TxBytes)

	assert.Equal(t, "2026-05", report.Months[1].Label)
	assert.Equal(t, uint64(3_000_000), report.Months[1].RxBytes)
}

func TestParseReport_aggregatesInterfaces(t *testing.T) {
	t.Parallel()

	report, err := parseReport(sampleDays, sampleMonths)
	require.NoError(t, err)
	assert.True(t, report.Installed)

	day := report.Days[len(report.Days)-1]
	assert.Equal(t, uint64(4_000_000), day.RxBytes)
	assert.Equal(t, uint64(2_000_000), day.TxBytes)
}
