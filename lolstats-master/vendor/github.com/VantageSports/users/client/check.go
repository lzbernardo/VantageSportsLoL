package client

import (
	"time"

	"github.com/hashicorp/golang-lru"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/VantageSports/common/certs"
	"github.com/VantageSports/common/constants/privileges"
	"github.com/VantageSports/users"
)

// DialAuthClient is a convenience function for generating an AuthCheckClient
// that is optionally cached and internal-auth-enabled. Almost every RPC service
// needs to check tokens, so this is just provided for DRY-ness.
func DialAuthCheck(addr, tlsCertPath, internKey string, cacheSize int64, insecure bool) (users.AuthCheckClient, error) {
	c, err := certs.ClientTLS(tlsCertPath, certs.Insecure(insecure))
	if err != nil {
		return nil, err
	}

	conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(credentials.NewTLS(c)))
	if err != nil {
		return nil, err
	}

	authClient := users.NewAuthCheckClient(conn)

	if internKey != "" {
		authClient = NewInternalAuth(authClient, internKey)
	}

	if cacheSize > 0 {
		if authClient, err = NewClaimsCache(authClient, 100); err != nil {
			return nil, err
		}
	}

	return authClient, nil
}

// claimsCache is an AuthCheck client wrapper that caches the claims of valid
// tokens, preventing the need to consult the auth server as frequently.
type claimsCache struct {
	next  users.AuthCheckClient
	cache *lru.Cache
}

// NewClaimsCache returns a new claimsCache client wrapping the specified AuthCheckClient.
func NewClaimsCache(next users.AuthCheckClient, cacheSize int) (*claimsCache, error) {
	cache, err := lru.New(cacheSize)
	if err != nil {
		return nil, err
	}

	return &claimsCache{
		next:  next,
		cache: cache,
	}, nil
}

// CheckToken first consults the local LRU cache for the cached claims. If those
// claims exist, and are not expired, they are returned before sending the
// request.
func (a *claimsCache) CheckToken(ctx context.Context, in *users.TokenRequest, opts ...grpc.CallOption) (*users.ClaimsResponse, error) {
	if claimsV, found := a.cache.Get(in.Token); found {
		claims := claimsV.(*users.Claims)
		if claims.Exp > float64(time.Now().Unix()) {
			return &users.ClaimsResponse{Claims: claims}, nil
		}
	}

	res, err := a.next.CheckToken(ctx, in, opts...)
	if err == nil && res.Claims != nil {
		a.cache.Add(in.Token, res.Claims)
	}

	return res, err
}

// internalAuth allows for inter-service communication without needing to
// provide each service with its own username/password. It uses a shared-key
// to authorize internal requests.
type internalAuth struct {
	next      users.AuthCheckClient
	secretKey string
}

// NewInternalAuth returns an internalAuth client wrapping the specified AuthCheckClient.
func NewInternalAuth(next users.AuthCheckClient, secretKey string) *internalAuth {
	return &internalAuth{
		next:      next,
		secretKey: secretKey,
	}
}

// CheckToken checks whether the token matches the secret key, and returns "god"
// privileges if so, otherwise forwarding the request on to the next handler.
func (a *internalAuth) CheckToken(ctx context.Context, in *users.TokenRequest, opts ...grpc.CallOption) (*users.ClaimsResponse, error) {
	if in.Token == a.secretKey {
		out := &users.ClaimsResponse{}
		out.Claims = users.NewClaims("internal", time.Hour*24)
		out.Claims.AddPrivilege(privileges.VantageGod)
		return out, nil
	}

	return a.next.CheckToken(ctx, in, opts...)
}
