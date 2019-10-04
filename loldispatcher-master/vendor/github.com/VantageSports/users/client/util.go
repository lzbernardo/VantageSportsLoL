package client

import (
	"fmt"

	"golang.org/x/net/context"
	"google.golang.org/grpc/metadata"

	"github.com/VantageSports/users"
)

// TokenContextKey is the context map key where the token string is stored.
const tokenContextKey = "vstoken"

type ErrAuthMissing struct {
	error
}

type ErrAuthInvalid struct {
	error
}

// ValidateCtxClaims is a utility function which extracts the jwt token from
// the provided context, validates the token with the provided AuthCheckClient,
// and optionally requires the specified permissions. Most services will call
// this immediately to verify credentials for protected actions.
func ValidateCtxClaims(ctx context.Context, checker users.AuthCheckClient, privs ...string) (*users.Claims, error) {
	md, found := metadata.FromContext(ctx)
	if !found || md == nil || md.Len() == 0 {
		return nil, ErrAuthMissing{fmt.Errorf("auth: no context metadata found")}
	}

	vals := md[tokenContextKey]
	if len(vals) == 0 || vals[0] == "" {
		return nil, ErrAuthMissing{fmt.Errorf("auth: no token in context")}
	}

	req := &users.TokenRequest{Token: vals[0]}
	res, err := checker.CheckToken(ctx, req)
	if err != nil {
		return nil, ErrAuthInvalid{err}
	}

	if len(privs) > 0 {
		if err = res.Claims.RequirePrivilege(privs...); err != nil {
			return nil, ErrAuthInvalid{err}
		}
	}

	return res.Claims, nil
}

func SetCtxToken(ctx context.Context, tokenString string) context.Context {
	md := metadata.Pairs(tokenContextKey, tokenString)
	return metadata.NewContext(ctx, md)
}
