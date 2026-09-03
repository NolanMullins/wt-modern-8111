//go:build windows

package playerteam

import "golang.org/x/sys/windows/registry"

func platformSteamRoots() []string {
	locations := []struct {
		key   registry.Key
		path  string
		flags uint32
	}{
		{registry.CURRENT_USER, `Software\Valve\Steam`, registry.QUERY_VALUE},
		{
			registry.LOCAL_MACHINE,
			`Software\Valve\Steam`,
			registry.QUERY_VALUE | registry.WOW64_32KEY,
		},
	}
	var roots []string
	for _, location := range locations {
		key, err := registry.OpenKey(location.key, location.path, location.flags)
		if err != nil {
			continue
		}
		value, _, err := key.GetStringValue("SteamPath")
		_ = key.Close()
		if err == nil && value != "" {
			roots = append(roots, value)
		}
	}
	return roots
}
