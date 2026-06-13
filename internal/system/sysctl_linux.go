//go:build linux

package system

import (
	"os"
	"strings"
)

// ReadSysctl returns the current value of every catalog parameter by reading
// /proc/sys. Reading requires no privilege; a key that cannot be read (for
// example because its module is not loaded) is reported with Available=false and
// an empty value rather than failing the whole report.
func ReadSysctl() SysctlReport {
	catalog := SysctlCatalog()
	settings := make([]SysctlSetting, 0, len(catalog))
	for _, param := range catalog {
		setting := SysctlSetting{SysctlParam: param}
		if data, err := os.ReadFile(sysctlProcPath(param.Key)); err == nil {
			setting.Value = normalizeSysctlValue(string(data))
			setting.Available = true
		}
		settings = append(settings, setting)
	}
	return SysctlReport{Settings: settings, Writable: true}
}

// sysctlProcPath maps a dotted sysctl key to its /proc/sys path. None of the
// catalog keys contain a literal dot inside a component, so replacing every dot
// with a path separator is unambiguous.
func sysctlProcPath(key string) string {
	return "/proc/sys/" + strings.ReplaceAll(key, ".", "/")
}

// normalizeSysctlValue collapses the tab/space separated fields the kernel emits
// (e.g. "4096\t87380\t16777216") to single spaces and trims surrounding
// whitespace, matching the format ValidateSysctl produces.
func normalizeSysctlValue(raw string) string {
	return strings.Join(strings.Fields(raw), " ")
}
