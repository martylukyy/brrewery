package vnstat

type Report struct {
	Installed bool            `json:"installed"`
	Message   string          `json:"message,omitempty"`
	Version   string          `json:"version,omitempty"`
	Days      []TrafficPeriod `json:"days,omitempty"`
	Months    []TrafficPeriod `json:"months,omitempty"`
	Summary   *Summary        `json:"summary,omitempty"`
}

type TrafficPeriod struct {
	Label   string `json:"label"`
	RxBytes uint64 `json:"rx_bytes"`
	TxBytes uint64 `json:"tx_bytes"`
}

// Summary is the traffic of the current periods, resolved against the
// server's local clock (the same one vnstat dates its entries with).
type Summary struct {
	ThisHour  Traffic `json:"this_hour"`
	LastHour  Traffic `json:"last_hour"`
	ThisDay   Traffic `json:"this_day"`
	ThisMonth Traffic `json:"this_month"`
	AllTime   Traffic `json:"all_time"`
}

type Traffic struct {
	RxBytes uint64 `json:"rx_bytes"`
	TxBytes uint64 `json:"tx_bytes"`
}
