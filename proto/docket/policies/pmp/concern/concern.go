package concern

import (
	"context"
	"fmt"

	"github.com/gov4git/gov4git/proto/account"
	"github.com/gov4git/gov4git/proto/ballot/ballot"
	"github.com/gov4git/gov4git/proto/ballot/common"
	"github.com/gov4git/gov4git/proto/ballot/load"
	"github.com/gov4git/gov4git/proto/docket/ops"
	"github.com/gov4git/gov4git/proto/docket/policies/pmp"
	"github.com/gov4git/gov4git/proto/docket/policy"
	"github.com/gov4git/gov4git/proto/docket/schema"
	"github.com/gov4git/gov4git/proto/gov"
	"github.com/gov4git/gov4git/proto/member"
	"github.com/gov4git/gov4git/proto/notice"
	"github.com/gov4git/lib4git/form"
	"github.com/gov4git/lib4git/must"
	"github.com/gov4git/lib4git/ns"
)

func init() {
	policy.Install(context.Background(), ConcernPolicyName, concernPolicy{})
}

const ConcernPolicyName = schema.PolicyName("pmp-concern-policy")

type concernPolicy struct{}

func (x concernPolicy) Open(
	ctx context.Context,
	cloned gov.OwnerCloned,
	motion schema.Motion,
	policyNS ns.NS,
	args ...any,

) (policy.Report, notice.Notices) {

	// initialize state
	state := NewConcernState(motion.ID)
	SaveState_StageOnly(ctx, cloned.Public.Tree(), policyNS, state)

	// open a poll for the motion
	ballot.Open_StageOnly(
		ctx,
		load.QVStrategyName,
		cloned,
		state.PriorityPoll,
		fmt.Sprintf("Prioritization poll for motion %v", motion.ID),
		fmt.Sprintf("Up/down vote the priority for concern (issue) %v", motion.ID),
		[]string{pmp.ConcernBallotChoice},
		member.Everybody,
	)

	return nil, notice.Noticef("Started managing this issue as Gov4Git concern `%v`.", motion.ID)
}

func (x concernPolicy) Score(
	ctx context.Context,
	cloned gov.OwnerCloned,
	motion schema.Motion,
	policyNS ns.NS,
	args ...any,

) (schema.Score, notice.Notices) {

	state := LoadState_Local(ctx, cloned.Public.Tree(), policyNS)

	// compute motion score from the priority poll ballot
	ads := ballot.Show_Local(ctx, cloned.Public.Tree(), state.PriorityPoll)
	attention := ads.Tally.Attention()

	return schema.Score{
		Attention: attention,
	}, notice.Noticef("Updated prioritization tally to %v.", ads.Tally.Scores[pmp.ConcernBallotChoice])
}

func (x concernPolicy) Update(
	ctx context.Context,
	cloned gov.OwnerCloned,
	motion schema.Motion,
	policyNS ns.NS,
	args ...any,

) (policy.Report, notice.Notices) {

	// update eligible proposals

	state := LoadState_Local(ctx, cloned.Public.Tree(), policyNS)

	eligible := schema.Refs{}
	for ref := range state.EligibleProposals.RefSet() {
		if IsProposalEligible(ctx, cloned.PublicClone(), ref.From) {
			eligible = append(eligible, ref)
		}
	}
	eligible.Sort()
	state.EligibleProposals = eligible

	state.PriorityPoll = pmp.ConcernPollBallotName(motion.ID)

	SaveState_StageOnly(ctx, cloned.Public.Tree(), policyNS, state)

	return x.updateFreeze(ctx, cloned, motion, policyNS)
}

func (x concernPolicy) Close(
	ctx context.Context,
	cloned gov.OwnerCloned,
	motion schema.Motion,
	policyNS ns.NS,
	args ...any,
	// args[0]=toID account.AccountID
	// args[1]=prop schema.Motion

) (policy.Report, notice.Notices) {

	must.Assertf(ctx, len(args) == 2, "issue closure requires two arguments, got %v", args)
	toID, ok := args[0].(account.AccountID)
	must.Assertf(ctx, ok, "unrecognized account ID argument %v", args[0])
	prop, ok := args[1].(schema.Motion)
	must.Assertf(ctx, ok, "unrecognized proposal motion argument %v", args[1])

	// close the poll for the motion
	priorityPollName := pmp.ConcernPollBallotName(motion.ID)
	chg := ballot.Close_StageOnly(
		ctx,
		cloned,
		priorityPollName,
		toID,
	)

	return &CloseReport{}, closeNotice(ctx, motion, chg.Result, prop)
}

func (x concernPolicy) Cancel(
	ctx context.Context,
	cloned gov.OwnerCloned,
	motion schema.Motion,
	policyNS ns.NS,
	args ...any,

) (policy.Report, notice.Notices) {

	// cancel the poll for the motion (returning credits to users)
	priorityPollName := pmp.ConcernPollBallotName(motion.ID)
	chg := ballot.Cancel_StageOnly(
		ctx,
		cloned,
		priorityPollName,
	)

	return &CancelReport{
		PriorityPollOutcome: chg.Result,
	}, cancelNotice(ctx, motion, chg.Result)
}

type PolicyView struct {
	State        *ConcernState  `json:"state"`
	PriorityPoll common.AdTally `json:"priority_poll"`
}

