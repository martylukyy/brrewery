package system

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// SysctlConfPath is the drop-in file brrewery owns for persisted kernel tuning.
// The high numeric prefix means systemd-sysctl applies it last, so brrewery's
// values win over the distribution defaults in /etc/sysctl.conf and earlier
// drop-ins.
const SysctlConfPath = "/etc/sysctl.d/99-brrewery.conf"

const sysctlConfHeader = "# Managed by brrewery. Edits are overwritten when sysctl settings are applied from the web UI.\n"

// Sysctl parameter kinds drive both server-side validation and the form control
// the UI renders for each parameter.
const (
	// SysctlInteger is a single signed integer, optionally bounded by Min/Max.
	SysctlInteger = "integer"
	// SysctlIntegerList is a fixed number of whitespace-separated integers, such
	// as net.ipv4.tcp_rmem's "min default max" triple.
	SysctlIntegerList = "integer_list"
	// SysctlEnum is a value that must be one of Choices.
	SysctlEnum = "enum"
)

// SysctlParam describes one tunable kernel parameter the UI can present. It is
// platform-neutral metadata; live values are read separately by ReadSysctl.
type SysctlParam struct {
	Key         string   `json:"key"`
	Label       string   `json:"label"`
	Description string   `json:"description"`
	Group       string   `json:"group"`
	Kind        string   `json:"kind"`
	Recommended string   `json:"recommended"`
	Unit        string   `json:"unit,omitempty"`
	Min         *int64   `json:"min,omitempty"`
	Max         *int64   `json:"max,omitempty"`
	Fields      int      `json:"fields,omitempty"`
	Choices     []string `json:"choices,omitempty"`
}

// SysctlSetting is a catalog parameter paired with its current value on the host.
type SysctlSetting struct {
	SysctlParam
	// Value is the current live value, normalized to single-space separators.
	// Empty when the key could not be read on this host.
	Value string `json:"value"`
	// Available reports whether the key exists on this kernel (e.g. the
	// congestion-control algorithm's module may not be loaded).
	Available bool `json:"available"`
}

// SysctlReport is the full tuning snapshot returned to the UI.
type SysctlReport struct {
	Settings []SysctlSetting `json:"settings"`
	// Writable reports whether this platform supports applying changes. It is
	// false on non-Linux builds so the UI can disable the apply action.
	Writable bool `json:"writable"`
}

// maxBuffer bounds the byte-sized network/buffer tunables. 2 GiB is comfortably
// above any sane value while keeping the field within a 32-bit kernel sysctl.
const maxBuffer = int64(1) << 31

func i64(v int64) *int64 { return &v }

