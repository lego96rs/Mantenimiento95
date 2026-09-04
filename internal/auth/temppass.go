package auth

import "crypto/rand"

const tempPassAlphabet = "ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz23456789"

func GenerateTempPassword() (string, error) {
	buf := make([]byte, 16)
	random := make([]byte, len(buf))
	if _, err := rand.Read(random); err != nil {
		return "", err
	}

	for i, b := range random {
		buf[i] = tempPassAlphabet[int(b)%len(tempPassAlphabet)]
	}

	return string(buf), nil
}
