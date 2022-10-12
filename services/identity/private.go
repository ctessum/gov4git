package identity

import (
	"context"

	"github.com/gov4git/gov4git/lib/git"
	"github.com/gov4git/gov4git/proto"
	"github.com/gov4git/gov4git/proto/identityproto"
)

type GetPrivateCredentialsIn struct{}

type GetPrivateCredentialsOut struct {
	PrivateCredentials identityproto.PrivateCredentials `json:"private_credentials"`
}

func (x IdentityService) GetPrivateCredentials(ctx context.Context, in *GetPrivateCredentialsIn) (*GetPrivateCredentialsOut, error) {
	// clone private identity repo locally
	private, err := git.MakeLocalInCtx(ctx, "private")
	if err != nil {
		return nil, err
	}
	if err := private.CloneBranch(ctx, x.IdentityConfig.PrivateURL, proto.IdentityBranch); err != nil {
		return nil, err
	}

	// read from the local clone
	out, err := x.GetPrivateCredentialsLocal(ctx, private, in)
	if err != nil {
		return nil, err
	}

	return out, nil
}

func (x IdentityService) GetPrivateCredentialsLocal(ctx context.Context, private git.Local, in *GetPrivateCredentialsIn) (*GetPrivateCredentialsOut, error) {
	var credentials identityproto.PrivateCredentials
	if _, err := private.Dir().ReadFormFile(ctx, identityproto.PrivateCredentialsPath, &credentials); err != nil {
		return nil, err
	}
	return &GetPrivateCredentialsOut{PrivateCredentials: credentials}, nil
}
