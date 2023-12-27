package metric

import "github.com/gov4git/gov4git/v2/proto/history"

var (
	metricHistoryNS = history.HistoryNS.Append("metric")
	metricHistory   = History{Root: metricHistoryNS}
)

type Event struct {
	Join    *JoinEvent    `json:"join"`
	Motion  *MotionEvent  `json:"motion"`
	Account *AccountEvent `json:"account"`
	Vote    *VoteEvent    `json:"vote"`
}
