package collab

import (
	"context"
	"fmt"

	"github.com/gov4git/gov4git/proto"
	"github.com/gov4git/gov4git/proto/ballot/ballot"
	"github.com/gov4git/gov4git/proto/gov"
	"github.com/gov4git/lib4git/form"
	"github.com/gov4git/lib4git/git"
)

func ScoreMotions(ctx context.Context, addr gov.GovAddress) git.Change[form.Map, Motions] {

	cloned := gov.Clone(ctx, addr)
	chg := ScoreMotions_StageOnly(ctx, addr, cloned.Tree())
	proto.Commit(ctx, cloned.Tree(), chg) // XXX: if no change
	cloned.Push(ctx)
	return chg
}

func ScoreMotions_StageOnly(ctx context.Context, govAddr gov.GovAddress, t *git.Tree) git.Change[form.Map, Motions] {

	motions := ListMotions_Local(ctx, t)
	for i, motion := range motions {
		switch {
		case motion.Scoring.Fixed != nil:
			motions[i].Score = *motion.Scoring.Fixed
			motionKV.Set(ctx, motionNS, t, motions[i].ID, motions[i])
		case motion.Scoring.Poll != nil:
			ast := ballot.Show_Local(ctx, govAddr, t, *motion.Scoring.Poll)
			motions[i].Score = ast.Tally.Scores[MotionPollBallotChoice]
			motionKV.Set(ctx, motionNS, t, motions[i].ID, motions[i])
		}
	}

	motions.Sort()

	return git.NewChange(
		fmt.Sprintf("Score all %d motions", len(motions)),
		"collab_score_motions",
		form.Map{},
		motions,
		form.Forms{},
	)
}
