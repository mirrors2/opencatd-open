/*
https://docs.anthropic.com/zh-CN/api/claude-on-vertex-ai

MODEL_ID=claude-3-5-sonnet@20240620
REGION=us-east5
PROJECT_ID=MY_PROJECT_ID

curl \
-X POST \
-H "Authorization: Bearer $(gcloud auth print-access-token)" \
-H "Content-Type: application/json" \
https://$LOCATION-aiplatform.googleapis.com/v1/projects/${PROJECT_ID}/locations/${LOCATION}/publishers/anthropic/models/${MODEL_ID}:streamRawPredict \
-d '{
  "anthropic_version": "vertex-2023-10-16",
  "messages": [{
    "role": "user",
    "content": "介绍一下你自己"
  }],
  "stream": true,
  "max_tokens": 4096
}'
*/

package vertexai

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/golang-jwt/jwt"
)

// json文件存储在ApiKey.ApiSecret中
type VertexSecretKey struct {
	Type                    string `json:"type"`
	ProjectID               string `json:"project_id"`
	PrivateKeyID            string `json:"private_key_id"`
	PrivateKey              string `json:"private_key"`
	ClientEmail             string `json:"client_email"`
	ClientID                string `json:"client_id"`
	AuthURI                 string `json:"auth_uri"`
	TokenURI                string `json:"token_uri"`
	AuthProviderX509CertURL string `json:"auth_provider_x509_cert_url"`
	ClientX509CertURL       string `json:"client_x509_cert_url"`
	UniverseDomain          string `json:"universe_domain"`
}

type VertexClaudeModel struct {
	VertexName string
	Region     string
}

var VertexClaudeModelMap = map[string]VertexClaudeModel{
	"claude-3-opus": {
		VertexName: "claude-3-opus@20240229",
		Region:     "us-east5",
	},
	"claude-3-sonnet": {
		VertexName: "claude-3-sonnet@20240229",
		Region:     "us-central1",
		// Region:     "asia-southeast1",
	},
	"claude-3-haiku": {
		VertexName: "claude-3-haiku@20240307",
		Region:     "us-central1",
		// Region:     "europe-west4",
	},
	"claude-3-opus-20240229": {
		VertexName: "claude-3-opus@20240229",
		Region:     "us-east5",
	},
	"claude-3-sonnet-20240229": {
		VertexName: "claude-3-sonnet@20240229",
		Region:     "us-central1",
		// Region:     "asia-southeast1",
	},
	"claude-3-haiku-20240307": {
		VertexName: "claude-3-haiku@20240307",
		Region:     "us-central1",
		// Region:     "europe-west4",
	},
	"claude-3-5-sonnet": {
		VertexName: "claude-3-5-sonnet@20240620",
		Region:     "us-east5",
		// Region:     "europe-west1",
	},
	"claude-3-5-sonnet-20240620": {
		VertexName: "claude-3-5-sonnet@20240620",
		Region:     "us-east5",
		// Region:     "europe-west1",
	},
}

func createSignedJWT(email, privateKeyPEM string) (string, error) {
	block, _ := pem.Decode([]byte(privateKeyPEM))
	if block == nil {
		return "", fmt.Errorf("failed to parse PEM block containing the private key")
	}

	privateKey, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return "", err
	}

	rsaKey, ok := privateKey.(*rsa.PrivateKey)
	if !ok {
		return "", fmt.Errorf("not an RSA private key")
	}

	now := time.Now()
	claims := jwt.MapClaims{
		"iss":   email,
		"aud":   "https://www.googleapis.com/oauth2/v4/token",
		"iat":   now.Unix(),
		"exp":   now.Add(10 * time.Minute).Unix(),
		"scope": "https://www.googleapis.com/auth/cloud-platform",
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	return token.SignedString(rsaKey)
}

func exchangeJwtForAccessToken(signedJWT string) (string, error) {
	authURL := "https://www.googleapis.com/oauth2/v4/token"
	data := url.Values{}
	data.Set("grant_type", "urn:ietf:params:oauth:grant-type:jwt-bearer")
	data.Set("assertion", signedJWT)

	resp, err := http.PostForm(authURL, data)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	accessToken, ok := result["access_token"].(string)
	if !ok {
		return "", fmt.Errorf("access token not found in response")
	}

	return accessToken, nil
}

// 获取gcloud auth token
func GcloudAuth(ClientEmail, PrivateKey string) (string, error) {
	signedJWT, err := createSignedJWT(ClientEmail, PrivateKey)
	if err != nil {
		return "", err
	}

	token, err := exchangeJwtForAccessToken(signedJWT)
	if err != nil {
		return "", fmt.Errorf("Invalid jwt token: %v\n", err)
	}

	return token, nil
}
