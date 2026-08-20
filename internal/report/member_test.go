package report

import (
	"bytes"
	"strings"
	"testing"

	"github.com/assaio/assaio/internal/store"
)

// rawMember is a name no pseudonym can contain, so "is it in this document" is a
// substring search and nothing subtler.
const rawMember = "zzq-alice-example"

func memberRows() []store.UsageRow {
	return []store.UsageRow{{
		Day: "2026-07-01", Tool: "claude-code", Model: "claude-opus-4-5",
		Project: "acme-web", Member: rawMember, In: 100, Out: 200,
	}}
}

// TestDefaultExportsCarryNoRawMember is B182's regression: refusing `--by member` protected
// the ranking and not the data, so every default rendering of a central store's rows is
// searched for the name that was synced.
func TestDefaultExportsCarryNoRawMember(t *testing.T) {
	built := Build(memberRows(), table())
	renders := map[string]func(*bytes.Buffer) error{
		"table": func(b *bytes.Buffer) error { return RenderTable(b, built, "day") },
		"json":  func(b *bytes.Buffer) error { return RenderJSON(b, built) },
		"csv":   func(b *bytes.Buffer) error { return RenderCSV(b, built) },
	}
	for name, render := range renders {
		t.Run(name, func(t *testing.T) {
			var buf bytes.Buffer
			if err := render(&buf); err != nil {
				t.Fatalf("render %s: %v", name, err)
			}
			if strings.Contains(buf.String(), rawMember) {
				t.Fatalf("default %s export names the member raw:\n%s", name, buf.String())
			}
		})
	}
}

func TestBuildPseudonymizesMember(t *testing.T) {
	built := Build(memberRows(), table())
	if got := built[0].Member; got == rawMember || !strings.HasPrefix(got, "member-") {
		t.Fatalf("Member = %q, want a member- pseudonym", got)
	}
}

// TestBuildPseudonymIsStable is what lets a plugin or a script still group by member: the
// label has to be the same one on the next run.
func TestBuildPseudonymIsStable(t *testing.T) {
	first := Build(memberRows(), table())[0].Member
	second := Build(memberRows(), table())[0].Member
	if first != second {
		t.Fatalf("pseudonym not stable: %q then %q", first, second)
	}
}

func TestBuildIdentifiedKeepsRawMember(t *testing.T) {
	built := BuildIdentified(memberRows(), table())
	if built[0].Member != rawMember {
		t.Fatalf("Member = %q, want the raw %q", built[0].Member, rawMember)
	}
}

// TestBuildDistinctMembersStayDistinct guards the memoization: sharing one map across rows
// must not collapse two people into one label.
func TestBuildDistinctMembersStayDistinct(t *testing.T) {
	rows := memberRows()
	rows = append(rows, store.UsageRow{Day: "2026-07-02", Tool: "codex", Member: "zzq-bob-example"})
	built := Build(rows, table())
	if built[0].Member == built[1].Member {
		t.Fatalf("two members share one label %q", built[0].Member)
	}
}

func TestMemberDisclosure(t *testing.T) {
	local := Build([]store.UsageRow{{Day: "d", Tool: "codex"}}, table())
	if got := MemberDisclosure(local, MemberPseudonymous); got != "" {
		t.Fatalf("a store with no member disclosed %q", got)
	}
	team := Build(memberRows(), table())
	if got := MemberDisclosure(team, MemberPseudonymous); !strings.Contains(got, "pseudonymous") {
		t.Fatalf("pseudonymous disclosure = %q", got)
	}
	if got := MemberDisclosure(team, MemberIdentified); !strings.Contains(got, "raw") {
		t.Fatalf("identified disclosure = %q", got)
	}
}
