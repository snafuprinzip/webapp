package webapp

import (
	"database/sql"
	"fmt"
	"github.com/go-yaml/yaml"
	"github.com/julienschmidt/httprouter"
	"golang.org/x/crypto/bcrypt"
	"log"
	"net/http"
	"os"
	"strings"
)

// User contains the necessary data for a registered user of the web service
type User struct {
	ID             string
	Email          string
	HashedPassword string
	Username       string
}

// UserStore is an abstraction interface to allow multiple data sources to save user info to
type UserStore interface {
	Find(string) (*User, error)
	FindByEmail(string) (*User, error)
	FindByUsername(string) (*User, error)
	Save(*User) error
}

// FileUserStore is an implementation of UserStore to save user data to the filesystem
type FileUserStore struct {
	filename string
	Users    map[string]User
}

// DBUserStore is an implementation of UserStore to save user data in the database
type DBUserStore struct {
	db *sql.DB
}

const (
	hashCost       = 10
	passwordLength = 8
	userIDLength   = 16
)

// GlobalUserStore is the Global Database of users
var GlobalUserStore UserStore

// NewUser creates a new User and encrypts his password
func NewUser(username, email, password string) (User, error) {
	user := User{
		Email:    email,
		Username: username,
	}

	// check for empty form fields
	if username == "" {
		return user, errNoUsername
	}
	if email == "" {
		return user, errNoEmail
	}
	if password == "" {
		return user, errNoPassword
	}

	// check if password is long enough
	if len(password) < passwordLength {
		return user, errPasswordTooShort
	}

	// check if username exists
	existingUser, err := GlobalUserStore.FindByUsername(username)
	if err != nil {
		return user, err
	}
	if existingUser != nil {
		return user, errUsernameExists
	}

	// check if email exists
	existingUser, err = GlobalUserStore.FindByEmail(email)
	if err != nil {
		return user, err
	}
	if existingUser != nil {
		return user, errEmailExists
	}

	// encrypt password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), hashCost)
	user.HashedPassword = string(hashedPassword)

	user.ID = GenerateID("usr", userIDLength)

	return user, err
}

// FindUser returns the user with the given username + password combination if found
func FindUser(username, password string) (*User, error) {
	// create dummy user to return username if login fails
	out := &User{
		Username: username,
	}

	// find user or return dummy with error message if it fails
	existingUser, err := GlobalUserStore.FindByUsername(username)
	if err != nil {
		return out, err
	}
	if existingUser == nil {
		return out, errCredentialsIncorrect
	}

	// compare user + password combination if user has been found before
	if bcrypt.CompareHashAndPassword(
		[]byte(existingUser.HashedPassword),
		[]byte(password),
	) != nil {
		return out, errCredentialsIncorrect
	}
	return existingUser, nil
}

// UpdateUser updates the User's email address and, if the current password matches, the password
func UpdateUser(user *User, email, currentPassword, newPassword string) (User, error) {
	// make a shallow copy of the user and set email
	out := *user
	out.Email = email

	// Check if email is already in use by another user
	existingUser, err := GlobalUserStore.FindByEmail(email)
	if err != nil {
		return out, err
	}
	if existingUser != nil && existingUser.ID != user.ID {
		return out, errEmailExists
	}

	// update email address
	user.Email = email

	// don't update password if existing password is empty
	if currentPassword == "" {
		return out, nil
	}

	if bcrypt.CompareHashAndPassword(
		[]byte(user.HashedPassword),
		[]byte(currentPassword),
	) != nil {
		return out, errPasswordIncorrect
	}

	if newPassword == "" {
		return out, errNoPassword
	}

	if len(newPassword) < passwordLength {
		return out, errPasswordTooShort
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), hashCost)
	user.HashedPassword = string(hashedPassword)

	return out, err
}

/****************************************
***  Handler                          ***
*****************************************/

// HandleUserNew shows the user registration page
// (GET /registration)
func HandleUserNew(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	// Display Home Page
	RenderTemplate(w, r, "users/new", map[string]interface{}{
		"Pagetitle": "Register User",
	})
}

// HandleUserCreate takes the form values from the registration page and creates a new user
// (POST /registration)
func HandleUserCreate(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	// Create User
	user, err := NewUser(
		r.FormValue("username"),
		r.FormValue("email"),
		r.FormValue("password"),
	)

	if err != nil {
		if IsValidationError(err) {
			RenderTemplate(w, r, "users/new", map[string]interface{}{
				"Pagetitle": "Register User",
				"Error":     err.Error(),
				"User":      user,
			})
			return
		}
		panic(err)
	}

	// save user
	err = GlobalUserStore.Save(&user)
	if err != nil {
		log.Fatalf("Unable to save user info: %s\n", err)
	}

	// create a new session
	session := NewSession(w)
	session.UserID = user.ID
	fmt.Println(GlobalSessionStore)

	err = GlobalSessionStore.Save(session)
	if err != nil {
		log.Fatalf("Unable to save session info: %s\n", err)
	}
	fmt.Println(GlobalSessionStore)

	// redirect back to / with status message
	http.Redirect(w, r, "/?flash=User+created", http.StatusFound)
}