// SysctlCatalog returns the curated set of kernel parameters brrewery exposes
// for tuning. It is intentionally a small, well-understood allow-list oriented
// at a high-throughput seedbox/media host: nothing outside this list can be
// written (see ValidateSysctl), so the blast radius stays bounded.
func SysctlCatalog() []SysctlParam {
	catalog := []SysctlParam{
		// Network — core buffers and queueing -------------------------------
		{
			Key: "net.core.rmem_max", Label: "Max receive buffer", Group: "Network",
			Description: "Largest receive socket buffer the kernel will allocate.",
			Kind:        SysctlInteger, Unit: "bytes", Recommended: "16777216",
			Min: i64(0), Max: i64(maxBuffer),
		},
		{
			Key: "net.core.wmem_max", Label: "Max send buffer", Group: "Network",
			Description: "Largest send socket buffer the kernel will allocate.",
			Kind:        SysctlInteger, Unit: "bytes", Recommended: "16777216",
			Min: i64(0), Max: i64(maxBuffer),
		},
		{
			Key: "net.core.rmem_default", Label: "Default receive buffer", Group: "Network",
			Description: "Default receive socket buffer size for new sockets.",
			Kind:        SysctlInteger, Unit: "bytes", Recommended: "16777216",
			Min: i64(0), Max: i64(maxBuffer),
		},
		{
			Key: "net.core.wmem_default", Label: "Default send buffer", Group: "Network",
			Description: "Default send socket buffer size for new sockets.",
			Kind:        SysctlInteger, Unit: "bytes", Recommended: "16777216",
			Min: i64(0), Max: i64(maxBuffer),
		},
		{
			Key: "net.core.optmem_max", Label: "Max ancillary buffer", Group: "Network",
			Description: "Maximum ancillary (control) message memory per socket.",
			Kind:        SysctlInteger, Unit: "bytes", Recommended: "16777216",
			Min: i64(0), Max: i64(maxBuffer),
		},
		{
			Key: "net.core.netdev_max_backlog", Label: "Device backlog", Group: "Network",
			Description: "Packets queued for the protocol stack when the NIC outpaces the CPU.",
			Kind:        SysctlInteger, Unit: "packets", Recommended: "3240000",
			Min: i64(0), Max: i64(maxBuffer),
		},
		{
			Key: "net.core.netdev_budget", Label: "Device poll budget", Group: "Network",
			Description: "Packets drained per NAPI poll cycle across all interfaces.",
			Kind:        SysctlInteger, Unit: "packets", Recommended: "200000",
			Min: i64(1), Max: i64(maxBuffer),
		},
		{
			Key: "net.core.somaxconn", Label: "Listen backlog", Group: "Network",
			Description: "Maximum length of the accept queue for listening sockets.",
			Kind:        SysctlInteger, Unit: "connections", Recommended: "50000",
			Min: i64(128), Max: i64(maxBuffer),
		},
		{
			Key: "net.core.default_qdisc", Label: "Default queue discipline", Group: "Network",
			Description: "Default packet scheduler for new interfaces. fq pairs with BBR.",
			Kind:        SysctlEnum, Recommended: "fq",
			Choices: []string{"fq", "fq_codel", "pfifo_fast", "cake"},
		},

		// Network — TCP buffers and congestion control ----------------------
		{
			Key: "net.ipv4.tcp_congestion_control", Label: "TCP congestion control", Group: "Network",
			Description: "Congestion-control algorithm. BBR sustains throughput on lossy/long paths.",
			Kind:        SysctlEnum, Recommended: "bbr",
			Choices: []string{"bbr", "cubic", "reno", "htcp"},
		},
		{
			Key: "net.ipv4.tcp_rmem", Label: "TCP receive memory", Group: "Network",
			Description: "min / default / max receive buffer (bytes) the TCP stack autotunes between.",
			Kind:        SysctlIntegerList, Fields: 3, Unit: "bytes", Recommended: "4096 524000 67110000",
			Min: i64(0), Max: i64(maxBuffer),
		},
		{
			Key: "net.ipv4.tcp_wmem", Label: "TCP send memory", Group: "Network",
			Description: "min / default / max send buffer (bytes) the TCP stack autotunes between.",
			Kind:        SysctlIntegerList, Fields: 3, Unit: "bytes", Recommended: "4096 524000 67110000",
			Min: i64(0), Max: i64(maxBuffer),
		},
		{
			Key: "net.ipv4.tcp_mem", Label: "TCP memory pages", Group: "Network",
			Description: "low / pressure / high TCP memory thresholds, in pages.",
			Kind:        SysctlIntegerList, Fields: 3, Unit: "pages", Recommended: "3086631 4115510 6173262",
			Min: i64(0), Max: i64(maxBuffer),
		},
		{
			Key: "net.ipv4.tcp_adv_win_scale", Label: "Window scaling bias", Group: "Network",
			Description: "Split between application buffer and TCP window in the receive buffer.",
			Kind:        SysctlInteger, Recommended: "2",
			Min: i64(-31), Max: i64(31),
		},

		// Network — backlogs and connection limits --------------------------
		{
			Key: "net.ipv4.tcp_max_syn_backlog", Label: "SYN backlog", Group: "Network",
			Description: "Half-open connections queued while awaiting the final ACK of the handshake.",
			Kind:        SysctlInteger, Unit: "connections", Recommended: "8192",
			Min: i64(0), Max: i64(maxBuffer),
		},
		{
			Key: "net.ipv4.tcp_max_tw_buckets", Label: "TIME-WAIT buckets", Group: "Network",
			Description: "Maximum sockets held in TIME-WAIT before the oldest are reset.",
			Kind:        SysctlInteger, Unit: "sockets", Recommended: "262144",
			Min: i64(0), Max: i64(maxBuffer),
		},
		{
			Key: "net.ipv4.tcp_max_orphans", Label: "Max orphaned sockets", Group: "Network",
			Description: "Maximum TCP sockets not attached to a file handle.",
			Kind:        SysctlInteger, Unit: "sockets", Recommended: "262144",
			Min: i64(0), Max: i64(maxBuffer),
		},

		// Network — TCP behaviour toggles -----------------------------------
		{
			Key: "net.ipv4.tcp_slow_start_after_idle", Label: "Slow start after idle", Group: "Network",
			Description: "Reset the congestion window after an idle period. 0 keeps long transfers fast.",
			Kind:        SysctlInteger, Recommended: "0",
			Min: i64(0), Max: i64(1),
		},
		{
			Key: "net.ipv4.tcp_no_metrics_save", Label: "Discard route metrics", Group: "Network",
			Description: "Skip caching TCP metrics per route between connections.",
			Kind:        SysctlInteger, Recommended: "0",
			Min: i64(0), Max: i64(1),
		},
		{
			Key: "net.ipv4.tcp_abort_on_overflow", Label: "Abort on overflow", Group: "Network",
			Description: "Reset connections when the accept queue overflows. 0 keeps them.",
			Kind:        SysctlInteger, Recommended: "0",
			Min: i64(0), Max: i64(1),
		},
		{
			Key: "net.ipv4.tcp_window_scaling", Label: "Window scaling", Group: "Network",
			Description: "Enable RFC 1323 window scaling for buffers larger than 64 KiB.",
			Kind:        SysctlInteger, Recommended: "1",
			Min: i64(0), Max: i64(1),
		},
		{
			Key: "net.ipv4.tcp_tw_reuse", Label: "Reuse TIME-WAIT", Group: "Network",
			Description: "Reuse TIME-WAIT sockets for new outbound connections.",
			Kind:        SysctlInteger, Recommended: "1",
			Min: i64(0), Max: i64(2),
		},
		{
			Key: "net.ipv4.tcp_rfc1337", Label: "RFC 1337 protection", Group: "Network",
			Description: "Drop RST for sockets in TIME-WAIT to avoid the RFC 1337 hazard.",
			Kind:        SysctlInteger, Recommended: "1",
			Min: i64(0), Max: i64(1),
		},
		{
			Key: "net.ipv4.tcp_sack", Label: "Selective ACK", Group: "Network",
			Description: "Enable selective acknowledgements for faster loss recovery.",
			Kind:        SysctlInteger, Recommended: "1",
			Min: i64(0), Max: i64(1),
		},
		{
			Key: "net.ipv4.tcp_fack", Label: "Forward ACK", Group: "Network",
			Description: "Forward-acknowledgement congestion control (legacy; ignored on newer kernels).",
			Kind:        SysctlInteger, Recommended: "1",
			Min: i64(0), Max: i64(1),
		},
		{
			Key: "net.ipv4.tcp_workaround_signed_windows", Label: "Signed window workaround", Group: "Network",
			Description: "Work around peers that treat the TCP window as a signed value.",
			Kind:        SysctlInteger, Recommended: "1",
			Min: i64(0), Max: i64(1),
		},
		{
			Key: "net.ipv4.tcp_syncookies", Label: "SYN cookies", Group: "Network",
			Description: "Use SYN cookies when the SYN backlog overflows (SYN-flood defense).",
			Kind:        SysctlInteger, Recommended: "1",
			Min: i64(0), Max: i64(2),
		},
		{
			Key: "net.ipv4.tcp_timestamps", Label: "TCP timestamps", Group: "Network",
			Description: "RFC 1323 timestamps. 0 trims per-packet overhead.",
			Kind:        SysctlInteger, Recommended: "0",
			Min: i64(0), Max: i64(1),
		},
		{
			Key: "net.ipv4.ip_no_pmtu_disc", Label: "Disable PMTU discovery", Group: "Network",
			Description: "0 keeps path-MTU discovery enabled.",
			Kind:        SysctlInteger, Recommended: "0",
			Min: i64(0), Max: i64(2),
		},
		{
			Key: "net.ipv4.tcp_mtu_probing", Label: "MTU probing", Group: "Network",
			Description: "Recover throughput across paths that black-hole large packets.",
			Kind:        SysctlInteger, Recommended: "1",
			Min: i64(0), Max: i64(2),
		},
		{
			Key: "net.ipv4.tcp_fastopen", Label: "TCP Fast Open", Group: "Network",
			Description: "Send data in the SYN to save a round trip. 3 enables client and server.",
			Kind:        SysctlInteger, Recommended: "3",
			Min: i64(0), Max: i64(3),
		},

		// Network — retries and timeouts ------------------------------------
		{
			Key: "net.ipv4.tcp_syn_retries", Label: "SYN retries", Group: "Network",
			Description: "SYN retransmissions before giving up on an outbound connection.",
			Kind:        SysctlInteger, Recommended: "2",
			Min: i64(1), Max: i64(255),
		},
		{
			Key: "net.ipv4.tcp_synack_retries", Label: "SYN-ACK retries", Group: "Network",
			Description: "SYN-ACK retransmissions for inbound connections.",
			Kind:        SysctlInteger, Recommended: "2",
			Min: i64(0), Max: i64(255),
		},
		{
			Key: "net.ipv4.tcp_orphan_retries", Label: "Orphan retries", Group: "Network",
			Description: "Retries on a closing socket before it is abandoned.",
			Kind:        SysctlInteger, Recommended: "2",
			Min: i64(0), Max: i64(255),
		},
		{
			Key: "net.ipv4.tcp_retries2", Label: "Data retries", Group: "Network",
			Description: "Retransmissions of live data before the connection is dropped.",
			Kind:        SysctlInteger, Recommended: "8",
			Min: i64(1), Max: i64(255),
		},
		{
			Key: "net.ipv4.tcp_fin_timeout", Label: "FIN timeout", Group: "Network",
			Description: "Seconds a socket lingers in FIN-WAIT-2.",
			Kind:        SysctlInteger, Unit: "seconds", Recommended: "60",
			Min: i64(1), Max: i64(600),
		},
		{
			Key: "net.ipv4.tcp_keepalive_time", Label: "Keepalive time", Group: "Network",
			Description: "Idle seconds before the first TCP keepalive probe.",
			Kind:        SysctlInteger, Unit: "seconds", Recommended: "7200",
			Min: i64(1), Max: i64(86400),
		},
		{
			Key: "net.ipv4.tcp_keepalive_intvl", Label: "Keepalive interval", Group: "Network",
			Description: "Seconds between successive keepalive probes.",
			Kind:        SysctlInteger, Unit: "seconds", Recommended: "75",
			Min: i64(1), Max: i64(32767),
		},
		{
			Key: "net.ipv4.tcp_keepalive_probes", Label: "Keepalive probes", Group: "Network",
			Description: "Unacknowledged probes before the connection is dropped.",
			Kind:        SysctlInteger, Recommended: "9",
			Min: i64(1), Max: i64(127),
		},
		{
			Key: "net.ipv4.ip_local_port_range", Label: "Ephemeral port range", Group: "Network",
			Description: "Low / high bound of the local port range for outbound connections.",
			Kind:        SysctlIntegerList, Fields: 2, Recommended: "1024 65535",
			Min: i64(1), Max: i64(65535),
		},

		// Memory -------------------------------------------------------------
		{
			Key: "vm.swappiness", Label: "Swappiness", Group: "Memory",
			Description: "How aggressively the kernel swaps anonymous pages (0-100). Low keeps processes in RAM.",
			Kind:        SysctlInteger, Recommended: "1",
			Min: i64(0), Max: i64(100),
		},
		{
			Key: "vm.dirty_ratio", Label: "Dirty ratio", Group: "Memory",
			Description: "Percent of RAM of dirty pages before writers block on writeback.",
			Kind:        SysctlInteger, Unit: "percent", Recommended: "30",
			Min: i64(0), Max: i64(100),
		},
		{
			Key: "vm.dirty_background_ratio", Label: "Dirty background ratio", Group: "Memory",
			Description: "Percent of RAM of dirty pages at which background writeback starts.",
			Kind:        SysctlInteger, Unit: "percent", Recommended: "10",
			Min: i64(0), Max: i64(100),
		},
		{
			Key: "vm.min_free_kbytes", Label: "Min free memory", Group: "Memory",
			Description: "Memory the kernel keeps free as a reserve for atomic allocations.",
			Kind:        SysctlInteger, Unit: "KB", Recommended: "512000",
			Min: i64(1024), Max: i64(maxBuffer),
		},
		{
			Key: "vm.zone_reclaim_mode", Label: "Zone reclaim mode", Group: "Memory",
			Description: "NUMA zone reclaim behaviour bitmask. 0 disables local-zone reclaim.",
			Kind:        SysctlInteger, Recommended: "1",
			Min: i64(0), Max: i64(7),
		},
		{
			Key: "vm.vfs_cache_pressure", Label: "VFS cache pressure", Group: "Memory",
			Description: "Tendency to reclaim inode/dentry caches. Below 100 keeps directory metadata cached longer for large libraries.",
			Kind:        SysctlInteger, Recommended: "100",
			Min: i64(1), Max: i64(1000),
		},

		// Filesystem ---------------------------------------------------------
		{
			Key: "fs.file-max", Label: "Max open files", Group: "Filesystem",
			Description: "System-wide ceiling on open file descriptors.",
			Kind:        SysctlInteger, Unit: "files", Recommended: "2000000",
			// fs.file-max is a kernel long; distros and containers commonly default
			// it to LONG_MAX (9223372036854775807), far above maxBuffer. Leave the
			// upper bound open so the live value validates — int64 parsing still
			// caps it at MaxInt64.
			Min: i64(1024),
		},
		{
			Key: "fs.inotify.max_user_watches", Label: "Inotify watches", Group: "Filesystem",
			Description: "Per-user inotify watch limit. The *arr apps watch large media trees and exhaust the default quickly.",
			Kind:        SysctlInteger, Unit: "watches", Recommended: "524288",
			Min: i64(8192), Max: i64(maxBuffer),
		},

		// Kernel -------------------------------------------------------------
		{
			Key: "kernel.pid_max", Label: "Max PID", Group: "Kernel",
			Description: "Highest process ID before the counter wraps.",
			Kind:        SysctlInteger, Recommended: "4194303",
			Min: i64(1024), Max: i64(maxBuffer),
		},
		{
			Key: "kernel.sched_migration_cost_ns", Label: "Scheduler migration cost", Group: "Kernel",
			Description: "Nanoseconds a task stays cache-hot before the scheduler may migrate it.",
			Kind:        SysctlInteger, Unit: "ns", Recommended: "5000000",
			Min: i64(0), Max: i64(maxBuffer),
		},
		{
			Key: "kernel.sched_autogroup_enabled", Label: "Scheduler autogroups", Group: "Kernel",
			Description: "Group tasks by session for desktop fairness. 0 favours server throughput.",
			Kind:        SysctlInteger, Recommended: "0",
			Min: i64(0), Max: i64(1),
		},
	}

	// Present parameters sorted by group, then by key, so the UI (which renders
	// in catalog order) groups them alphabetically with keys ordered within each.
	sort.SliceStable(catalog, func(i, j int) bool {
		if catalog[i].Group != catalog[j].Group {
			return catalog[i].Group < catalog[j].Group
		}
		return catalog[i].Key < catalog[j].Key
	})
	return catalog
}

