package webapp

import (
	"errors"
)

type ValidationError error

var (
	errNoUsername       = ValidationError(errors.New("You must supply a username"))
	errNoEmail          = ValidationError(errors.New("You must supply an email"))
	errNoPassword       = ValidationError(errors.New("You must supply a password"))
	errPasswordTooShort = ValidationError(errors.New("Your password is too short"))

	errUsernameExists = ValidationError(errors.New("Username is already taken"))
	errEmailExists    = ValidationError(errors.New("An account has already been registered with that email address"))

	errCredentialsIncorrect = ValidationError(errors.New("We couldn't find a user with this username+password combination"))
	errPasswordIncorrect    = ValidationError(errors.New("Passwords didn't match"))
)

func IsValidationError(err error) bool {
	_, ok := err.(ValidationError)
	return ok
}
