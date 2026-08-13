package calibration_test

import (
	"sort"
	"testing"

	"github.com/assaio/assaio/internal/calibration"
	"github.com/assaio/assaio/internal/pricing"
)

// A price table that has fallen behind the models in use is indistinguishable from a
// complete one from the inside: the parser is right, the arithmetic is right, and the
// headline is short by whatever the missing model cost. On the maintainer's store that was
// 45.5% of the tokens for five weeks, and no test failed.
//
// This is the half of the guard CI can run. It fails when a trace teaches assaio to read a
// model the vendored table cannot cost -- the case where the repo itself knows about a model
// nobody priced. It cannot see a model that only exists in someone's store, which is what
// `doctor --strict` is for; the two together are the mechanism, and neither alone is.
func TestEveryCalibratedModelHasAPrice(t *testing.T) {
	adjudicated, err := calibration.LoadAdjudicated("testdata")
	if err != nil {
		t.Fatal(err)
	}
	table, err := pricing.Load()
	if err != nil {
		t.Fatal(err)
	}
	unpriced := map[string][]string{}
	for i := range adjudicated {
		a := &adjudicated[i]
		parse, ok := parsers[a.Source]
		if !ok {
			continue // TestEverySourceIsCalibrated owns that absence
		}
		records, _, _, parseErr := parse(a.Trace)
		if parseErr != nil {
			t.Fatalf("%s/%s: %v", a.Source, a.Trace, parseErr)
		}
		for j := range records {
			model := records[j].Model
			if model == "" || priced(table, model) {
				continue
			}
			unpriced[model] = append(unpriced[model], a.Source)
		}
	}
	if len(unpriced) == 0 {
		return
	}
	models := make([]string, 0, len(unpriced))
	for model := range unpriced {
		models = append(models, model)
	}
	sort.Strings(models)
	t.Fatalf("the vendored price table cannot cost %v, which these traces prove assaio reads -- refresh internal/pricing/litellm.json", models)
}

// priced asks the table the same two questions Cost does, so a model this test passes is a
// model a report can actually price.
func priced(t pricing.Table, model string) bool {
	if _, ok := t[model]; ok {
		return true
	}
	_, ok := t[pricing.NormalizeModel(model)]
	return ok
}
