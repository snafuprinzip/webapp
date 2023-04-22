package webapp

import (
	"errors"
)

type ValidationError error

var (
	errNoUsername = map[string]ValidationError{
		"en": ValidationError(errors.New("you must supply a username")),
		"de": ValidationError(errors.New("sie m&uuml;ssen einen Benutzernamen angeben")),
	}
	errNoEmail = map[string]ValidationError{
		"en": ValidationError(errors.New("you must supply an email")),
		"de": ValidationError(errors.New("sie m&uuml;ssen eine E-Mail Adresse angeben")),
	}
	errNoPassword = map[string]ValidationError{
		"en": ValidationError(errors.New("you must supply a password")),
		"de": ValidationError(errors.New("sie m&uuml;ssen ein Passwort angeben")),
	}
	errPasswordTooShort = map[string]ValidationError{
		"en": ValidationError(errors.New("your password is too short")),
		"de": ValidationError(errors.New("das angegebene Passwort ist zu kurz")),
	}
	errUsernameExists = map[string]ValidationError{
		"en": ValidationError(errors.New("username is already taken")),
		"de": ValidationError(errors.New("der Benutzername ist bereits vergeben")),
	}
	errEmailExists = map[string]ValidationError{
		"en": ValidationError(errors.New("an account has already been registered with that email address")),
		"de": ValidationError(errors.New("ein Konto mit der E-Mail Adresse existiert bereits")),
	}

	errCredentialsIncorrect = map[string]ValidationError{
		"en": ValidationError(errors.New("couldn't find a user with this username+password combination")),
		"de": ValidationError(errors.New("kein Benutzer mit diesem Namen und dem angegebenen Passwort gefunden")),
	}
	errPasswordIncorrect = map[string]ValidationError{
		"en": ValidationError(errors.New("passwords didn't match")),
		"de": ValidationError(errors.New("die Passw&ouml;rter stimmen nicht &uuml;berein")),
	}
)

func IsValidationError(err error) bool {
	_, ok := err.(ValidationError)
	return ok
}
