package calibration

import (
	"encoding/json"
	"strconv"
	"strings"
)

// A metamorphic property is two encodings of the same fact that must produce the same total.
// It needs no expected value either, and it catches the one thing an invariant cannot: a
// reader that handles one encoding of a fact and silently drops the other. Both of v0.12's
// defects were exactly that, and so was v0.14's -- each looked correct on the encoding
// somebody had thought to write a fixture for.

// SplitResponseBlocks rewrites a Claude Code transcript so every assistant line becomes two
// lines carrying the same message id and the same usage, which is how the source writes a
// response with more than one content block. A token total must not move: the usage belongs
// to the response, not to the line. Counting per line is what made the flagship source's
// every token figure roughly double for eleven releases.
func SplitResponseBlocks(trace []byte) []byte {
	var out strings.Builder
	for i, line := range splitLines(trace) {
		var d map[string]json.RawMessage
		if err := json.Unmarshal(line, &d); err != nil || !isAssistantWithID(d) {
			writeLine(&out, line)
			continue
		}
		writeLine(&out, line)
		d["uuid"] = json.RawMessage(strconv.Quote("split-" + strconv.Itoa(i)))
		d["message"] = withEmptyContent(d["message"])
		echo, err := json.Marshal(d)
		if err != nil {
			continue
		}
		writeLine(&out, echo)
	}
	return []byte(out.String())
}

// CreationAsPatch rewrites a Claude Code file creation from the body form the source
// actually writes -- the whole file in `content` beside an empty structuredPatch -- into the
// patch form an ordinary edit uses. The line count must not move. Reading only the patch is
// what made every file the source created count as zero.
func CreationAsPatch(trace []byte) []byte {
	var out strings.Builder
	for _, line := range splitLines(trace) {
		var d map[string]json.RawMessage
		if err := json.Unmarshal(line, &d); err != nil {
			writeLine(&out, line)
			continue
		}
		rewritten, ok := creationToPatch(d)
		if !ok {
			writeLine(&out, line)
			continue
		}
		writeLine(&out, rewritten)
	}
	return []byte(out.String())
}

type toolResultShape struct {
	Type            string            `json:"type"`
	Content         string            `json:"content"`
	FilePath        string            `json:"filePath"`
	StructuredPatch []json.RawMessage `json:"structuredPatch"`
}

func creationToPatch(d map[string]json.RawMessage) ([]byte, bool) {
	raw, ok := d["toolUseResult"]
	if !ok {
		return nil, false
	}
	var t toolResultShape
	if err := json.Unmarshal(raw, &t); err != nil || t.Type != "create" || len(t.StructuredPatch) > 0 {
		return nil, false
	}
	body := strings.Split(strings.TrimSuffix(t.Content, "\n"), "\n")
	added := make([]string, 0, len(body))
	for _, l := range body {
		added = append(added, "+"+l)
	}
	patch, err := json.Marshal(map[string]any{
		"filePath":        t.FilePath,
		"structuredPatch": []map[string]any{{"lines": added}},
	})
	if err != nil {
		return nil, false
	}
	d["toolUseResult"] = patch
	line, err := json.Marshal(d)
	if err != nil {
		return nil, false
	}
	return line, true
}

// CreationAsDiff rewrites a Codex file creation from the whole-body `add` form into the
// unified diff an `update` uses. Same property, other source: assaio lost these lines here
// first (B119) and then again in Claude Code, which is what a shared property would have
// stopped the second time.
func CreationAsDiff(trace []byte) []byte {
	var out strings.Builder
	for _, line := range splitLines(trace) {
		var d map[string]json.RawMessage
		if err := json.Unmarshal(line, &d); err != nil {
			writeLine(&out, line)
			continue
		}
		rewritten, ok := codexAddToDiff(d)
		if !ok {
			writeLine(&out, line)
			continue
		}
		writeLine(&out, rewritten)
	}
	return []byte(out.String())
}

func codexAddToDiff(d map[string]json.RawMessage) ([]byte, bool) {
	var p struct {
		Type    string `json:"type"`
		Success bool   `json:"success"`
		Changes map[string]struct {
			Type    string `json:"type"`
			Content string `json:"content"`
		} `json:"changes"`
	}
	if err := json.Unmarshal(d["payload"], &p); err != nil || p.Type != "patch_apply_end" {
		return nil, false
	}
	changed := false
	rewritten := make(map[string]any, len(p.Changes))
	for path, c := range p.Changes {
		if c.Type != "add" {
			return nil, false
		}
		body := strings.Split(strings.TrimSuffix(c.Content, "\n"), "\n")
		diff := []string{"--- /dev/null", "+++ b/x", "@@ -0,0 +1," + strconv.Itoa(len(body)) + " @@"}
		for _, l := range body {
			diff = append(diff, "+"+l)
		}
		rewritten[path] = map[string]any{"type": "update", "unified_diff": strings.Join(diff, "\n")}
		changed = true
	}
	if !changed {
		return nil, false
	}
	payload, err := json.Marshal(map[string]any{
		"type": "patch_apply_end", "success": p.Success, "changes": rewritten,
	})
	if err != nil {
		return nil, false
	}
	d["payload"] = payload
	line, err := json.Marshal(d)
	if err != nil {
		return nil, false
	}
	return line, true
}

func isAssistantWithID(d map[string]json.RawMessage) bool {
	var typ string
	if err := json.Unmarshal(d["type"], &typ); err != nil || typ != "assistant" {
		return false
	}
	var m struct {
		ID string `json:"id"`
	}
	return json.Unmarshal(d["message"], &m) == nil && m.ID != ""
}

// withEmptyContent strips the echoed line's content blocks: a repeated block would be a
// second tool call, and the property under test is about the response's tokens.
func withEmptyContent(msg json.RawMessage) json.RawMessage {
	var m map[string]json.RawMessage
	if err := json.Unmarshal(msg, &m); err != nil {
		return msg
	}
	m["content"] = json.RawMessage("[]")
	out, err := json.Marshal(m)
	if err != nil {
		return msg
	}
	return out
}

func splitLines(trace []byte) [][]byte {
	var out [][]byte
	for _, l := range strings.Split(strings.TrimSuffix(string(trace), "\n"), "\n") {
		if l != "" {
			out = append(out, []byte(l))
		}
	}
	return out
}

func writeLine(b *strings.Builder, line []byte) {
	b.Write(line)
	b.WriteByte('\n')
}
