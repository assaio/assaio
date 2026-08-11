package docs

import (
	"strconv"
	"strings"
)

func section(b *strings.Builder, id, title, note string) {
	b.WriteString(`<section id="` + id + `"><h2>` + esc(title) + `</h2><p class="note">` + note + `</p>`)
}

func writeSignals(b *strings.Builder, signals []Signal) {
	section(b, "signals", "Signals", `What assaio can report, and what each figure counts.
<strong>What a zero means</strong> is part of the declaration, because "no rework happened" and
"this source never recorded rework" are different facts. <em>Sources</em> names the shipped
parsers that can answer it; an empty column is a signal nothing in the tree produces yet.`)
	t := newTable(b, "signals", "Id", "Unit", "Status", "Grains", "A zero means", "What it is", "Sources")
	for i := range signals {
		s := &signals[i]
		t.row("signal."+s.ID,
			cell("m", s.ID), cell("", s.Unit), cell("", s.Status),
			cell("dim", strings.Join(s.Grains, " ")),
			cell("dim", s.ZeroMeans), cell("", s.Describe),
			cell("dim", join(s.Sources)))
	}
	t.close()
	b.WriteString(`</section>`)
}

func writeSources(b *strings.Builder, sources []Source) {
	section(b, "sources", "Sources", `"Supports Gemini CLI" is misleading when it means tokens but
no edits, so every source publishes its capability per signal rather than per checkmark. The three
columns are the tier table's one-bit summaries; the gaps are what the source does not carry, in a
reader's terms. Out-of-tree exec parsers are absent by design — their depth is whatever their
author implemented, so it is read from the records they emit.`)
	t := newTable(b, "sources", "Tool", "Tier", "Tokens", "Activity", "Attribution", "Signals", "Gaps")
	for _, s := range sources {
		t.row("source."+s.Tool,
			cell("m", s.Tool), cell("", s.Tier),
			cell("dim", yesNo(s.Tokens)), cell("dim", yesNo(s.Activity)), cell("dim", yesNo(s.Attribution)),
			cell("dim", strconv.Itoa(len(s.Answers))),
			rawCell("", bullets(s.Gaps)))
	}
	t.close()
	b.WriteString(`</section>`)
}

func writeValidators(b *strings.Builder, validators []Validator) {
	section(b, "validators", "Validators", `The metric reads <code>assaio-agent analyze</code> runs,
each one file in the codebase and each printing a directional verdict with its caveat. A
<em>window</em>-scoped read cannot honestly be narrowed to one project — a flat plan price or
attribution pooled across every project is a fact about the whole window — so the dashboard's
project drill-down skips those rather than recomputing them over a slice.`)
	t := newTable(b, "validators", "Name", "Title", "Scope", "What it reads")
	for _, v := range validators {
		t.row("validator."+v.Name,
			cell("m", v.Name), cell("", v.Title), cell("dim", v.Scope), cell("", v.Describe))
	}
	t.close()
	b.WriteString(`</section>`)
}

func writeCommands(b *strings.Builder, commands []Command, global []Flag) {
	section(b, "commands", "Commands", `Every invocation, with the flags it declares. The global
flags below are accepted by all of them; a flag listed under a command is that command's own.`)
	if len(global) > 0 {
		b.WriteString(`<div class="cmd"><h3>global</h3><p>Accepted by every command.</p>`)
		flagTable(b, global)
		b.WriteString(`</div>`)
	}
	b.WriteString(`<div data-covers="commands">`)
	for _, c := range commands {
		b.WriteString(`<div class="cmd" data-claim="command.` + esc(c.Path) + `">`)
		b.WriteString(`<h3>` + esc(c.Path) + `</h3><p>` + esc(c.Short) + `</p>`)
		flagTable(b, c.Flags)
		b.WriteString(`</div>`)
	}
	b.WriteString(`</div></section>`)
}

