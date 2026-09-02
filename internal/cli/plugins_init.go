package cli

import (
	"embed"
	"fmt"
	"slices"
	"strings"
	"text/template"

	"github.com/spf13/cobra"

	"github.com/assaio/assaio/internal/config"
	"github.com/assaio/assaio/internal/plugin"
)

// The skeletons are checked in rather than built from strings so each one is a file a
// contributor can read, run and diff on its own. They carry a .tmpl extension because a
// package directory holding a second `package main` would not build.
//
//go:embed scaffold/*.tmpl
var scaffolds embed.FS

// scaffoldKind is what differs between the three protocols: where it is declared, which
// handshake version it must carry -- read from the runtime rather than written down twice --
// and the command that holds it to its contract.
type scaffoldKind struct {
	configKey string
	protocol  int
	verify    func(name string) string
}

var scaffoldKinds = map[string]scaffoldKind{
	"parser": {"plugins", plugin.ParserProtocolVersion, func(n string) string {
		return "assaio-agent plugins verify " + n
	}},
	"metric": {"metrics", plugin.MetricProtocolVersion, func(n string) string {
		return "assaio-agent metrics verify " + n
	}},
	"rule": {"rules", plugin.RuleProtocolVersion, func(string) string {
		return "assaio-agent check   # a rule is held to its contract inside the gate"
	}},
}

var scaffoldLangs = []string{"go", "python", "sh"}

func newPluginsInitCmd() *cobra.Command {
	var kind, lang, name string
	c := &cobra.Command{
		Use:   "init",
		Short: "Print a working out-of-tree plugin skeleton",
		Long: "Print a runnable skeleton for an out-of-tree plugin: a correct handshake, the " +
			"verbs its protocol defines, and one document assaio's boundary accepts.\n\n" +
			"The program goes to stdout and the next steps to stderr, so `assaio-agent plugins " +
			"init --kind metric --lang python > my-metric.py` writes only the program.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runPluginsInit(cmd, kind, lang, name)
		},
	}
	c.Flags().StringVar(&kind, "kind", "metric", "parser|metric|rule")
	c.Flags().StringVar(&lang, "lang", "python", "go|python|sh")
	c.Flags().StringVar(&name, "name", "", "plugin name, [a-z0-9-]+ (default my-<kind>)")
	return c
}

func runPluginsInit(cmd *cobra.Command, kind, lang, name string) error {
	spec, ok := scaffoldKinds[kind]
	if !ok {
		return fmt.Errorf("unknown --kind %q (want one of %s)", kind, strings.Join(kindNames(), ", "))
	}
	if !slices.Contains(scaffoldLangs, lang) {
		return fmt.Errorf("unknown --lang %q (want one of %s)", lang, strings.Join(scaffoldLangs, ", "))
	}
	if name == "" {
		name = "my-" + kind
	}
	// The name becomes the plugin:<name> namespace and must match the handshake, so it is held
	// to the same pattern config.yaml holds it to rather than to a second copy of the rule.
	if err := (config.PluginConfig{Name: name, Command: "placeholder"}).Validate(); err != nil {
		return fmt.Errorf("--name %q: %w", name, err)
	}

	tmpl, err := template.ParseFS(scaffolds, "scaffold/"+kind+"."+langExt(lang)+".tmpl")
	if err != nil {
		return err
	}
	if err := tmpl.Execute(cmd.OutOrStdout(), struct {
		Name     string
		Protocol int
	}{name, spec.protocol}); err != nil {
		return err
	}
	printScaffoldNextSteps(cmd, kind, lang, name, spec)
	return nil
}

// langExt maps the flag onto the template's middle extension; Go's skeleton is a main.go, so
// its templates are named for the file they become rather than for the flag.
func langExt(lang string) string {
	if lang == "python" {
		return "py"
	}
	return lang
}

// printScaffoldNextSteps goes to stderr so the program itself can be redirected into a file.
// Everything here is what a reader would otherwise have to find in three documents: where to
// declare it, what makes it executable, and the one command that tells them it conforms.
func printScaffoldNextSteps(cmd *cobra.Command, kind, lang, name string, spec scaffoldKind) {
	lw := &lineWriter{w: cmd.ErrOrStderr()}
	file := name + fileSuffix(lang)
	lw.printf("\n# 1. save the program above as %s", file)
	if lang == "go" {
		lw.printf(" and build it:\n#      go mod init %s && go build -o %s .\n", name, name)
	} else {
		lw.printf(" and make it executable:\n#      chmod +x %s\n", file)
	}
	lw.printf("# 2. declare it in ~/.config/assaio/config.yaml -- plugins are opt-in only,\n"+
		"#    never discovered from PATH:\n#\n#      %s:\n#        - name: %s\n#          command: /absolute/path/to/%s\n",
		spec.configKey, name, name)
	if kind == "metric" {
		lw.println("#\n#    A metric declares what it reads in its own `describe` output. The optional\n" +
			"#    `needs:` key here only *narrows* that -- it is your veto, not the declaration.")
	}
	lw.printf("# 3. check it against the boundary:\n#      %s\n", spec.verify(name))
	lw.printf("# 4. run the published conformance vectors in your own CI, without this binary:\n"+
		"#      docs/conformance/%s.json\n", vectorFile(kind))
}

func fileSuffix(lang string) string {
	if lang == "go" {
		return "/main.go"
	}
	return "." + langExt(lang)
}

// vectorFile names the catalogue that judges what this kind of plugin writes. A metric author
// needs both of theirs, and the declaration is the one they meet first.
func vectorFile(kind string) string {
	switch kind {
	case "parser":
		return "parser-record"
	case "rule":
		return "rule-alerts"
	}
	return "metric-declaration"
}

func kindNames() []string {
	out := make([]string, 0, len(scaffoldKinds))
	for k := range scaffoldKinds {
		out = append(out, k)
	}
	slices.Sort(out)
	return out
}
