package attribution

import (
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/assaio/assaio/internal/event"
)

func renderDeliveryCoverage(w io.Writer, c *Changes, since time.Time) error {
	total := "—"
	if c.RepositoryTotal != nil {
		total = strconv.FormatInt(*c.RepositoryTotal, 10)
	}
	if _, err := fmt.Fprintf(w, renderRepositoryTotalRead, total, c.PullRequestsRead, c.HistoryRead); err != nil {
		return err
	}
	bound := ""
	if c.HistoryBackTo.After(since) {
		bound = "; stopped at the 20-page bound"
	}
	_, err := fmt.Fprintf(w, renderHistoryCoverage, c.Source, c.ObservedAt.Format(time.RFC3339),
		c.HistoryBackTo.Format(time.RFC3339), c.HistoryRead, bound)
	return err
}

func renderPRHistory(w io.Writer, pr *LinkedPullRequest) error {
	if pr.RequestedChangesRevisions != nil {
		if _, err := fmt.Fprintf(w, renderReviewedRevisions, derivedCount(pr.RequestedChangesRevisions),
			readCount(pr.Reviews), deliveryReason(pr.RequestedChangesRevisions.Reason)); err != nil {
			return err
		}
	}
	if pr.ReviewRounds != nil {
		if _, err := fmt.Fprintf(w, renderExactRoundsWithheld, deliveryReason(pr.ReviewRounds.Reason)); err != nil {
			return err
		}
	}
	if pr.MergeMethod != nil {
		method := "—"
		if pr.MergeMethod.Value != nil {
			method = *pr.MergeMethod.Value
		}
		if _, err := fmt.Fprintf(w, renderMergeMethod, method, pr.MergeMethod.Source.Name, deliveryReason(pr.MergeMethod.Reason)); err != nil {
			return err
		}
	}
	h := pr.HistoricalChecks
	if h == nil {
		return nil
	}
	suites, runs := []string{}, []string{}
	for _, commit := range h.ByCommit {
		suites = append(suites, commit.Commit+" "+populationCount(commit.Suites))
		for i := range commit.SuiteObservations {
			s := &commit.SuiteObservations[i]
			runs = append(runs, s.ID+" "+populationCount(s.Payload.Runs))
		}
	}
	if _, err := fmt.Fprintf(w, renderRawHistoryCoverage, populationCount(h.Commits), coverageList(suites), coverageList(runs)); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w, "      "+h.ReadState+": "+deliveryReason(h.ReadState)); err != nil {
		return err
	}
	_, err := fmt.Fprint(w, renderSourceTimeCaveat)
	return err
}

func renderDeliveryRates(w io.Writer, c *Changes) error {
	for i := range c.Rates {
		rate := &c.Rates[i]
		if rate.ID == "failed-check-run-presence" {
			if _, err := fmt.Fprintf(w, renderCiRatioWithheld, deliveryReason(rate.Reason)); err != nil {
				return err
			}
			continue
		}
		value, numerator := "—", "—"
		if rate.Value != nil {
			value = fmt.Sprintf("%.1f%%", *rate.Value*100)
		}
		if rate.Numerator != nil {
			numerator = strconv.Itoa(*rate.Numerator)
		}
		if _, err := fmt.Fprintf(w, renderSnapshotRate, renderSnapshotRateLabel, value, numerator,
			rate.Denominator, rate.Eligible, deliveryReason(rate.Reason)); err != nil {
			return err
		}
		for _, gap := range rate.Gaps {
			if _, err := fmt.Fprintf(w, "    %s=%d\n", gap.Reason, gap.PullRequests); err != nil {
				return err
			}
		}
	}
	return nil
}

func derivedCount(c *DerivedCount) string {
	if c == nil || c.Value == nil {
		return "—"
	}
	return strconv.Itoa(*c.Value)
}

func populationCount(p *event.Population) string {
	if p == nil {
		return "—"
	}
	return fmt.Sprintf("%d/%d", p.Listed, p.Total)
}

func coverageList(parts []string) string {
	if len(parts) == 0 {
		return "—"
	}
	return strings.Join(parts, "; ")
}

func deliveryReason(reason string) string {
	if reason == "" || reason == "read" {
		return "—"
	}
	if text, ok := deliveryReasons[reason]; ok {
		return text
	}
	return reason
}
