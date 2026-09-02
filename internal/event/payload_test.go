package event

import (
	"reflect"
	"strings"
	"testing"
	"time"
)

// payloads is every concrete payload this build defines. The registry and this list must
// agree: a type nothing can fill is a name pretending to be a capability.
var payloads = []Payload{Commit{}}

func TestEveryRegisteredTypeHasAPayload(t *testing.T) {
	have := map[string]bool{}
	for _, p := range payloads {
		have[p.eventType()] = true
	}
	for _, name := range types {
		if !have[name] {
			t.Errorf("type %s is registered but no payload answers to it", name)
		}
	}
	for _, p := range payloads {
		if !known(p.eventType()) {
			t.Errorf("payload %T answers to unregistered type %s", p, p.eventType())
		}
	}
}

// stringFields is every string the contract is allowed to carry, and why each one is safe.
// A newly added string field fails this test until it is listed here, which is the whole
// point: "prompts and code are never collected" only stays true if nothing free-text can
// enter an envelope in the first place (ADR 0007).
var stringFields = map[string]string{
	"Event.Type":            "closed vocabulary",
	"Event.ID":              "the source's own key for an artifact it already had",
	"Event.TimeSource":      "closed vocabulary",
	"Event.Grain":           "closed vocabulary",
	"Event.Privacy":         "closed vocabulary",
	"Event.Provenance":      "closed vocabulary",
	"Event.Source.Name":     "the tool or connector name assaio itself assigns",
	"Event.Source.Version":  "the source's own format or API version",
	"Event.Source.Build":    "the assaio build that read it",
	"Event.Subject.Project": "a repository basename, never a path",
	"Event.Subject.Session": "the session id the tool assigned",
	"Event.Subject.Member":  "a pseudonymous member id set by the server",
}

func TestContractCarriesNoFreeText(t *testing.T) {
	roots := []reflect.Type{reflect.TypeOf(Event{})}
	for _, p := range payloads {
		roots = append(roots, reflect.TypeOf(p))
	}
	for _, root := range roots {
		walkStringFields(t, root, root.Name())
	}
}

func walkStringFields(t *testing.T, typ reflect.Type, path string) {
	t.Helper()
	for i := range typ.NumField() {
		f := typ.Field(i)
		name := path + "." + f.Name
		switch f.Type.Kind() {
		case reflect.String:
			if _, ok := stringFields[name]; !ok {
				t.Errorf("%s is a string field the contract does not account for -- "+
					"add it to stringFields with why it cannot carry a prompt, a diff or a path", name)
			}
		case reflect.Struct:
			if f.Type == reflect.TypeOf(time.Time{}) {
				continue
			}
			walkStringFields(t, f.Type, name)
		case reflect.Int, reflect.Int64, reflect.Bool, reflect.Interface:
			// A number or a two-value flag cannot carry content; the payload interface is
			// walked as each concrete type above.
		default:
			t.Errorf("%s is a %s -- a map, slice or pointer is room for content the "+
				"contract promises it cannot hold", name, f.Type.Kind())
		}
	}
}

func assertErr(t *testing.T, err error, want string) {
	t.Helper()
	switch {
	case want == "" && err != nil:
		t.Fatalf("want no error, got %v", err)
	case want != "" && err == nil:
		t.Fatalf("want an error mentioning %q, got none", want)
	case want != "" && !strings.Contains(err.Error(), want):
		t.Fatalf("error %q does not mention %q", err, want)
	}
}
