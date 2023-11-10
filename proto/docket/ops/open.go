package ops

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/gov4git/gov4git/proto"
	"github.com/gov4git/gov4git/proto/docket/policy"
	"github.com/gov4git/gov4git/proto/docket/schema"
	"github.com/gov4git/gov4git/proto/gov"
	"github.com/gov4git/lib4git/git"
	"github.com/gov4git/lib4git/must"
)

func OpenMotion(
	ctx context.Context,
	addr gov.GovOwnerAddress,
	id schema.MotionID,
	policy schema.PolicyName,
	title string,
	desc string,
	typ schema.MotionType,
	trackerURL string,
	labels []string,

) git.ChangeNoResult {

	cloned := gov.CloneOwner(ctx, addr)
	chg := OpenMotion_StageOnly(ctx, addr, cloned, id, policy, title, desc, typ, trackerURL, labels)
	return proto.CommitIfChanged(ctx, cloned.Public, chg)
}

func OpenMotion_StageOnly(
	ctx context.Context,
	addr gov.GovOwnerAddress,
	cloned gov.GovOwnerCloned,
	id schema.MotionID,
	policyName schema.PolicyName,
	title string,
	desc string,
	typ schema.MotionType,
	trackerURL string,
	labels []string,

) git.ChangeNoResult {

	t := cloned.Public.Tree()
	labels = slices.Clone(labels)
	slices.Sort(labels)

	must.Assert(ctx, !IsMotion_Local(ctx, t, id), schema.ErrMotionAlreadyExists)
	motion := schema.Motion{
		OpenedAt:   time.Now(),
		ID:         id,
		Type:       typ,
		Policy:     policyName,
		TrackerURL: trackerURL,
		Title:      title,
		Body:       desc,
		Labels:     labels,
		Closed:     false,
	}
	schema.MotionKV.Set(ctx, schema.MotionNS, t, id, motion)

	// apply policy
	pcy := policy.Get(ctx, policyName.String())
	pcy.Open(
		ctx,
		addr,
		cloned,
		motion,
		policy.MotionPolicyNS(id),
	)

	return git.NewChangeNoResult(fmt.Sprintf("Open motion %v", id), "docket_open_motion")
}