func (x concernPolicy) Show(
	ctx context.Context,
	cloned gov.Cloned,
	motion schema.Motion,
	policyNS ns.NS,
	args ...any,

) form.Form {

	// retrieve policy state
	policyState := LoadState_Local(ctx, cloned.Tree(), policyNS)

	// retrieve poll state
	priorityPollName := pmp.ConcernPollBallotName(motion.ID)
	pollState := ballot.Show_Local(ctx, cloned.Tree(), priorityPollName)

	return PolicyView{
		State:        policyState,
		PriorityPoll: pollState,
	}
}

func (x concernPolicy) AddRefTo(
	ctx context.Context,
	cloned gov.OwnerCloned,
	refType schema.RefType,
	from schema.Motion,
	to schema.Motion,
	fromPolicyNS ns.NS,
	toPolicyNS ns.NS,
	args ...any,

) (policy.Report, notice.Notices) {

	return nil, nil
}

func (x concernPolicy) AddRefFrom(
	ctx context.Context,
	cloned gov.OwnerCloned,
	refType schema.RefType,
	from schema.Motion,
	to schema.Motion,
	fromPolicyNS ns.NS,
	toPolicyNS ns.NS,
	args ...any,

) (policy.Report, notice.Notices) {

	if refType != pmp.ResolvesRefType {
		return nil, nil
	}

	if !IsProposalEligible(ctx, cloned.PublicClone(), from.ID) {
		return nil, nil
	}

	toState := LoadState_Local(ctx, cloned.Public.Tree(), toPolicyNS)
	ref := schema.Ref{Type: refType, From: from.ID, To: to.ID}

	if toState.EligibleProposals.Contains(ref) {
		return nil, nil
	}

	toState.EligibleProposals = append(toState.EligibleProposals, ref)
	SaveState_StageOnly(ctx, cloned.Public.Tree(), toPolicyNS, toState)

	return nil, notice.Noticef("This issue has been referenced by an eligible PR, managed as Gov4Git proposal `%v`.", from.ID)
}

func (x concernPolicy) RemoveRefTo(
	ctx context.Context,
	cloned gov.OwnerCloned,
	refType schema.RefType,
	from schema.Motion,
	to schema.Motion,
	fromPolicyNS ns.NS,
	toPolicyNS ns.NS,
	args ...any,

) (policy.Report, notice.Notices) {

	return nil, nil
}

func (x concernPolicy) RemoveRefFrom(
	ctx context.Context,
	cloned gov.OwnerCloned,
	refType schema.RefType,
	from schema.Motion,
	to schema.Motion,
	fromPolicyNS ns.NS,
	toPolicyNS ns.NS,
	args ...any,

) (policy.Report, notice.Notices) {

	toState := LoadState_Local(ctx, cloned.Public.Tree(), toPolicyNS)
	ref := schema.Ref{Type: refType, From: from.ID, To: to.ID}

	if !toState.EligibleProposals.Contains(ref) {
		return nil, nil
	}

	toState.EligibleProposals = toState.EligibleProposals.Remove(ref)
	SaveState_StageOnly(ctx, cloned.Public.Tree(), toPolicyNS, toState)

	return nil, notice.Noticef("This issue is no longer referenced by the PR, managed as Gov4Git proposal `%v`.", from.ID)
}

func (x concernPolicy) updateFreeze(
	ctx context.Context,
	cloned gov.OwnerCloned,
	motion schema.Motion,
	policyNS ns.NS,
	args ...any,

) (policy.Report, notice.Notices) {

	toState := LoadState_Local(ctx, cloned.Public.Tree(), policyNS)

	notices := notice.Notices{}
	if toState.EligibleProposals.Len() > 0 && !motion.Frozen {
		ops.FreezeMotion_StageOnly(ctx, cloned, motion.ID)
		notices = append(notices, notice.Noticef("Freezing ❄️ issue as there are eligible PRs addressing it.")...)
	}
	if toState.EligibleProposals.Len() == 0 && motion.Frozen {
		ops.UnfreezeMotion_StageOnly(ctx, cloned, motion.ID)
		notices = append(notices, notice.Noticef("Unfreezing 🌤️ issue as there are no eligible PRs are addressing it.")...)
	}

	return nil, notices
}

func (x concernPolicy) Freeze(
	ctx context.Context,
	cloned gov.OwnerCloned,
	motion schema.Motion,
	policyNS ns.NS,
	args ...any,

) (policy.Report, notice.Notices) {

	// freeze priority poll, if not already frozen
	priorityPoll := pmp.ConcernPollBallotName(motion.ID)
	if ballot.IsFrozen_Local(ctx, cloned.PublicClone(), priorityPoll) {
		return nil, nil
	}
	ballot.Freeze_StageOnly(ctx, cloned, priorityPoll)

	return nil, notice.Noticef("This issue, managed by Gov4Git concern `%v`, has been frozen ❄️", motion.ID)
}

func (x concernPolicy) Unfreeze(
	ctx context.Context,
	cloned gov.OwnerCloned,
	motion schema.Motion,
	policyNS ns.NS,
	args ...any,

) (policy.Report, notice.Notices) {

	// unfreeze the priority poll ballot, if frozen
	priorityPoll := pmp.ConcernPollBallotName(motion.ID)
	if !ballot.IsFrozen_Local(ctx, cloned.PublicClone(), priorityPoll) {
		return nil, nil
	}
	ballot.Unfreeze_StageOnly(ctx, cloned, priorityPoll)

	return nil, notice.Noticef("This issue, managed by Gov4Git concern `%v`, has been unfrozen 🌤️", motion.ID)
}

// motion.Un/Freeze --calls--> policy Un/Freeze --calls--> ballot Un/Freeze
