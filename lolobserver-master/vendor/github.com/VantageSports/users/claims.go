package users

import (
	"fmt"
	"time"

	jwt "github.com/dgrijalva/jwt-go"

	"github.com/VantageSports/common/constants/privileges"
)

func NewClaims(subject string, expiresIn time.Duration) *Claims {
	return &Claims{
		Sub: subject,
		Iat: float64(time.Now().Unix()),
		Exp: float64(time.Now().Add(expiresIn).Unix()),
	}
}

// AddPrivilege adds the specified privileges to the claims.
func (c *Claims) AddPrivilege(privs ...string) {
	if c.Privileges == nil {
		c.Privileges = map[string]bool{}
	}
	for _, p := range privs {
		c.Privileges[p] = true
	}
}

// HasPrivilege returns true iff all specified privileges are present.
func (c *Claims) HasPrivilege(privs ...string) bool {
	for _, priv := range privs {
		if c.Privileges == nil {
			return false
		}
		if !c.Privileges[priv] && !c.Privileges[privileges.VantageGod] {
			return false
		}
	}
	return true
}

// RequirePrivilege is like HasPrivilege, but returns an error if HasPrivilege
// would return false.
func (c *Claims) RequirePrivilege(privs ...string) error {
	for _, priv := range privs {
		if !c.HasPrivilege(priv) {
			return fmt.Errorf("privilege not found: %v", priv)
		}
	}
	return nil
}

// Valid is just a method that needs to be on claims to use
// jwt.NewWithClaims
func (c *Claims) Valid() error {
	sc := jwt.StandardClaims{
		Subject:   c.Sub,
		ExpiresAt: int64(c.Exp),
		IssuedAt:  int64(c.Iat),
	}
	return sc.Valid()
}
