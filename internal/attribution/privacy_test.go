package attribution

import (
	"reflect"
	"strings"
	"testing"
	"time"
)

// personOrRank are the words a field would carry if the document named who did the work or
// ordered anyone by it. ADR 0020 keeps both out of every correlation output. The check reads
// field names as JSON serializes them -- what a reader, a script or an agent skill sees -- and
// not values: a login inside an innocently named string would pass it, which is why the
// document carries no free string from a forge at all.
var personOrRank = []string{
	"author", "committer", "login", "user", "member", "person", "people", "reviewer", "assignee",
	"owner", "actor", "creator", "mergedby", "approver", "requester", "sender", "account", "handle",
	"email", "rank", "score", "leader", "percentile",
}

// TestTheDocumentNamesNoPersonAndRanksNothing walks every field the evidence document can
// serialize, nested types included, so a field added later has to pass the same check. An
// interface field fails outright: whatever it holds is invisible to this walk.
func TestTheDocumentNamesNoPersonAndRanksNothing(t *testing.T) {
	walkJSONNames(t, reflect.TypeOf(Document{}), "Document", map[reflect.Type]bool{})
}

func walkJSONNames(t *testing.T, typ reflect.Type, path string, seen map[reflect.Type]bool) {
	t.Helper()
	for {
		switch typ.Kind() {
		case reflect.Slice, reflect.Array, reflect.Pointer:
			typ = typ.Elem()
			continue
		case reflect.Map:
			walkJSONNames(t, typ.Key(), path+"[key]", seen)
			typ = typ.Elem()
			continue
		case reflect.Interface:
			t.Errorf("%s is an interface, so what it serializes cannot be checked", path)
			return
		}
		break
	}
	if typ.Kind() != reflect.Struct || typ == reflect.TypeOf(time.Time{}) || seen[typ] {
		return
	}
	seen[typ] = true
	for i := range typ.NumField() {
		f := typ.Field(i)
		name := strings.Split(f.Tag.Get("json"), ",")[0]
		if name == "" {
			name = f.Name
		}
		lower := strings.ToLower(name)
		for _, word := range personOrRank {
			if strings.Contains(lower, word) {
				t.Errorf("%s.%s serializes as %q, which names a person or a rank", path, f.Name, name)
			}
		}
		walkJSONNames(t, f.Type, path+"."+f.Name, seen)
	}
}