func flagTable(b *strings.Builder, fs []Flag) {
	if len(fs) == 0 {
		return
	}
	t := newTable(b, "", "Flag", "Type", "Default", "What it does")
	for i := range fs {
		f := &fs[i]
		t.row("", cell("m", flagName(f)), cell("dim", f.Type),
			cell("dim", orDash(f.Default)), cell("", f.Usage))
	}
	t.close()
}

func writeConfig(b *strings.Builder, keys []ConfigKey) {
	section(b, "config", "Configuration", `Defaults, then the YAML file, then
<code>ASSAIO_</code>-prefixed environment variables — in that order. A blank environment column
means the key has <em>no</em> environment form: one variable carries one string, and a list of
entries cannot be built from that. A blank default means the zero value applies, which for a list
means "none configured", never "none allowed" — except where the default reads
<em>unset</em>, which is not the same as the zero value, and what it means then is stated with the
key rather than implied by its type.`)
	t := newTable(b, "config", "Key", "Environment variable", "Type", "Default")
	for _, k := range keys {
		key := k.Key
		if k.List {
			key += "[]"
		}
		t.row("config."+k.Key, cell("m", key), cell("m", orDash(k.Env)),
			cell("dim", k.Type), cell("dim", defaultText(k)))
	}
	t.close()
	b.WriteString(`</section>`)
}

// defaultText keeps "nothing was set" apart from "the zero value applies". They are the same
// blank in Go and different facts to a reader: `labels.defaults` is a nil *bool whose absence
// keeps the built-in rules, so publishing it as a blank boolean states the opposite.
func defaultText(k ConfigKey) string {
	if k.Optional && k.Default == "" {
		return "unset"
	}
	return orDash(k.Default)
}

func writeWire(b *strings.Builder, in []Field, w MetricWire) {
	section(b, "metric-plugin", "The metric contract, both ways of writing one", `A metric is
either a Go file compiled in or an executable speaking JSON on stdin and stdout (ADR 0004). Both
lists are generated from the types themselves, side by side, because an out-of-tree author who
cannot see a field a built-in validator reads is working against a weaker contract than the one
assaio publishes.
<strong>These tables describe the document's shape, not its obligations</strong> — which fields
the boundary rejects a result for omitting, and which it overwrites on arrival, are enforced by
code rather than by a struct tag, so they are stated in
<a href="https://github.com/assaio/assaio/blob/main/docs/extending/metric-plugin.md">the protocol
page</a> along with what each field means. <code>[]</code> addresses an array element and
<code>{}</code> a map value under a key the document does not fix.`)
	b.WriteString(`<h3 class="sub">In-tree — what <code>analyze.Input</code> hands a validator</h3>`)
	t := newTable(b, "", "Field", "Go type")
	for _, f := range in {
		t.row("", cell("m", f.Name), cell("dim", f.Type))
	}
	t.close()
	b.WriteString(`<h3 class="sub">Out-of-tree — what a plugin reads on <code>stdin</code></h3>`)
	wireTable(b, w.Input)
	b.WriteString(`<h3 class="sub">Out-of-tree — what a plugin may return on <code>stdout</code></h3>`)
	wireTable(b, w.Result)
	b.WriteString(`</section>`)
}

func wireTable(b *strings.Builder, fields []WireField) {
	t := newTable(b, "", "Field", "Type", "Absent when")
	for _, f := range fields {
		typ := f.Type
		if f.Nullable {
			typ += " or null"
		}
		absent := "never"
		if f.OmitEmpty {
			absent = "empty"
		}
		t.row("", cell("m", f.Path), cell("dim", typ), cell("dim", absent))
	}
	t.close()
}

func flagName(f *Flag) string {
	if f.Shorthand != "" {
		return "--" + f.Name + ", -" + f.Shorthand
	}
	return "--" + f.Name
}

func join(s []string) string {
	if len(s) == 0 {
		return "—"
	}
	return strings.Join(s, " ")
}

func orDash(s string) string {
	if s == "" {
		return "—"
	}
	return s
}
