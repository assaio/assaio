package github

import "time"

// History is walked independently of sessions. Nesting it into the fifty-PR query exceeds
// GitHub's resource budget even on small repositories; both walks retain the same call caps.
const historyQuery = `query($owner: String!, $name: String!, $cursor: String) {
  repository(owner: $owner, name: $name) {
    pullRequests(first: 5, after: $cursor, orderBy: {field: UPDATED_AT, direction: DESC}) {
      totalCount
      pageInfo { hasNextPage endCursor }
      nodes {
        id number updatedAt
        history: commits(first: 100) {
          totalCount
          nodes {
            commit {
              oid
              checkSuites(last: 10) {
                totalCount
                nodes {
                  id status conclusion createdAt updatedAt
                  checkRuns(last: 5, filterBy: {checkType: ALL}) {
                    totalCount
                    nodes { id status conclusion startedAt completedAt }
                  }
                }
              }
            }
          }
        }
      }
    }
  }
}`

type historyConnection struct {
	TotalCount *int64 `json:"totalCount"`
	Nodes      []*struct {
		Commit *struct {
			OID         string `json:"oid"`
			CheckSuites *struct {
				TotalCount *int64       `json:"totalCount"`
				Nodes      []*suiteNode `json:"nodes"`
			} `json:"checkSuites"`
		} `json:"commit"`
	} `json:"nodes"`
}

type suiteNode struct {
	ID         string    `json:"id"`
	Status     string    `json:"status"`
	Conclusion string    `json:"conclusion"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
	CheckRuns  *struct {
		TotalCount *int64 `json:"totalCount"`
		Nodes      []*struct {
			ID          string    `json:"id"`
			Status      string    `json:"status"`
			Conclusion  string    `json:"conclusion"`
			StartedAt   time.Time `json:"startedAt"`
			CompletedAt time.Time `json:"completedAt"`
		} `json:"nodes"`
	} `json:"checkRuns"`
}
