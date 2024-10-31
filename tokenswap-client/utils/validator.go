package utils

import (
	"regexp"
)

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
