package v1

import (
	"bytes"
	"crypto/hmac"
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"github.com/acquia/http-hmac-go/signers"
	"hash"
	"net/http"
	"regexp"
	"strings"
)

type V1Signer struct {
	*signers.Digester
	*signers.Identifiable
}

func NewV1Signer(digest func() hash.Hash) (*V1Signer, *signers.AuthenticationError) {
	re, err := regexp.Compile("(?i)^\\s*Acquia\\s*[^:]+\\s*:\\s*[0-9a-zA-Z\\+/=]+\\s*$")
	if err != nil {
		return nil, signers.Errorf(500, signers.ErrorTypeInternalError, "Could not compile regular expression for identifier: %s", err.Error())
	}
	return &V1Signer{
		Digester: &signers.Digester{
			Digest: digest,
		},
		Identifiable: &signers.Identifiable{
			IdRegex: re,
		},
	}, nil
}

func ParseAuthHeaders(req *http.Request) map[string]string {
	return map[string]string{}
}

func (v *V1Signer) ParseAuthHeaders(req *http.Request) map[string]string {
	return ParseAuthHeaders(req)
}

func (v *V1Signer) HashBody(req *http.Request) (string, *signers.AuthenticationError) {
	h := md5.New()
	data, err := signers.ReadBody(req)
	if err != nil {
		return "", signers.Errorf(500, signers.ErrorTypeInternalError, "Failed to read request body: %s", err.Error())
	}
	h.Write(data)

	return hex.EncodeToString(h.Sum(nil)), nil
}

func (v *V1Signer) readCustomHeaders(authHeaders map[string]string) []string {
	if d, ok := authHeaders["headers"]; ok {
		return strings.Split(d, ";")
	}
	return []string{}
}

func (v *V1Signer) CreateSignable(req *http.Request, authHeaders map[string]string) []byte {
	bodyhash, err := v.HashBody(req)
	if err != nil {
		return nil
	}
	var b bytes.Buffer

	b.WriteString(strings.ToUpper(req.Method))
	b.WriteString("\n")

	b.WriteString(bodyhash)
	b.WriteString("\n")

	b.WriteString(strings.ToLower(req.Header.Get("Content-Type")))
	b.WriteString("\n")

	b.WriteString(req.Header.Get("Date"))
	b.WriteString("\n")

	ch := v.readCustomHeaders(authHeaders)
	if len(ch) > 0 {
		for _, hname := range ch {
			b.WriteString(fmt.Sprintf("%s: %s\n", strings.ToLower(hname), strings.Join(req.Header[hname], ", ")))
		}
	} else {
		b.WriteString("\n")
	}

	b.WriteString(req.URL.RequestURI())

	return b.Bytes()
}

func (v *V1Signer) Sign(req *http.Request, authHeaders map[string]string, secret string) (string, *signers.AuthenticationError) {
	h := hmac.New(v.Digest, []byte(secret))
	b := v.CreateSignable(req, authHeaders)
	h.Write(b)
	hsm := h.Sum(nil)
	return base64.StdEncoding.EncodeToString(hsm), nil
}

func (v *V1Signer) GetIdentificationRegex() *regexp.Regexp {
	return v.IdRegex
}

func (v *V1Signer) GetResponseSigner() signers.ResponseSigner {
	return nil
}
