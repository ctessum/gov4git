package boot

import (
	"context"

	"github.com/gov4git/gov4git/v2/proto"
	"github.com/gov4git/gov4git/v2/proto/account"
	"github.com/gov4git/gov4git/v2/proto/gov"
	"github.com/gov4git/gov4git/v2/proto/id"
	"github.com/gov4git/gov4git/v2/proto/member"
	"github.com/gov4git/gov4git/v2/proto/motion/motionpolicies/pmp_0"
	"github.com/gov4git/lib4git/form"
	"github.com/gov4git/lib4git/git"
)

func Boot(
	ctx context.Context,
	ownerAddr gov.OwnerAddress,
) git.Change[form.None, id.PrivateCredentials] {

	ownerCloned := gov.CloneOwner(ctx, ownerAddr)
	privChg := Boot_Local(ctx, ownerCloned)
	ownerCloned.Public.Push(ctx)
	ownerCloned.Private.Push(ctx)
	return privChg
}

func Boot_Local(
	ctx context.Context,
	ownerCloned gov.OwnerCloned,
) git.Change[form.None, id.PrivateCredentials] {

	// initialize project identity
	chg := id.Init_Local(ctx, ownerCloned.IDOwnerCloned())

	// create group everybody
	chg2 := member.SetGroup_StageOnly(ctx, ownerCloned.PublicClone(), member.Everybody)

	// create treasury accounts
	account.Boot_StageOnly(ctx, ownerCloned.PublicClone())

	// create PMP accounts
	pmp_0.Boot_StageOnly(ctx, ownerCloned.PublicClone())

	proto.Commit(ctx, ownerCloned.Public.Tree(), chg2)
	return chg
}
