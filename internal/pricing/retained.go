package pricing

// fillFrom adds to t every key of retained that the snapshot no longer lists, marked Retained,
// and returns how many it added. A key the snapshot still lists keeps the snapshot's answer, even
// when that answer is "no token rate".
//
// retained is every unprefixed key a vendored snapshot has priced since the ledger was added,
// at its latest price. LiteLLM drops a model once it is past its deprecation date, and cost is
// computed at read time, so a refresh that dropped the key would turn history that was priced
// into an unpriced share. The last list price LiteLLM published stays the estimate for it.
func (t Table) fillFrom(retained Table, listed map[string]bool) int {
	added := 0
	for name, p := range retained {
		if listed[name] {
			continue
		}
		p.Retained = true
		t[name] = p
		added++
	}
	return added
}
