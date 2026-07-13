package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
)

type bearerRuntime struct {
	issuer         string
	keySet         oidc.KeySet
	allowedClients map[string]struct{}
}

var (
	bearerRuntimeInst *bearerRuntime
	bearerRuntimeErr  error
	bearerRuntimeOnce sync.Once
)

func (s *Service) apiBearerEnabled() bool {
	return s.env.PBW_OIDC_ENABLED && len(s.apiAllowedClients()) > 0
}

func (s *Service) apiAllowedClients() map[string]struct{} {
	raw := strings.TrimSpace(s.env.PBW_API_OIDC_CLIENT_IDS)
	if raw == "" {
		return nil
	}

	allowed := map[string]struct{}{}
	for _, part := range strings.Split(raw, ",") {
		clientID := strings.TrimSpace(part)
		if clientID != "" {
			allowed[clientID] = struct{}{}
		}
	}
	return allowed
}

func (s *Service) bearerRuntime(ctx context.Context) (*bearerRuntime, error) {
	bearerRuntimeOnce.Do(func() {
		issuer := strings.TrimSuffix(s.env.PBW_OIDC_ISSUER, "/")
		if issuer == "" {
			bearerRuntimeErr = fmt.Errorf("PBW_OIDC_ISSUER is empty")
			return
		}

		allowed := s.apiAllowedClients()
		if len(allowed) == 0 {
			bearerRuntimeErr = fmt.Errorf("PBW_API_OIDC_CLIENT_IDS is empty")
			return
		}

		bearerRuntimeInst = &bearerRuntime{
			issuer:         issuer,
			keySet:         oidc.NewRemoteKeySet(context.Background(), issuer+"/protocol/openid-connect/certs"),
			allowedClients: allowed,
		}
	})

	return bearerRuntimeInst, bearerRuntimeErr
}

// GetUserFromBearerToken validates a Keycloak access token (JWT) from Authorization header.
func (s *Service) GetUserFromBearerToken(
	ctx context.Context, authHeader string,
) (bool, OidcClaims, error) {
	if !s.apiBearerEnabled() {
		return false, OidcClaims{}, nil
	}

	token := bearerTokenFromHeader(authHeader)
	if token == "" {
		return false, OidcClaims{}, nil
	}

	runtime, err := s.bearerRuntime(ctx)
	if err != nil {
		return false, OidcClaims{}, err
	}

	payload, err := runtime.keySet.VerifySignature(ctx, token)
	if err != nil {
		return false, OidcClaims{}, fmt.Errorf("verify bearer token: %w", err)
	}

	var rawClaims map[string]any
	if err := json.Unmarshal(payload, &rawClaims); err != nil {
		return false, OidcClaims{}, fmt.Errorf("parse bearer token claims: %w", err)
	}

	if err := validateBearerClaims(rawClaims, runtime); err != nil {
		return false, OidcClaims{}, err
	}

	return true, claimsFromMap(rawClaims), nil
}

func bearerTokenFromHeader(authHeader string) string {
	authHeader = strings.TrimSpace(authHeader)
	if authHeader == "" {
		return ""
	}

	const prefix = "Bearer "
	if !strings.HasPrefix(authHeader, prefix) {
		return ""
	}

	return strings.TrimSpace(strings.TrimPrefix(authHeader, prefix))
}

// bearerClockSkew tolerates minor clock drift between Keycloak and this host.
const bearerClockSkew = 30 * time.Second

func validateBearerClaims(rawClaims map[string]any, runtime *bearerRuntime) error {
	iss, _ := rawClaims["iss"].(string)
	iss = strings.TrimSuffix(iss, "/")
	if iss != runtime.issuer {
		return fmt.Errorf("invalid token issuer")
	}

	// Keycloak sets typ=Bearer for access tokens; reject refresh/logout tokens
	// signed with the same keys.
	if typ, _ := rawClaims["typ"].(string); typ != "" && !strings.EqualFold(typ, "Bearer") {
		return fmt.Errorf("token type not allowed: %s", typ)
	}

	now := time.Now()

	exp, _ := rawClaims["exp"].(float64)
	if exp == 0 {
		return fmt.Errorf("token has no exp claim")
	}
	if now.Add(-bearerClockSkew).Unix() > int64(exp) {
		return fmt.Errorf("token is expired")
	}

	if nbf, _ := rawClaims["nbf"].(float64); nbf != 0 {
		if now.Add(bearerClockSkew).Unix() < int64(nbf) {
			return fmt.Errorf("token is not valid yet")
		}
	}

	azp, _ := rawClaims["azp"].(string)
	if azp == "" {
		clientID, _ := rawClaims["client_id"].(string)
		azp = clientID
	}
	if azp == "" {
		return fmt.Errorf("token has no azp/client_id")
	}
	if _, ok := runtime.allowedClients[azp]; !ok {
		return fmt.Errorf("client not allowed for api: %s", azp)
	}

	return nil
}

func claimsFromMap(rawClaims map[string]any) OidcClaims {
	email, _ := rawClaims["email"].(string)
	name, _ := rawClaims["name"].(string)
	preferred, _ := rawClaims["preferred_username"].(string)
	verified, _ := rawClaims["email_verified"].(bool)

	return OidcClaims{
		Email:         email,
		EmailVerified: verified,
		Name:          name,
		PreferredUser: preferred,
		Groups:        GroupsFromClaims(rawClaims),
	}
}
