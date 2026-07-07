package auth

import "golang.org/x/crypto/bcrypt"

// HashPassword returns a bcrypt hash of the plain-text password.
// bcrypt automatically salts the password, so we don't need to manage salts ourselves.
func HashPassword(plain string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// CheckPassword compares a plain-text password against a stored bcrypt hash.
// Returns true if they match.
func CheckPassword(hash, plain string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain))
	return err == nil
}
