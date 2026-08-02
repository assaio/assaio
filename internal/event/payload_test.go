package event

import (
	"reflect"
	"strings"
	"testing"
	"time"
)

// payloads is every concrete payload this build defines. The registry and this list must
// agree: a type nothing can fill is a name pretending to be a capability.
var payloads = []Payload{Usage{}, Edit{}}

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

func TestUsageValidate(t *testing.T) {
	tests := []struct {
		name    string
		usage   Usage
		wantErr string
	}{
		{"empty is valid: a turn can legitimately cost nothing", Usage{}, ""},
		{"counts only", Usage{InputTokens: 1, OutputTokens: 2, ReasoningTokens: 2}, ""},
		{"negative input", Usage{InputTokens: -1}, "inputTokens is negative"},
		{"negative cache read", Usage{CacheReadTokens: -5}, "cacheReadTokens is negative"},
		{
			"reasoning cannot exceed the output it is part of",
			Usage{OutputTokens: 10, ReasoningTokens: 11},
			"exceeds outputTokens",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assertErr(t, tc.usage.validate(), tc.wantErr)
		})
	}
}

func TestEditValidate(t *testing.T) {
	tests := []struct {
		name    string
		edit    Edit
		wantErr string
	}{
		{"one added line is activity", Edit{LinesAdded: 1}, ""},
		{"an edit with no lines is still activity", Edit{Edits: 1}, ""},
		{"all zero is not an observation", Edit{}, "carries no activity"},
		{"negative removed", Edit{LinesAdded: 1, LinesRemoved: -2}, "linesRemoved is negative"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assertErr(t, tc.edit.validate(), tc.wantErr)
		})
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
	"Usage.Model":           "the model identifier the tool reported",
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
		case reflect.Int, reflect.Int64, reflect.Interface:
			// Numbers cannot carry content; the payload interface is walked as each
			// concrete type above.
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
