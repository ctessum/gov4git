package concern

import (
	"bytes"
	"context"
	"fmt"

	"github.com/gov4git/gov4git/v2/proto/ballot/ballotproto"
	"github.com/gov4git/gov4git/v2/proto/motion/motionpolicies/pmp_1"
	"github.com/gov4git/gov4git/v2/proto/motion/motionproto"
	"github.com/gov4git/gov4git/v2/proto/notice"
)

func cancelNotice(ctx context.Context, motion motionproto.Motion, outcome ballotproto.Outcome) notice.Notices {

	var w bytes.Buffer

	fmt.Fprintf(&w, "This issue, managed as Gov4Git concern `%v`, has been cancelled 🌂\n\n", motion.ID)

	fmt.Fprintf(&w, "The issue priority tally was `%0.6f`.\n\n", outcome.Scores[pmp_1.ConcernBallotChoice])

	// refunded
	fmt.Fprintf(&w, "Refunds issued:\n")
	for _, refund := range ballotproto.FlattenRefunds(outcome.Refunded) {
		fmt.Fprintf(&w, "- User @%v was refunded `%0.6f` credits\n", refund.User, refund.Amount.Quantity)
	}
	fmt.Fprintln(&w, "")

	// tally by user
	fmt.Fprintf(&w, "Tally breakdown by user:\n")
	for user, ss := range outcome.ScoresByUser {
		fmt.Fprintf(&w, "- User @%v contributed `%0.6f` votes\n", user, ss[pmp_1.ConcernBallotChoice].Vote())
	}

	return notice.NewNotice(ctx, w.String())
}

func closeNotice(
	ctx context.Context,
	con motionproto.Motion,
	outcome ballotproto.Outcome,
	prop motionproto.Motion,

) notice.Notices {

	var w bytes.Buffer

	fmt.Fprintf(&w, "This issue, managed as Gov4Git concern `%v`, has been closed 🎉\n\n", con.ID)

	fmt.Fprintf(&w, "The issue priority tally was `%0.6f`.\n\n", outcome.Scores[pmp_1.ConcernBallotChoice])

	// resolved by PR
	fmt.Fprintf(&w, "Ths issue was resolved by [PR #%v](%v):\n\n", prop.ID, prop.TrackerURL)

	// tally by user
	fmt.Fprintf(&w, "Tally breakdown by user:\n")
	for user, ss := range outcome.ScoresByUser {
		fmt.Fprintf(&w, "- User @%v contributed `%0.6f` votes\n", user, ss[pmp_1.ConcernBallotChoice].Vote())
	}

	return notice.NewNotice(ctx, w.String())
}
