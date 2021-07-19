package jwt

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

func (t *jwt) Parse(jwt string, options ...ParseOptions) (Token, string, error) {

	const NoPadding rune = -1
	var token Token
	var now = time.Now().UTC().Unix()
	var parseOptions ParseOptions

	// Init Parse Options
	if len(options) != 0 {
		parseOptions = options[0]
	} else {
		parseOptions = t.config.ParseOptions
	}

	// Split Token values
	jwtParts := strings.Split(jwt, ".")
	if len(jwtParts) != 3 {
		return Token{}, ValidationErrorMalformed,
			fmt.Errorf("%s: failed to split the token values", ValidationErrorMalformed)
	}

	// Parse Headers
	valueByte, err := base64.URLEncoding.WithPadding(NoPadding).DecodeString(jwtParts[0])
	if err != nil {
		return Token{}, ValidationErrorHeadersMalformed, err
	}
	err = json.Unmarshal(valueByte, &token.Headers)
	if err != nil {
		return Token{}, ValidationErrorHeadersMalformed, err
	}

	// Parse Claims
	valueByte, err = base64.URLEncoding.WithPadding(NoPadding).DecodeString(jwtParts[1])
	if err != nil {
		return Token{}, ValidationErrorClaimsMalformed, err
	}
	err = json.Unmarshal(valueByte, &token.Claims)
	if err != nil {
		return Token{}, ValidationErrorClaimsMalformed, err
	}

	// Get Signature
	token.Signature = jwtParts[2]

	// Validate Signature
	if parseOptions.SkipSignatureValidation == false {
		headersPart, err := makeHeaderPart(token.Headers)
		if err != nil {
			return Token{}, ValidationErrorUnverifiable, fmt.Errorf("failed to make the headersPart: %s", err)
		}
		claimsPart, err := makeClaimsPart(token.Claims)
		if err != nil {
			return Token{}, ValidationErrorUnverifiable, fmt.Errorf("failed to make the claimsPart: %s", err)
		}
		unsignedToken := headersPart + "." + claimsPart
		signature, err := makeSignature(unsignedToken, token.Headers.SignatureAlgorithm, t.config.Key)
		if err != nil {
			return Token{}, ValidationErrorUnverifiable, fmt.Errorf("failed to make the signature: %s", err)
		}
		if signature != token.Signature {
			return Token{}, ValidationErrorSignatureInvalid,
				fmt.Errorf("failed to validate signature: jwtSample %s, jwt %s",
					headersPart + "." + claimsPart + "." + signature, jwt)
		}
	}

	// Validate Headers
	if parseOptions.RequiredHeaderContentType && token.Headers.ContentType == "" {
		return Token{}, ValidationErrorHeadersContentType, errTokenIsInvalid
	}
	if parseOptions.RequiredHeaderKeyId && token.Headers.KeyId == "" {
		return Token{}, ValidationErrorHeadersKeyId, errTokenIsInvalid
	}
	if parseOptions.RequiredHeaderCritical && token.Headers.Critical == "" {
		return Token{}, ValidationErrorHeadersCritical, errTokenIsInvalid
	}

	// Validate Claims
	if parseOptions.RequiredClaimIssuer && token.Claims.Issuer == "" {
		return Token{}, ValidationErrorClaimsIssuer, errTokenIsInvalid
	}
	if parseOptions.RequiredClaimSubject && token.Claims.Subject == "" {
		return Token{}, ValidationErrorClaimsSubject, errTokenIsInvalid
	}
	if parseOptions.RequiredClaimAudience && token.Claims.Audience == "" {
		return Token{}, ValidationErrorClaimsAudience, errTokenIsInvalid
	}
	if parseOptions.RequiredClaimJwtId && token.Claims.JwtId == "" {
		return Token{}, ValidationErrorClaimsJwtId, errTokenIsInvalid
	}
	if parseOptions.RequiredClaimData && token.Claims.Data == nil {
		return Token{}, ValidationErrorClaimsData, errTokenIsInvalid
	}
	if parseOptions.SkipClaimsValidation == false {
		// Validate ExpirationTime value
		if now > time.Unix(token.Claims.ExpirationTime, 0).UTC().Unix() {
			return Token{}, ValidationErrorClaimsExpired, errTokenIsInvalid
		}
		// Validate NotBefore value
		if token.Claims.NotBefore != 0 {
			if now < token.Claims.NotBefore {
				return Token{}, ValidationErrorClaimsNotValidYet, errTokenIsInvalid
			}
		}
		// Validate IssuedAt value
		if now < token.Claims.IssuedAt {
			return Token{}, ValidationErrorClaimsIssuedAt, errTokenIsInvalid
		}
	}

	return token, "", nil
}