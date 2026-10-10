package event

import (
	"errors"
	"fmt"
	"time"
)

// Population is the source's whole connection and the part actually returned. Nil means
// unavailable; a present zero is a source-stated empty connection.
type Population struct {
	Total  int64 `json:"total"`
	Listed int64 `json:"listed"`
}

func (p Population) validate() error {
	if p.Total < 0 || p.Listed < 0 || p.Listed > p.Total {
		return fmt.Errorf("invalid connection population %d/%d", p.Listed, p.Total)
	}
	return nil
}

// CheckSuite observes a suite on a currently listed commit, including all-run read coverage.
// It does not establish a PR pipeline, workflow identity or repair sequence.
type CheckSuite struct {
	Number     int64       `json:"number"`
	Commit     string      `json:"commit"`
	State      string      `json:"state"`
	Conclusion string      `json:"conclusion,omitempty"`
	CreatedAt  time.Time   `json:"createdAt,omitzero"`
	UpdatedAt  time.Time   `json:"updatedAt,omitzero"`
	Runs       *Population `json:"runs"`
}

func (CheckSuite) eventType() string { return TypeCheckSuite }

//nolint:gocritic // the Payload interface is satisfied by values; validated once per source observation.
func (p CheckSuite) validate() error {
	if !isObjectID(p.Commit) {
		return errors.New("suite commit is not a commit hash")
	}
	if err := (Check{Number: p.Number, Kind: "run", State: p.State, Conclusion: p.Conclusion}).validate(); err != nil {
		return err
	}
	if p.Runs != nil {
		return p.Runs.validate()
	}
	return nil
}

func isForgeID(id string) bool {
	if id == "" || len(id) > 128 {
		return false
	}
	for _, r := range id {
		if (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') && r != '_' && r != '-' && r != '=' {
			return false
		}
	}
	return true
}
