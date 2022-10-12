package arb

import (
	"context"

	"github.com/gov4git/gov4git/lib/files"
	"github.com/gov4git/gov4git/lib/form"
	"github.com/gov4git/gov4git/lib/git"
	"github.com/gov4git/gov4git/proto"
	"github.com/gov4git/gov4git/proto/govproto"
	"github.com/gov4git/gov4git/proto/identityproto"
	"github.com/gov4git/gov4git/services/identity"
)

type VoteIn struct {
	BallotBranch string  `json:"ballot_branch"`
	BallotPath   string  `json:"ballot_path"`
	VoteChoice   string  `json:"vote_choice"`
	VoteStrength float64 `json:"vote_strength"`
}

type VoteOut struct {
	VoteRepo     string `json:"vote_repo"`
	VoteBranch   string `json:"vote_branch"`
	BallotRepo   string `json:"ballot_repo"`
	BallotBranch string `json:"ballot_branch"`
}

func (x GovArbService) Vote(ctx context.Context, in *VoteIn) (*VoteOut, error) {
	// find ballot advertisement in community repo

	// clone community repo at referendum branch locally
	community, err := git.MakeLocalInCtx(ctx, "community")
	if err != nil {
		return nil, err
	}
	if err := community.CloneBranch(ctx, x.GovConfig.CommunityURL, in.BallotBranch); err != nil {
		return nil, err
	}

	// find ballot ad
	findAd, err := x.FindBallotAdLocal(ctx, community,
		&FindBallotAdIn{BallotBranch: in.BallotBranch, BallotPath: in.BallotPath})
	if err != nil {
		return nil, err
	}

	// cast vote in voters public identity repo

	// clone voter identity public repo at the main identity branch
	voter, err := git.MakeLocalInCtx(ctx, "voter")
	if err != nil {
		return nil, err
	}
	if err := voter.CloneBranch(ctx, x.IdentityConfig.PublicURL, proto.IdentityBranch); err != nil {
		return nil, err
	}

	// retrieve the voter's private keys from the private identity repo
	idService := identity.IdentityService{IdentityConfig: x.IdentityConfig}
	voterCredentials, err := idService.GetPrivateCredentials(ctx, &identity.GetPrivateCredentialsIn{})
	if err != nil {
		return nil, err
	}

	// compute the name of the vote branch
	voteBranch, err := govproto.BallotVoteBranch(ctx, findAd.BallotAdBytes)
	if err != nil {
		return nil, err
	}

	// checkout an existing voting branch or create an orphan one
	if err := voter.CheckoutBranch(ctx, voteBranch); err != nil {
		if err := voter.CheckoutNewOrphan(ctx, voteBranch); err != nil {
			return nil, err
		}
		if err := voter.ResetHard(ctx); err != nil {
			return nil, err
		}
	}

	// add vote to vote branch
	vote := govproto.GovBallotVote{
		BallotAd: findAd.BallotAd,
		Choice:   in.VoteChoice,
		Strength: in.VoteStrength,
	}
	// sign vote
	voteData, err := form.EncodeForm(ctx, vote)
	if err != nil {
		return nil, err
	}
	signature, err := identityproto.SignPlaintext(ctx, &voterCredentials.PrivateCredentials, voteData)
	if err != nil {
		return nil, err
	}
	signatureData, err := form.EncodeForm(ctx, signature)
	if err != nil {
		return nil, err
	}

	// write vote and signature
	stage := files.ByteFiles{
		files.ByteFile{Path: govproto.GovBallotVoteFilepath, Bytes: voteData},
		files.ByteFile{Path: govproto.GovBallotVoteSignatureFilepath, Bytes: signatureData},
	}
	if err := voter.Dir().WriteByteFiles(stage); err != nil {
		return nil, err
	}
	if err := voter.Add(ctx, stage.Paths()); err != nil {
		return nil, err
	}
	msg := govproto.BallotVoteCommitHeader(x.GovConfig.CommunityURL, in.BallotBranch, in.BallotPath)
	if err := voter.Commit(ctx, msg); err != nil {
		return nil, err
	}

	// push identity repo to origin
	if err := voter.PushUpstream(ctx); err != nil {
		return nil, err
	}

	return &VoteOut{
		VoteRepo:     x.IdentityConfig.PublicURL,
		VoteBranch:   voteBranch,
		BallotRepo:   x.GovConfig.CommunityURL,
		BallotBranch: in.BallotBranch,
	}, nil
}