// HandleUserEdit shows the account information page to change email or password
// (GET /account)
func HandleUserEdit(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	user := RequestUser(r)
	RenderTemplate(w, r, "users/edit", map[string]interface{}{
		"Pagetitle": "Edit User",
		"User":      user,
	})
}

// HandleUserUpdate updates the user information with the new email or password information
// from the account information page
// (POST /account)
func HandleUserUpdate(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	currentUser := RequestUser(r)
	email := r.FormValue("email")
	currentPassword := r.FormValue("currentPassword")
	newPassword := r.FormValue("newPassword")

	user, err := UpdateUser(currentUser, email, currentPassword, newPassword)
	if err != nil {
		if IsValidationError(err) {
			RenderTemplate(w, r, "users/edit", map[string]interface{}{
				"Pagetitle": "Edit User",
				"User":      user,
				"Error":     err.Error(),
			})
			return
		}
		log.Fatalf("Error updating user: %s\n", err)
	}

	err = GlobalUserStore.Save(currentUser)
	if err != nil {
		log.Fatalf("Error updating user in Global user store: %s\n", err)
	}

	http.Redirect(w, r, "/account?flash=user+updated", http.StatusFound)
}

/****************************************
***  Storage Backends                 ***
*****************************************/

/**********************************
***  File User Store            ***
***********************************/

// NewFileUserStore creates a new FileUserStore under the given filename
func NewFileUserStore(filename string) (*FileUserStore, error) {
	store := &FileUserStore{
		Users:    map[string]User{},
		filename: filename,
	}
	contents, err := os.ReadFile(filename)
	if err != nil {
		// ignore error if it's a file does not exist error
		if os.IsNotExist(err) {
			return store, nil
		}
		return nil, err
	}
	err = yaml.Unmarshal(contents, store)
	if err != nil {
		return nil, err
	}
	return store, nil
}

// Save adds a new user and saves the GlobalUserStore to the Filesystem
func (store FileUserStore) Save(user *User) error {
	store.Users[user.ID] = *user

	contents, err := yaml.Marshal(store)
	if err != nil {
		return err
	}

	err = os.WriteFile(store.filename, contents, 0660)
	if err != nil {
		return err
	}

	return nil
}

// Find returns the user with the given id if found
func (store FileUserStore) Find(id string) (*User, error) {
	user, ok := store.Users[id]
	if ok {
		return &user, nil
	}
	return nil, nil
}

// FindByUsername returns the user with the given Username if found
func (store FileUserStore) FindByUsername(name string) (*User, error) {
	if name == "" {
		return nil, nil
	}

	for _, user := range store.Users {
		if strings.ToLower(name) == strings.ToLower(user.Username) {
			return &user, nil
		}
	}
	return nil, nil
}

// FindByEmail returns the user with the given email address if found
func (store FileUserStore) FindByEmail(email string) (*User, error) {
	if email == "" {
		return nil, nil
	}

	for _, user := range store.Users {
		if strings.ToLower(email) == strings.ToLower(user.Email) {
			return &user, nil
		}
	}
	return nil, nil
}

/**********************************
***  DB User Store              ***
***********************************/

func NewDBUserStore() UserStore {
	schema, err := os.ReadFile("sql/users_table.sql")
	if err != nil {
		log.Fatalf("Unable to read sql schema for the users table: %s\n", err)
	}

	_, err = GlobalMySQLDB.Exec(string(schema))
	if err != nil {
		log.Fatalf("Unable to create schema in database: %s\n", err)
	}

	return &DBUserStore{
		db: GlobalMySQLDB,
	}
}

func (store DBUserStore) Save(user *User) error {
	_, err := store.db.Exec(
		`
	REPLACE INTO users
	    (id, username, email, password)
	    VALUES (?, ?, ?, ?)`,
		user.ID,
		user.Username,
		user.Email,
		user.HashedPassword,
	)
	return err
}

func (store DBUserStore) Find(id string) (*User, error) {
	row := store.db.QueryRow(
		`
		SELECT id, username, email, password
		FROM users
		WHERE id = ?`,
		id,
	)

	user := User{}
	err := row.Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.HashedPassword,
	)
	// return nil and no error when the Scan returns no findings
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &user, err
}

func (store DBUserStore) FindByUsername(name string) (*User, error) {
	row := store.db.QueryRow(
		`
		SELECT id, username, email, password
		FROM users
		WHERE username = ?`,
		name,
	)

	user := User{}
	err := row.Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.HashedPassword,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &user, err
}

func (store DBUserStore) FindByEmail(email string) (*User, error) {
	row := store.db.QueryRow(
		`
		SELECT id, username, email, password
		FROM users
		WHERE email = ?`,
		email,
	)

	user := User{}
	err := row.Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.HashedPassword,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &user, err
}
