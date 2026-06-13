//go:build !linux

package system

// ReadSysctl returns the catalog with no live values on non-Linux platforms and
// reports the host as non-writable, so the UI can still show what brrewery would
// tune while disabling the apply action.
func ReadSysctl() SysctlReport {
	catalog := SysctlCatalog()
	settings := make([]SysctlSetting, 0, len(catalog))
	for _, param := range catalog {
		settings = append(settings, SysctlSetting{SysctlParam: param})
	}
	return SysctlReport{Settings: settings, Writable: false}
}
