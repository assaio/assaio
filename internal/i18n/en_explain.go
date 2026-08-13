package i18n

// enExplain maps a validator's name to its long-form page. The pages live in the two
// files beside this one, grouped by what they measure. Each page expands the validator's
// own Describe, HowToRead, and Caveats -- it must never contradict them, since both
// surfaces describe the same metric.
var enExplain = map[string]string{
	"adoption":           explainAdoption,
	"burn-anomaly":       explainBurnAnomaly,
	"cache-hygiene":      explainCacheHygiene,
	"concentration":      explainConcentration,
	"context":            explainContext,
	"coverage":           explainCoverage,
	"edit-loops":         explainEditLoops,
	"explore-produce":    explainExploreProduce,
	"friction":           explainFriction,
	"intent":             explainIntent,
	"model-fit":          explainModelFit,
	"model-right-sizing": explainModelRightSizing,
	"reasoning-share":    explainReasoningShare,
	"recovery":           explainRecovery,
	"rework":             explainRework,
	"rhythm":             explainRhythm,
	"session-taxonomy":   explainSessionTaxonomy,
	"skill-economics":    explainSkillEconomics,
	"subscription-fit":   explainSubscriptionFit,
	"throughput":         explainThroughput,
	"turn-efficiency":    explainTurnEfficiency,
}
