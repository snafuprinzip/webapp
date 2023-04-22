package webapp

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/julienschmidt/httprouter"
	"golang.org/x/crypto/bcrypt"
	"gopkg.in/yaml.v3"
	"log"
	"net/http"
	"os"
	"strings"
)

// User contains the necessary data for a registered user of the web service
type User struct {
	ID             string
	Username       string
	Email          string
	HashedPassword string
	Sessions       []Session
}

// UserStore is an abstraction interface to allow multiple data sources to save user info to
type UserStore interface {
	Find(string) (*User, error)
	All() ([]User, error)
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

// CreateAdminAccount creates a superuser account for the application administration if none exists yet
func CreateAdminAccount() {
	// Create a random admin user if none exists
	admin, err := GlobalUserStore.Find("admin")
	if err != nil {
		log.Fatalf("Unable to read from global user store: %s\n", err)
	}

	if admin == nil {
		password := GenerateRandomPassword(16)
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), 10)
		if err != nil {
			log.Fatalf("Unable to generate admin password: %s\n", err)
		}

		admin = &User{
			ID:             "admin",
			Email:          "root@localhost",
			HashedPassword: string(hashedPassword),
			Username:       "admin",
		}
		err = GlobalUserStore.Save(admin)
		if err != nil {
			log.Fatalf("Unable to save admin password: %s\n", err)
		}
		log.Printf("No Admin account found, creating a new one with the following credentials:\n"+
			"Username: admin\nPassword: %s\n\nPlease not these down and put it in a secure location.", password)
	}
}

// NewUser creates a new User and encrypts his password
func NewUser(username, email, password string) (User, error) {
	user := User{
		ID:       GenerateID("usr", userIDLength),
		Email:    email,
		Username: username,
	}

	lang, _ := GetLanguageFromUser(user.ID)

	// check for empty form fields
	if username == "" {
		return user, errNoUsername[lang]
	}
	if email == "" {
		return user, errNoEmail[lang]
	}
	if password == "" {
		return user, errNoPassword[lang]
	}

	// check if password is long enough
	if len(password) < passwordLength {
		return user, errPasswordTooShort[lang]
	}

	// check if username exists
	existingUser, err := GlobalUserStore.FindByUsername(username)
	if err != nil {
		return user, err
	}
	if existingUser != nil {
		return user, errUsernameExists[lang]
	}

	// check if email exists
	existingUser, err = GlobalUserStore.FindByEmail(email)
	if err != nil {
		return user, err
	}
	if existingUser != nil {
		return user, errEmailExists[lang]
	}

	// encrypt password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), hashCost)
	user.HashedPassword = string(hashedPassword)

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
		return out, errCredentialsIncorrect["en"]
	}

	lang, _ := GetLanguageFromUser(existingUser.ID)

	// compare user + password combination if user has been found before
	if bcrypt.CompareHashAndPassword(
		[]byte(existingUser.HashedPassword),
		[]byte(password),
	) != nil {
		return out, errCredentialsIncorrect[lang]
	}
	return existingUser, nil
}

