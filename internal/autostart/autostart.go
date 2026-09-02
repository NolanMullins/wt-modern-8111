package autostart

const applicationName = "WT Modern 8111"

func command(executable string) string {
	return `"` + executable + `" -open=false`
}
