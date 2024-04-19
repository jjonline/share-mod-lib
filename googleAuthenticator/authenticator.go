package googleAuthenticator

import (
	"crypto/rand"
	"encoding/base32"
	"regexp"
	"strings"
	"time"

	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
)

type Authenticator struct {
	key *otp.Key
}

// NewAuthenticator new google auth
func NewAuthenticator(issuer string, accountName string, formattedKey string) *Authenticator {
	rx := regexp.MustCompile(`\W+`)
	secret := []byte(rx.ReplaceAllString(formattedKey, ""))
	ret, _ := totp.Generate(totp.GenerateOpts{
		Issuer:      issuer,
		AccountName: accountName,
		SecretSize:  uint(len(secret)),
		Secret:      secret,
	})
	return &Authenticator{key: ret}
}

// VerifyToken verify token
func (a *Authenticator) VerifyToken(passcode string) bool {
	rv, _ := totp.ValidateCustom(
		passcode,
		a.key.Secret(),
		time.Now().UTC(),
		totp.ValidateOpts{
			Period:    30,
			Skew:      1,
			Digits:    otp.DigitsSix,
			Algorithm: otp.AlgorithmSHA1,
		},
	)
	return rv
}

// GenerateToken 生成passcode
func (a *Authenticator) GenerateToken() (passcode string) {
	passcode, _ = totp.GenerateCode(a.key.Secret(), time.Now().UTC())
	return
}

// GenerateTotpUri 生成google auth app可以識別的uri
func (a *Authenticator) GenerateTotpUri() string {
	return a.key.URL()
}

// GenerateKey get key
func GenerateKey() (formattedKey string) {
	formattedKey = encodeGoogleAuthKey(generateOtpKey())
	return
}

// Generate a key
func generateOtpKey() []byte {
	// 20 cryptographically random binary bytes (160-bit key)
	key := make([]byte, 20)
	_, _ = rand.Read(key)
	return key
}

// Text-encode the key as base32 (in the style of Google Authenticator - same as Facebook, Microsoft, etc)
func encodeGoogleAuthKey(bin []byte) string {
	// 32 ascii characters without trailing '='s
	rx := regexp.MustCompile(`=`)
	base32Str := rx.ReplaceAllString(base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(bin), "")
	base32Str = strings.ToLower(base32Str)

	// lowercase with a space every 4 characters
	rx = regexp.MustCompile(`(\w{4})`)
	key := strings.TrimSpace(rx.ReplaceAllString(base32Str, "$1 "))

	return key
}