// ValidateSysctl checks caller-supplied key/value pairs against the curated
// catalog and returns a sanitized copy that is safe to write to the sysctl conf
// file. It is the trust boundary between the web UI and the kernel: unknown
// keys, non-numeric or out-of-range values, malformed lists, and values outside
// an enum's allowed set are all rejected, so nothing outside the catalog (and no
// shell/conf metacharacters) can ever reach disk.
func ValidateSysctl(values map[string]string) (map[string]string, error) {
	if len(values) == 0 {
		return nil, fmt.Errorf("no sysctl values provided")
	}

	catalog := SysctlCatalog()
	byKey := make(map[string]SysctlParam, len(catalog))
	for _, p := range catalog {
		byKey[p.Key] = p
	}

	out := make(map[string]string, len(values))
	for key, raw := range values {
		param, ok := byKey[key]
		if !ok {
			return nil, fmt.Errorf("unknown sysctl parameter %q", key)
		}
		clean, err := validateSysctlValue(param, raw)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", key, err)
		}
		out[key] = clean
	}
	return out, nil
}

func validateSysctlValue(p SysctlParam, raw string) (string, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", fmt.Errorf("value is required")
	}

	switch p.Kind {
	case SysctlEnum:
		for _, choice := range p.Choices {
			if value == choice {
				return value, nil
			}
		}
		return "", fmt.Errorf("must be one of: %s", strings.Join(p.Choices, ", "))

	case SysctlIntegerList:
		fields := strings.Fields(value)
		want := p.Fields
		if want == 0 {
			want = len(fields)
		}
		if len(fields) != want {
			return "", fmt.Errorf("expected %d space-separated integers", want)
		}
		for _, field := range fields {
			if err := checkSysctlInt(p, field); err != nil {
				return "", err
			}
		}
		return strings.Join(fields, " "), nil

	case SysctlInteger, "":
		if err := checkSysctlInt(p, value); err != nil {
			return "", err
		}
		return value, nil

	default:
		return "", fmt.Errorf("unsupported parameter kind %q", p.Kind)
	}
}

func checkSysctlInt(p SysctlParam, s string) error {
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return fmt.Errorf("%q is not a valid integer", s)
	}
	if p.Min != nil && n < *p.Min {
		return fmt.Errorf("must be at least %d", *p.Min)
	}
	if p.Max != nil && n > *p.Max {
		return fmt.Errorf("must be at most %d", *p.Max)
	}
	return nil
}

// SysctlConfContent renders the persisted drop-in file for the given values. Keys
// are sorted so the output is deterministic (stable file, idempotent Ansible
// runs). Values are assumed to have passed ValidateSysctl.
func SysctlConfContent(values map[string]string) string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	var b strings.Builder
	b.WriteString(sysctlConfHeader)
	for _, key := range keys {
		fmt.Fprintf(&b, "%s = %s\n", key, values[key])
	}
	return b.String()
}
