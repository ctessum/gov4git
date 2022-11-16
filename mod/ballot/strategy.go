package ballot

import (
	"context"

	"github.com/gov4git/gov4git/mod/id"
	"github.com/gov4git/lib4git/git"
)

type Strategy interface {
	StrategyName() string
	Tally(
		ctx context.Context,
		govRepo id.OwnerRepo,
		govTree id.OwnerTree,
		//
		ad *Advertisement,
		current *TallyForm,
		fetched []FetchedVote,
	) git.Change[TallyForm] // tallying can change other aspects of the repo, like user balances
}
