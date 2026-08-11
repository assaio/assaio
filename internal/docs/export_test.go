package docs

// The checks that need the real command tree live in the external test package, because
// internal/cli imports this one. These give them the two internals they have to see.

// Exemptions is the exempt table.
func Exemptions() map[string]string { return exempt }

// Addressable reports whether a claim id names anything in ref.
func Addressable(ref *Reference, id string) bool { return newIndex(ref).ids[id] }
