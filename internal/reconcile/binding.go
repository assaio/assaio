package reconcile

import (
	"fmt"
	"sort"
	"strings"
)

// Binding records which column of the export was read as which field. It is printed with
// every reconciliation: assaio cannot verify a vendor's column names against a real export
// it has never seen, so the honest substitute is showing the operator exactly what it bound
// and letting them catch a wrong guess.
type Binding struct {
	Day      string `json:"day"`
	Cost     string `json:"cost"`
	Model    string `json:"model,omitempty"`
	Tokens   string `json:"tokens,omitempty"`
	Currency string `json:"currency,omitempty"`
}

// Field names --map accepts, and the header aliases each is recognized by. The aliases are
// candidates, never a claim of vendor support: an export whose headers miss them all is not
// unsupported, it just needs --map.
const (
	FieldDay      = "day"
	FieldCost     = "cost"
	FieldModel    = "model"
	FieldTokens   = "tokens"
	FieldCurrency = "currency"
)

var aliases = map[string][]string{
	FieldDay:      {"day", "date", "usage_date", "timestamp", "start_time", "period", "bucket"},
	FieldCost:     {"cost", "amount", "cost_usd", "amount_usd", "spend", "total_cost", "charge"},
	FieldModel:    {"model", "model_name", "sku", "line_item", "description"},
	FieldTokens:   {"tokens", "total_tokens", "token_count", "usage_tokens"},
	FieldCurrency: {"currency", "currency_code"},
}

// bindColumns matches an export's headers against the aliases, with overrides winning over
// any alias. Day and cost are required: without a date there is no window to compare over,
// and without money there is nothing to compare. Everything else is optional and its
// absence costs a named cause its evidence rather than producing a zero.
func bindColumns(headers []string, overrides map[string]string) (Binding, map[string]int, error) {
	index := make(map[string]int, len(headers))
	for i, h := range headers {
		index[normalize(h)] = i
	}
	cols := map[string]int{}
	var b Binding
	for _, f := range []string{FieldDay, FieldCost, FieldModel, FieldTokens, FieldCurrency} {
		name, i, ok := resolveField(f, index, headers, overrides)
		if !ok {
			continue
		}
		cols[f] = i
		switch f {
		case FieldDay:
			b.Day = name
		case FieldCost:
			b.Cost = name
		case FieldModel:
			b.Model = name
		case FieldTokens:
			b.Tokens = name
		case FieldCurrency:
			b.Currency = name
		}
	}
	for _, f := range []string{FieldDay, FieldCost} {
		if _, ok := cols[f]; !ok {
			return Binding{}, nil, fmt.Errorf(
				"no column read as %s; pass --map %s=<column>. Columns seen: %s",
				f, f, strings.Join(headers, ", "),
			)
		}
	}
	return b, cols, nil
}

// resolveField finds the column for one field: an explicit override first, then the first
// alias present in the header row.
func resolveField(field string, index map[string]int, headers []string, overrides map[string]string) (name string, col int, ok bool) {
	if want, has := overrides[field]; has {
		i, found := index[normalize(want)]
		if !found {
			return "", 0, false
		}
		return headers[i], i, true
	}
	for _, a := range aliases[field] {
		if i, found := index[a]; found {
			return headers[i], i, true
		}
	}
	return "", 0, false
}

// ParseMap turns "cost=amount,day=date" into overrides, rejecting a field this package has
// no column for -- a silently ignored mapping would leave the operator believing they had
// corrected a binding they had not.
func ParseMap(spec string) (map[string]string, error) {
	out := map[string]string{}
	if strings.TrimSpace(spec) == "" {
		return out, nil
	}
	for _, pair := range strings.Split(spec, ",") {
		field, column, found := strings.Cut(strings.TrimSpace(pair), "=")
		field, column = normalize(field), strings.TrimSpace(column)
		if !found || field == "" || column == "" {
			return nil, fmt.Errorf("bad --map entry %q (want field=column)", pair)
		}
		if _, known := aliases[field]; !known {
			return nil, fmt.Errorf("unknown --map field %q (want one of %s)", field, strings.Join(mappableFields(), ", "))
		}
		out[field] = column
	}
	return out, nil
}

func mappableFields() []string {
	out := make([]string, 0, len(aliases))
	for f := range aliases {
		out = append(out, f)
	}
	sort.Strings(out)
	return out
}

// normalize lowercases a header and folds the separators exports differ on, so "Usage Date",
// "usage-date" and "usage_date" bind to the same field.
func normalize(h string) string {
	h = strings.ToLower(strings.TrimSpace(h))
	h = strings.NewReplacer(" ", "_", "-", "_", ".", "_").Replace(h)
	return strings.Trim(h, "_")
}
