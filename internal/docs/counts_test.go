package docs

import "testing"

func TestNumberWord(t *testing.T) {
	tests := map[int]string{
		0: "zero", 5: "five", 19: "nineteen", 20: "twenty", 22: "twenty-two",
		35: "thirty-five", 90: "ninety", 99: "ninety-nine", 100: "", -1: "",
	}
	for n, want := range tests {
		if got := numberWord(n); got != want {
			t.Errorf("numberWord(%d) = %q, want %q", n, got, want)
		}
	}
}

func TestMentionsCount(t *testing.T) {
	tests := []struct {
		text string
		n    int
		want bool
	}{
		{"Nineteen reads, one faceplate", 19, true},
		{"19 reads", 19, true},
		{"twenty-two figures", 22, true},
		{"Nineteen reads", 20, false},
		// "nine" must not be found inside "nineteen", or a count could fall behind by ten
		// without anything noticing.
		{"Nineteen reads", 9, false},
		{"five sources", 5, true},
		{"twenty-five", 5, false},
	}
	for _, tt := range tests {
		if got := mentionsCount(tt.text, tt.n); got != tt.want {
			t.Errorf("mentionsCount(%q, %d) = %v, want %v", tt.text, tt.n, got, tt.want)
		}
	}
}

func TestElementTexts(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    []string
	}{
		{"reads the element's own text", `<span data-claim="signals.count">Twenty-two</span> figures`, []string{"Twenty-two"}},
		{"tolerates other attributes after it", `<span data-claim="signals.count" class="n">22</span>`, []string{"22"}},
		{"absent claim reads nothing", `<span>22</span>`, nil},
		{
			name: "every occurrence is returned, so none can fall behind alone",
			content: `<h2><span data-claim="signals.count">Twenty-two</span> figures</h2>` +
				`<p>which of the <span data-claim="signals.count">21</span> figures</p>`,
			want: []string{"Twenty-two", "21"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := elementTexts(tt.content, "signals.count")
			if len(got) != len(tt.want) {
				t.Fatalf("elementTexts = %q, want %q", got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("elementTexts[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}
