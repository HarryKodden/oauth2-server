// Command dpop-proof generates an ES256 DPoP proof JWT for integration tests.
//
//	go run ./scripts/dpop-proof -htm POST -htu https://as.example/token
//	go run ./scripts/dpop-proof -htm POST -htu https://as.example/token -nonce <value> -key /tmp/dpop.key
//
// Prints two lines: proof JWT, then jkt thumbprint.
package main

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/go-jose/go-jose/v4"
	"github.com/google/uuid"
)

func main() {
	htm := flag.String("htm", "POST", "HTTP method")
	htu := flag.String("htu", "", "HTTP URI (required)")
	ath := flag.String("ath", "", "optional access token hash (ath claim)")
	nonce := flag.String("nonce", "", "optional DPoP nonce claim")
	keyPath := flag.String("key", "", "optional path to PEM EC private key (create if missing)")
	flag.Parse()
	if *htu == "" {
		fmt.Fprintln(os.Stderr, "htu is required")
		os.Exit(1)
	}

	priv, err := loadOrCreateKey(*keyPath)
	if err != nil {
		panic(err)
	}
	jwk := &jose.JSONWebKey{Key: &priv.PublicKey, Algorithm: string(jose.ES256), Use: "sig"}
	jktBytes, err := jwk.Thumbprint(crypto.SHA256)
	if err != nil {
		panic(err)
	}
	jkt := base64.RawURLEncoding.EncodeToString(jktBytes)

	signer, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.ES256, Key: priv}, &jose.SignerOptions{
		ExtraHeaders: map[jose.HeaderKey]interface{}{
			"typ": "dpop+jwt",
			"jwk": jwk,
		},
	})
	if err != nil {
		panic(err)
	}
	claims := map[string]interface{}{
		"jti": uuid.NewString(),
		"htm": *htm,
		"htu": *htu,
		"iat": time.Now().Unix(),
	}
	if *ath != "" {
		claims["ath"] = *ath
	}
	if *nonce != "" {
		claims["nonce"] = *nonce
	}
	payload, err := json.Marshal(claims)
	if err != nil {
		panic(err)
	}
	obj, err := signer.Sign(payload)
	if err != nil {
		panic(err)
	}
	proof, err := obj.CompactSerialize()
	if err != nil {
		panic(err)
	}
	fmt.Printf("%s\n%s\n", proof, jkt)
}

func loadOrCreateKey(path string) (*ecdsa.PrivateKey, error) {
	if path == "" {
		return ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	}
	if data, err := os.ReadFile(path); err == nil && len(data) > 0 {
		block, _ := pem.Decode(data)
		if block == nil {
			return nil, fmt.Errorf("invalid PEM in %s", path)
		}
		key, err := x509.ParseECPrivateKey(block.Bytes)
		if err != nil {
			return nil, err
		}
		return key, nil
	}
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}
	der, err := x509.MarshalECPrivateKey(priv)
	if err != nil {
		return nil, err
	}
	pemBytes := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: der})
	if err := os.WriteFile(path, pemBytes, 0600); err != nil {
		return nil, err
	}
	return priv, nil
}
