package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/database/dbgen"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/util/pathutil"
	"golang.org/x/oauth2"
)

const oidcStateCookieName = "pbw_oidc_state"

type OidcClaims struct {
	Email         string   `json:"email"`
	EmailVerified bool     `json:"email_verified"`
	Name          string   `json:"name"`
	PreferredUser string   `json:"preferred_username"`
	Groups        []string `json:"groups,omitempty"`
}

type oidcRuntime struct {
	provider *oidc.Provider
	oauth2   oauth2.Config
}

var (
	oidcRuntimeInst *oidcRuntime
	oidcRuntimeErr  error
	oidcRuntimeOnce sync.Once
)

func (s *Service) OidcEnabled() bool {
	return s.env.PBW_OIDC_ENABLED
}

func (s *Service) oidcRedirectURL() string {
	baseURL := strings.TrimSuffix(s.env.PBW_PUBLIC_URL, "/")
	return baseURL + pathutil.BuildPath("/auth/oidc/callback")
}

func (s *Service) oidcRuntime(ctx context.Context) (*oidcRuntime, error) {
	oidcRuntimeOnce.Do(func() {
		issuer := strings.TrimSuffix(s.env.PBW_OIDC_ISSUER, "/")
		provider, err := oidc.NewProvider(context.Background(), issuer)
		if err != nil {
			oidcRuntimeErr = fmt.Errorf("init oidc provider: %w", err)
			return
		}

		oidcRuntimeInst = &oidcRuntime{
			provider: provider,
			oauth2: oauth2.Config{
				ClientID:     s.env.PBW_OIDC_CLIENT_ID,
				ClientSecret: s.env.PBW_OIDC_CLIENT_SECRET,
				RedirectURL:  s.oidcRedirectURL(),
				Endpoint:     provider.Endpoint(),
				Scopes:       []string{oidc.ScopeOpenID, "profile", "email", "group"},
			},
		}
	})

	return oidcRuntimeInst, oidcRuntimeErr
}

func (s *Service) NewOidcState() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func (s *Service) OidcAuthCodeURL(state string) (string, error) {
	runtime, err := s.oidcRuntime(context.Background())
	if err != nil {
		return "", err
	}
	return runtime.oauth2.AuthCodeURL(state, oauth2.AccessTypeOffline), nil
}

func (s *Service) LoginWithOidc(
	ctx context.Context, code, ip, userAgent string,
) (dbgen.AuthServiceCreateSessionRow, error) {
	runtime, err := s.oidcRuntime(ctx)
	if err != nil {
		return dbgen.AuthServiceCreateSessionRow{}, err
	}

	token, err := runtime.oauth2.Exchange(ctx, code)
	if err != nil {
		return dbgen.AuthServiceCreateSessionRow{}, fmt.Errorf("exchange oidc code: %w", err)
	}

	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok || rawIDToken == "" {
		return dbgen.AuthServiceCreateSessionRow{}, errors.New("oidc response has no id_token")
	}

	verifier := runtime.provider.Verifier(&oidc.Config{ClientID: s.env.PBW_OIDC_CLIENT_ID})
	idToken, err := verifier.Verify(ctx, rawIDToken)
	if err != nil {
		return dbgen.AuthServiceCreateSessionRow{}, fmt.Errorf("verify id_token: %w", err)
	}

	var claims OidcClaims
	var rawClaims map[string]any
	if err := idToken.Claims(&rawClaims); err != nil {
		return dbgen.AuthServiceCreateSessionRow{}, fmt.Errorf("parse oidc claims: %w", err)
	}
	if err := idToken.Claims(&claims); err != nil {
		return dbgen.AuthServiceCreateSessionRow{}, fmt.Errorf("parse oidc claims: %w", err)
	}

	groups := GroupsFromClaims(rawClaims)
	if len(groups) == 0 && token.AccessToken != "" {
		accessVerifier := runtime.provider.Verifier(&oidc.Config{SkipClientIDCheck: true})
		if verifiedAccess, err := accessVerifier.Verify(ctx, token.AccessToken); err == nil {
			var rawAccessClaims map[string]any
			if err := verifiedAccess.Claims(&rawAccessClaims); err == nil {
				groups = GroupsFromClaims(rawAccessClaims)
			}
		}
	}
	claims.Groups = groups

	email := strings.ToLower(strings.TrimSpace(claims.Email))
	if email == "" {
		return dbgen.AuthServiceCreateSessionRow{}, errors.New("oidc token has no email claim")
	}

	name := strings.TrimSpace(claims.Name)
	if name == "" {
		name = strings.TrimSpace(claims.PreferredUser)
	}
	if name == "" {
		name = email
	}

	user, err := s.dbgen.AuthServiceLoginGetUserByEmail(ctx, email)
	if err != nil {
		user, err = s.dbgen.UsersServiceCreateSsoUser(ctx, dbgen.UsersServiceCreateSsoUserParams{
			Name:  name,
			Email: email,
		})
		if err != nil {
			return dbgen.AuthServiceCreateSessionRow{}, fmt.Errorf("create sso user: %w", err)
		}
	}

	return s.CreateSession(ctx, user.ID, ip, userAgent, groups, false)
}

func OidcStateCookieName() string {
	return oidcStateCookieName
}
