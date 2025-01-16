package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/bluesky-social/indigo/atproto/crypto"
	"github.com/bluesky-social/indigo/atproto/identity"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/golang-jwt/jwt/v5"
	"github.com/willdot/bskyfeedgen/frontend"
)

// The contents of this file have been borrowed from here: https://github.com/orthanc/bluesky-go-feeds/blob/f719f113f1afc9080e50b4b1f5ca239aa3073c79/web/auth.go#L20-L46
// It essentially allows the signing method that atproto uses for JWT to be used when verifying the JWT that they send in requests

const (
	ES256K = "ES256K"
	ES256  = "ES256"
)

type AtProtoSigningMethod struct {
	alg string
}

func (m *AtProtoSigningMethod) Alg() string {
	return m.alg
}

func (m *AtProtoSigningMethod) Verify(signingString string, signature []byte, key interface{}) error {
	err := key.(crypto.PublicKey).HashAndVerifyLenient([]byte(signingString), signature)
	return err
}

func (m *AtProtoSigningMethod) Sign(signingString string, key interface{}) ([]byte, error) {
	return nil, fmt.Errorf("unimplemented")
}

func init() {
	ES256K := AtProtoSigningMethod{alg: "ES256K"}
	jwt.RegisterSigningMethod(ES256K.Alg(), func() jwt.SigningMethod {
		return &ES256K
	})

	ES256 := AtProtoSigningMethod{alg: "ES256"}
	jwt.RegisterSigningMethod(ES256.Alg(), func() jwt.SigningMethod {
		return &ES256
	})

}

var directory = identity.DefaultDirectory()

func getRequestUserDID(r *http.Request) (string, error) {
	headerValues := r.Header["Authorization"]

	if len(headerValues) != 1 {
		return "", fmt.Errorf("missing authorization header")
	}
	token := strings.TrimSpace(strings.Replace(headerValues[0], "Bearer ", "", 1))

	keyfunc := func(token *jwt.Token) (interface{}, error) {
		did := syntax.DID(token.Claims.(jwt.MapClaims)["iss"].(string))
		identity, err := directory.LookupDID(r.Context(), did)
		if err != nil {
			return nil, fmt.Errorf("unable to resolve did %s: %s", did, err)
		}
		key, err := identity.PublicKey()
		if err != nil {
			return nil, fmt.Errorf("signing key not found for did %s: %s", did, err)
		}
		return key, nil
	}

	validMethods := jwt.WithValidMethods([]string{ES256, ES256K})

	parsedToken, err := jwt.ParseWithClaims(token, jwt.MapClaims{}, keyfunc, validMethods)
	if err != nil {
		return "", fmt.Errorf("invalid token: %s", err)
	}

	claims, ok := parsedToken.Claims.(jwt.MapClaims)
	if !ok {
		return "", fmt.Errorf("token contained no claims")
	}

	issVal, ok := claims["iss"].(string)
	if !ok {
		return "", fmt.Errorf("iss claim missing")
	}

	return string(syntax.DID(issVal)), nil
}

const (
	jwtCookieName = "JWT"
	didCookieName = "DID"
)

func (s *Server) authMiddleware(next func(http.ResponseWriter, *http.Request)) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		jwtCookie, err := r.Cookie(jwtCookieName)
		if err != nil {
			slog.Error("read JWT cookie", "error", err)
			frontend.Login("", "").Render(r.Context(), w)
			return
		}
		if jwtCookie == nil {
			slog.Error("missing JWT cookie")
			frontend.Login("", "").Render(r.Context(), w)
			return
		}

		didCookie, err := r.Cookie(didCookieName)
		if err != nil {
			slog.Error("read DID cookie", "error", err)
			frontend.Login("", "").Render(r.Context(), w)
			return
		}
		if didCookie == nil {
			slog.Error("missing DID cookie")
			frontend.Login("", "").Render(r.Context(), w)
			return
		}

		claims := jwt.MapClaims{}
		_, _, err = jwt.NewParser().ParseUnverified(jwtCookie.Value, &claims)
		if err != nil {
			slog.Error("parsing JWT", "error", err)
			frontend.Login("", "").Render(r.Context(), w)
			return
		}

		if expiry, ok := claims["exp"].(string); ok {
			expiryInt, err := strconv.Atoi(expiry)
			if err != nil {
				slog.Error("invalid claims from token", "error", err)
				frontend.Login("", "").Render(r.Context(), w)
				return
			}

			if time.Now().Unix() > int64(expiryInt) {
				frontend.Login("", "").Render(r.Context(), w)
				return
			}
		}
		next(w, r)
	}
}
