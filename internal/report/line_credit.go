package report

import (
	"github.com/assaio/assaio/internal/parser"
	"github.com/assaio/assaio/internal/pricing"
	"github.com/assaio/assaio/internal/store"
)

// LineCredit keeps a session-credited source's lines with the session that wrote them. Such a
// source (parser.CreditsLinesPerSession) writes a session's whole line count onto one of its
// model rows, so pairing that row's lines with that row's cost pairs them with part of the
// session: with one model's tokens in a per-model split, and with only the priced models' cost
// when the session also ran an unpriced one.
//
// A usage row carries no session id. Its grouping without the model is the nearest thing: every
// record of one session shares its day, project and entrypoint, so a session never spans two
// groups. Two sessions on the same day and project can share one, so what the group knows is
// that several models ran there, not that one session ran them: a group like that leaves out
// more than one session may have needed, counted beside the ratio and never paired with the
// wrong cost.
type LineCredit struct {
	groups map[creditKey]*creditGroup
}

type creditKey struct {
	day, tool, project, entrypoint, member, granularity, task, outcome, difficulty string
}

type creditGroup struct {
	models   map[string]struct{}
	unpriced bool
}

func creditKeyOf(r *store.UsageRow) creditKey {
	return creditKey{
		r.Day, r.Tool, r.Project, r.Entrypoint, r.Member, r.Granularity, r.Task, r.Outcome, r.Difficulty,
	}
}

// NewLineCredit groups the session-credited rows of one row set. Build it over every row a
// basis will see, before any split: a split by model is exactly what hides a session's other
// models from the row that carries its lines.
func NewLineCredit(rows []store.UsageRow, t pricing.Table) LineCredit {
	c := LineCredit{groups: make(map[creditKey]*creditGroup)}
	for i := range rows {
		r := &rows[i]
		if !parser.CreditsLinesPerSession(r.Tool) {
			continue
		}
		k := creditKeyOf(r)
		g, ok := c.groups[k]
		if !ok {
			g = &creditGroup{models: make(map[string]struct{})}
			c.groups[k] = g
		}
		g.models[r.Model] = struct{}{}
		if _, priced := t.CostTokens(r.Model, pricing.Tokens{}); !priced {
			g.unpriced = true
		}
	}
	return c
}

// Fold adds r to b. cost and priced are r's own price lookup. perModel says b is one model's
// share of the window. A group that ran several models goes to the shared bucket whole when its
// lines cannot be paired with its cost: in a per-model basis, where no one model can claim them,
// and anywhere when one of its models has no known price. A group of one model needs neither
// rule, since its row's own price lookup already speaks for the whole group.
func (c LineCredit) Fold(b *LineRateBasis, r *store.UsageRow, cost float64, priced, perModel bool) {
	tokens := RowTokens(r)
	if g, ok := c.groups[creditKeyOf(r)]; ok && len(g.models) > 1 && (perModel || g.unpriced) {
		b.LineRows++
		b.SharedTokens += tokens
		b.SharedLines += r.LinesAdded
		return
	}
	b.add(r.Tool, r.LinesAdded, tokens, cost, priced)
}
