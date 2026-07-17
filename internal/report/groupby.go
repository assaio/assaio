package report

import "sort"

// groupBy groups n items (indexed 0..n-1) by keyAt(i), creating each new group via
// newGroup and folding every member into it via accumulate, in key-sorted order.
func groupBy[T any](n int, keyAt func(i int) string, newGroup func(key string) T, accumulate func(g *T, i int)) []T {
	order := make([]string, 0)
	groups := make(map[string]*T)
	for i := range n {
		key := keyAt(i)
		g, ok := groups[key]
		if !ok {
			v := newGroup(key)
			g = &v
			groups[key] = g
			order = append(order, key)
		}
		accumulate(g, i)
	}

	sort.Strings(order)
	out := make([]T, 0, len(order))
	for _, key := range order {
		out = append(out, *groups[key])
	}
	return out
}
