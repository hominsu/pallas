package utils

import (
	"math/rand"
	"strings"
	"time"
)

const (
	Byte     uint64 = 1
	KibiByte        = 1024 * Byte
	MebiByte        = 1024 * KibiByte
	GibiByte        = 1024 * MebiByte
	TebiByte        = 1024 * GibiByte
	PebiByte        = 1024 * TebiByte
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func GeneratePassword(length, minSpecialChar, minNum, minUpperCase int) string {
	const (
		lowerCharSet   = "abcdefghijklmnopqrstuvwxyz"
		upperCharSet   = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
		specialCharSet = "~`!@#$%^&*()_+-=:\";',./<>?[]\\{}|"
		numberSet      = "0123456789"
		allCharSet     = lowerCharSet + upperCharSet + specialCharSet + numberSet
	)

	var password strings.Builder

	for i := 0; i < minSpecialChar; i++ {
		random := rand.Intn(len(specialCharSet))
		password.WriteString(string(specialCharSet[random]))
	}

	for i := 0; i < minNum; i++ {
		random := rand.Intn(len(numberSet))
		password.WriteString(string(numberSet[random]))
	}

	for i := 0; i < minUpperCase; i++ {
		random := rand.Intn(len(upperCharSet))
		password.WriteString(string(upperCharSet[random]))
	}

	remainingLength := length - minSpecialChar - minNum - minUpperCase
	for i := 0; i < remainingLength; i++ {
		random := rand.Intn(len(allCharSet))
		password.WriteString(string(allCharSet[random]))
	}
	inRune := []rune(password.String())
	rand.Shuffle(len(inRune), func(i, j int) {
		inRune[i], inRune[j] = inRune[j], inRune[i]
	})
	return string(inRune)
}