// UpdateUser updates the User's email address and, if the current password matches, the password
func UpdateUser(user *User, email, currentPassword, newPassword string) (User, error) {
	var lang string = "en"

	// make a shallow copy of the user and set email
	out := *user
	out.Email = email

	// Check if email is already in use by another user
	existingUser, err := GlobalUserStore.FindByEmail(email)
	if err != nil {
		return out, err
	}
	if existingUser != nil {
		lang, _ = GetLanguageFromUser(existingUser.ID)
	}
	if existingUser != nil && existingUser.ID != user.ID {
		return out, errEmailExists[lang]
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
		return out, errPasswordIncorrect[lang]
	}

	if newPassword == "" {
		return out, errNoPassword[lang]
	}

	if len(newPassword) < passwordLength {
		return out, errPasswordTooShort[lang]
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
func HandleUserCreate(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
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

func HandleUsersIndex(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	var users []User
	var err error

	user := RequestUser(r)
	if user == nil {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	if user.ID == "admin" {
		users, err = GlobalUserStore.All()
	} else {
		w.WriteHeader(http.StatusForbidden)
		return
	}
	if err != nil {
		log.Println("Unable to read from GlobalUserStore:", err)
	}

	RenderTemplate(w, r, "users/index", map[string]interface{}{
		"Pagetitle": "Users",
		"Users":     users,
	})
}

func HandleUsersGETv1(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	var users []User
	var err error

	user := RequestUser(r)
	if user == nil {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	if user.ID == "admin" {
		users, err = GlobalUserStore.All()
	} else {
		w.WriteHeader(http.StatusForbidden)
		return
	}
	if err != nil {
		log.Println("Unable to read from GlobalUserStore:", err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	writer := json.NewEncoder(w)
	writer.SetIndent("", "    ")
	if err := writer.Encode(users); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
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

func (store FileUserStore) All() ([]User, error) {
	var userlist []User
	for _, v := range store.Users {
		userlist = append(userlist, v)
	}
	return userlist, nil
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
	_, err := GlobalPostgresDB.Exec(`
CREATE TABLE IF NOT EXISTS users (
  id varchar(255) NOT NULL DEFAULT '',
  username varchar(255) NOT NULL DEFAULT '',
  email varchar(255) NOT NULL DEFAULT '',
  password text NOT NULL,
  PRIMARY KEY (id)
);
`)
	if err != nil {
		log.Fatalf("Unable to create users table in database: %s\n", err)
	}

	_, err = GlobalPostgresDB.Exec(`
CREATE INDEX IF NOT EXISTS username_idx ON users( username );`)
	if err != nil {
		log.Fatalf("Unable to create users table username index in database: %s\n", err)
	}

	_, err = GlobalPostgresDB.Exec(`
CREATE INDEX IF NOT EXISTS email_idx ON users( email );`)
	if err != nil {
		log.Fatalf("Unable to create users table email index in database: %s\n", err)
	}

	return &DBUserStore{
		db: GlobalPostgresDB,
	}
}

func (store DBUserStore) Save(user *User) error {
	_, err := store.db.Exec(
		`
	INSERT INTO users
	    (id, username, email, password)
	    VALUES ($1, $2, $3, $4)
	    	    ON CONFLICT (id)
	    DO UPDATE SET id=$1, username=$2, email=$3, password=$4`,
		user.ID,
		user.Username,
		user.Email,
		user.HashedPassword,
	)
	return err
}

func (store DBUserStore) All() ([]User, error) {
	rows, err := store.db.Query(
		`
		SELECT id, username, email, password
		FROM users
		`,
	)
	if err != nil {
		return nil, err
	}

	var users []User
	for rows.Next() {
		user := User{}
		err := rows.Scan(
			&user.ID,
			&user.Username,
			&user.Email,
			&user.HashedPassword,
		)
		if err != nil {
			return nil, err
		}

		user.Sessions, _ = GlobalSessionStore.FindByUser(user.ID)

		users = append(users, user)
	}

	return users, nil
}

func (store DBUserStore) Find(id string) (*User, error) {
	row := store.db.QueryRow(
		`
		SELECT id, username, email, password
		FROM users
		WHERE id = $1`,
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
	user.Sessions, _ = GlobalSessionStore.FindByUser(user.ID)
	return &user, err
}

func (store DBUserStore) FindByUsername(name string) (*User, error) {
	row := store.db.QueryRow(
		`
		SELECT id, username, email, password
		FROM users
		WHERE username = $1`,
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
	user.Sessions, _ = GlobalSessionStore.FindByUser(user.ID)
	return &user, err
}

func (store DBUserStore) FindByEmail(email string) (*User, error) {
	row := store.db.QueryRow(
		`
		SELECT id, username, email, password
		FROM users
		WHERE email = $1`,
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
	user.Sessions, _ = GlobalSessionStore.FindByUser(user.ID)
	return &user, err
}
