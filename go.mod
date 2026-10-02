module github.com/assaio/assaio

go 1.26.0

// The language version stays at 1.25 so `go install` works on it. The toolchain floor is
// 1.27.1: below it, `go test -fuzz` reports `context deadline exceeded` as a failure when
// -fuzztime expires (go.dev/issue/75804), which makes a clean nightly fuzz run indistinguishable
// from a finding. `make vuln` holds the floor against standard-library advisories, and CI
// builds releases on it.
toolchain go1.27.1

require (
	github.com/jedib0t/go-pretty/v6 v6.8.3
	github.com/knadh/koanf/parsers/yaml v1.1.1
	github.com/knadh/koanf/providers/env v1.1.0
	github.com/knadh/koanf/providers/file v1.2.1
	github.com/knadh/koanf/providers/rawbytes v1.0.1
	github.com/knadh/koanf/providers/structs v1.0.1
	github.com/knadh/koanf/v2 v2.3.7
	github.com/spf13/cobra v1.10.2
	github.com/spf13/pflag v1.0.10
	github.com/yuin/goldmark v1.8.6
	golang.org/x/text v0.42.0
	modernc.org/sqlite v1.60.1
)

require (
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/fatih/structs v1.1.0 // indirect
	github.com/fsnotify/fsnotify v1.9.0 // indirect
	github.com/go-viper/mapstructure/v2 v2.4.0 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/knadh/koanf/maps v0.1.2 // indirect
	github.com/mattn/go-isatty v0.0.24 // indirect
	github.com/mattn/go-runewidth v0.0.16 // indirect
	github.com/mitchellh/copystructure v1.2.0 // indirect
	github.com/mitchellh/reflectwalk v1.0.2 // indirect
	github.com/ncruces/go-strftime v1.0.0 // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec // indirect
	github.com/rivo/uniseg v0.4.7 // indirect
	go.yaml.in/yaml/v3 v3.0.4 // indirect
	golang.org/x/sys v0.48.0 // indirect
	modernc.org/libc v1.77.1 // indirect
	modernc.org/mathutil v1.7.1 // indirect
	modernc.org/memory v1.12.1 // indirect
)
