package github

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

// pullRequestQuery is the only question assaio asks the forge: for one repository, its pull
// requests newest update first, each with the fields below and nothing else. It names no title,
// body, branch, label, author, reviewer, comment or check; a test holds it to that list. The
// cursor is left out on the first page, where GraphQL reads it as null.
const pullRequestQuery = `query($owner: String!, $name: String!, $cursor: String) {
  repository(owner: $owner, name: $name) {
    pullRequests(first: 50, after: $cursor, orderBy: {field: UPDATED_AT, direction: DESC}) {
      pageInfo { hasNextPage endCursor }
      nodes {
        id number state updatedAt mergedAt
        mergeCommit { oid }
        commits(first: 100) { totalCount nodes { commit { oid } } }
      }
    }
  }
}`

// node is one pull request as the query returns it.
type node struct {
	ID          string    `json:"id"`
	Number      int64     `json:"number"`
	State       string    `json:"state"`
	UpdatedAt   time.Time `json:"updatedAt"`
	MergedAt    time.Time `json:"mergedAt"`
	MergeCommit *struct {
		OID string `json:"oid"`
	} `json:"mergeCommit"`
	Commits struct {
		TotalCount int64 `json:"totalCount"`
		Nodes      []struct {
			Commit struct {
				OID string `json:"oid"`
			} `json:"commit"`
		} `json:"nodes"`
	} `json:"commits"`
}

type page struct {
	Nodes       []node
	HasNextPage bool
	EndCursor   string
}

type response struct {
	Data *struct {
		Repository *struct {
			PullRequests struct {
				PageInfo struct {
					HasNextPage bool   `json:"hasNextPage"`
					EndCursor   string `json:"endCursor"`
				} `json:"pageInfo"`
				Nodes []node `json:"nodes"`
			} `json:"pullRequests"`
		} `json:"repository"`
	} `json:"data"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

// forgeError is an error the forge itself reported in a GraphQL answer, as opposed to an answer
// that was not one: its message says more than gh's exit status does.
type forgeError struct{ message string }

func (e *forgeError) Error() string { return "the forge answered with an error: " + e.message }

// parsePage reads one response. Any error the forge reports fails the page even beside partial
// data, and so does a missing repository: a page read in part is not a page.
func parsePage(body []byte) (page, error) {
	var r response
	dec := json.NewDecoder(bytes.NewReader(body))
	if err := dec.Decode(&r); err != nil {
		return page{}, fmt.Errorf("unreadable response: %w", err)
	}
	if len(r.Errors) > 0 {
		return page{}, &forgeError{message: strings.TrimSpace(r.Errors[0].Message)}
	}
	if r.Data == nil || r.Data.Repository == nil {
		return page{}, errors.New("the forge found no such repository")
	}
	prs := r.Data.Repository.PullRequests
	return page{Nodes: prs.Nodes, HasNextPage: prs.PageInfo.HasNextPage, EndCursor: prs.PageInfo.EndCursor}, nil
}
