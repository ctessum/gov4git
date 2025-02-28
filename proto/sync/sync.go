package sync

import (
	"context"

	"github.com/gov4git/gov4git/v2/proto/ballot/ballotapi"
	"github.com/gov4git/gov4git/v2/proto/bureau"
	"github.com/gov4git/gov4git/v2/proto/gov"
	"github.com/gov4git/gov4git/v2/proto/member"
	"github.com/gov4git/lib4git/form"
	"github.com/gov4git/lib4git/git"
)

func Sync(
	ctx context.Context,
	govAddr gov.OwnerAddress,
	maxPar int,
) git.Change[form.Map, form.Map] {

	// collect votes and tally all open ballots
	tallyChg := ballotapi.TallyAll(ctx, govAddr, maxPar)

	// process bureau requests by users
	bureauChg := bureau.Process(ctx, govAddr, member.Everybody)

	return git.NewChange(
		"Governance-community sync",
		"sync_sync",
		form.Map{},
		form.Map{
			"tally_result":  tallyChg.Result,
			"bureau_result": bureauChg.Result,
		},
		form.Forms{tallyChg, bureauChg},
	)
}
