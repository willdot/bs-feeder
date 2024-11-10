package main

import (
	"fmt"

	"github.com/bluesky-social/indigo/atproto/crypto"
	"github.com/golang-jwt/jwt/v5"
)

// The contents of this file have been borrowed from here: https://github.com/orthanc/bluesky-go-feeds/blob/f719f113f1afc9080e50b4b1f5ca239aa3073c79/web/auth.go#L20-L46
// It essentially allows the signing method that atproto uses for JWT to be used when verifying the JWT that they send in requests

type AtProtoSigningMethod struct {
	alg string
}

func (m *AtProtoSigningMethod) Alg() string {
	return m.alg
}

func (m *AtProtoSigningMethod) Verify(signingString string, signature []byte, key interface{}) error {
	return key.(crypto.PublicKey).HashAndVerifyLenient([]byte(signingString), signature)
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
