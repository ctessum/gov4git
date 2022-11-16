package ballot

import (
	"context"

	"github.com/gov4git/gov4git/mod"
	"github.com/gov4git/gov4git/mod/gov"
	"github.com/gov4git/gov4git/mod/id"
	"github.com/gov4git/gov4git/mod/mail"
	"github.com/gov4git/lib4git/git"
	"github.com/gov4git/lib4git/must"
	"github.com/gov4git/lib4git/ns"
)

func Vote[S Strategy](
	ctx context.Context,
	voterAddr id.OwnerAddress,
	govAddr gov.CommunityAddress,
	ballotName ns.NS,
	elections []Election,
) git.Change[mail.SeqNo] {

	govRepo := git.CloneRepo(ctx, git.Address(govAddr))
	voterRepo, voterTree := id.CloneOwner(ctx, voterAddr)
	chg := VoteStageOnly[S](ctx, voterAddr, govAddr, voterTree, govRepo, ballotName, elections)
	mod.Commit(ctx, voterTree.Public, chg.Msg)
	git.Push(ctx, voterRepo.Public)

	return chg
}

func VoteStageOnly[S Strategy](
	ctx context.Context,
	voterAddr id.OwnerAddress,
	govAddr gov.CommunityAddress,
	voterTree id.OwnerTree,
	govRepo *git.Repository,
	ballotName ns.NS,
	elections []Election,
) git.Change[mail.SeqNo] {

	govTree := git.Worktree(ctx, govRepo)
	openAdNS := OpenBallotNS[S](ballotName).Sub(adFilebase)
	ad := git.FromFile[Advertisement](ctx, govTree, openAdNS.Path())
	verifyElections[S](ctx, voterAddr, govAddr, voterTree, govTree, ad, elections)
	envelope := VoteEnvelope{
		AdCommit:  git.Head(ctx, govRepo),
		Ad:        ad,
		Elections: elections,
	}

	return mail.SendSignedStageOnly(ctx, voterTree, govTree, BallotTopic[S](ballotName), envelope)
}

func verifyElections[S Strategy](
	ctx context.Context,
	voterAddr id.OwnerAddress,
	govAddr gov.CommunityAddress,
	voterTree id.OwnerTree,
	govTree *git.Tree,
	ad Advertisement,
	elections []Election,
) {
	// check elections use available choices
	if len(ad.Choices) > 0 {
		for _, e := range elections {
			if !stringIsIn(e.VoteChoice, ad.Choices) {
				must.Errorf(ctx, "election %v is not an available choice", e.VoteChoice)
			}
		}
	}

	// TODO: check sufficient balance
}

func stringIsIn(s string, in []string) bool {
	for _, in := range in {
		if s == in {
			return true
		}
	}
	return false
}
