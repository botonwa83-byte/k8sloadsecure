package pkg

import (
	"errors"
	"unicode"

	"golang.org/x/crypto/bcrypt"
)

func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

func CheckPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func ValidatePassword(password string) error {
	if len(password) < 8 {
		return errors.New("密码长度不能少于8位")
	}
	var hasUpper, hasLower, hasDigit bool
	for _, c := range password {
		switch {
		case unicode.IsUpper(c):
			hasUpper = true
		case unicode.IsLower(c):
			hasLower = true
		case unicode.IsDigit(c):
			hasDigit = true
		}
	}
	if !hasUpper {
		return errors.New("密码必须包含大写字母")
	}
	if !hasLower {
		return errors.New("密码必须包含小写字母")
	}
	if !hasDigit {
		return errors.New("密码必须包含数字")
	}
	return nil
}
