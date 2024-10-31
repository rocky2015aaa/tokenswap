package utils

import (
	"crypto/rand"
	"math/big"
	"regexp"
	"strconv"
	"strings"
)

const (
	charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	length  = 6
)

func GenerateID() (string, error) {
	id := make([]byte, length)
	for i := range id {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			return "", err
		}
		id[i] = charset[num.Int64()]
	}
	return string(id), nil
}

func ValidateDecimal1or2Places(value float64) bool {
	numDecPlaces := func(v float64) int {
		s := strconv.FormatFloat(v, 'f', -1, 64)
		i := strings.IndexByte(s, '.')
		if i > -1 {
			return len(s) - i - 1
		}
		return 0
	}

	decimalPlaces := numDecPlaces(value)
	if value == float64(int(value)) {
		return true
	} else {
		return decimalPlaces == 1 || decimalPlaces == 2
	}
}

func ValidateTokenAddress(token, address string) bool {
	var re *regexp.Regexp
	if token == "XEL" {
		re = regexp.MustCompile(`^xet:[a-z0-9]{59}$`)
	}
	if token == "USDC" || token == "USDT" {
		re = regexp.MustCompile(`^0x[0-9a-fA-F]{40}$`)
	}
	return re.MatchString(address)
}
