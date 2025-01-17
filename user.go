package webapp

import (
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"encoding/xml"
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
	ID             string    `json:"id" yaml:"id"`
	Username       string    `json:"username" yaml:"username"`
	Email          string    `json:"email" yaml:"email"`
	HashedPassword string    `json:"hashedPassword,omitempty" yaml:"hashedPassword,omitempty"`
	Sessions       []Session `json:"sessions" yaml:"sessions"`
}

const (
	hashCost       = 10
	passwordLength = 8
	userIDLength   = 16
)

// CreateAdminAccount creates a superuser account for the application administration if none exists yet
func CreateAdminAccount() {
	// Create admin user with random password if none exists
	admin, err := GlobalUserStore.Find("admin")
	if err != nil {
		Logf(FatalLevel, "Unable to read from global user store: %s\n", err)
	}

	if admin == nil {
		password := GenerateRandomPassword(16)
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), 10)
		if err != nil {
			Logf(FatalLevel, "Unable to generate admin password: %s\n", err)
		}

		admin = &User{
			ID:             "admin",
			Email:          "root@localhost",
			HashedPassword: string(hashedPassword),
			Username:       "admin",
		}
		err = GlobalUserStore.Save(admin)
		if err != nil {
			Logf(FatalLevel, "Unable to save admin password: %s\n", err)
		}
		Logf(InfoLevel, "No Admin account found, creating a new one with the following credentials:\n"+
			"Username: admin\nPassword: %s\n\nPlease note these down and put them in a secure location.", password)
	}
}

// NewUser creates a new User and encrypts his password
func NewUser(username, email, password string) (User, error) {
	user := User{
		ID:       GenerateID("usr", userIDLength),
		Email:    email,
		Username: username,
	}

	lang := GetLanguage(user.ID, nil, nil)

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

	lang := GetLanguage(existingUser.ID, nil, nil)

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
func UpdateUser(user *User, username, email, currentPassword, newPassword string, admin bool) (User, error) {
	var lang string = "en"

	// make a shallow copy of the user and set email
	out := *user
	out.Username = username
	out.Email = email

	// Check if email is already in use by another user
	existingUser, err := GlobalUserStore.FindByEmail(email)
	if err != nil {
		return out, err
	}
	if existingUser != nil {
		lang = GetLanguage(existingUser.ID, nil, nil)
	}
	if existingUser != nil && existingUser.ID != user.ID {
		return out, errEmailExists[lang]
	}

	// update email address
	user.Email = email
	user.Username = username

	if !admin {
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
	}

	if newPassword == "" {
		return out, errNoPassword[lang]
	}

	if len(newPassword) < passwordLength {
		return out, errPasswordTooShort[lang]
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), hashCost)
	user.HashedPassword = string(hashedPassword)

	return *user, err
	//return out, err
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
				"Pagetitle": "NewUser",
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
		Logf(FatalLevel, "Unable to save user info: %s\n", err)
	}

	// create a new session
	session := NewSession(w)
	session.UserID = user.ID

	err = GlobalSessionStore.Save(session)
	if err != nil {
		Logf(FatalLevel, "Unable to save session info: %s\n", err)
	}

	// redirect back to / with status message
	http.Redirect(w, r, "/?flash=User+created", http.StatusFound)
}

// HandleUserEdit shows the account information page to change email or password
// (GET /account)
func HandleUserEdit(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	var user *User
	var err error
	uid := params.ByName("id")
	if uid == "" {
		user = RequestUser(r)
	} else {
		user, err = GlobalUserStore.Find(uid)
		if err != nil {
			log.Println("User", uid, "not found:", err)
			http.Redirect(w, r, "/?flash=user+not+found", http.StatusNotFound)
			return
		}
	}

	currentUser := RequestUser(r)
	if user.ID != currentUser.ID && currentUser.ID != "admin" {
		log.Println("Edit User", uid, "not allowed for user", currentUser.ID, currentUser.Username)
		http.Redirect(w, r, "/?flash=edit+user+not+allowed", http.StatusForbidden)
		return
	}

	RenderTemplate(w, r, "users/edit", map[string]interface{}{
		"Pagetitle": "EditUser",
		"User":      user,
	})
}

// HandleUserUpdate updates the user information with the new email or password information
// from the account information page
// (POST /account)
func HandleUserUpdate(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	var user *User
	var err error
	uid := params.ByName("id")
	if uid == "" {
		user = RequestUser(r)
	} else {
		user, err = GlobalUserStore.Find(uid)
		if err != nil {
			log.Println("User", uid, "not found:", err)
			http.Redirect(w, r, "/?flash=user+not+found", http.StatusNotFound)
			return
		}
	}

	currentUser := RequestUser(r)
	if user.ID != currentUser.ID && currentUser.ID != "admin" {
		log.Println("Edit User", uid, "not allowed for user", currentUser.ID, currentUser.Username)
		http.Redirect(w, r, "/?flash=edit+user+not+allowed", http.StatusForbidden)
		return
	}

	username := r.FormValue("username")
	email := r.FormValue("email")
	currentPassword := r.FormValue("currentPassword")
	newPassword := r.FormValue("newPassword")

	u, err := UpdateUser(user, username, email, currentPassword, newPassword, currentUser.ID == "admin")
	user = &u
	if err != nil {
		if IsValidationError(err) {
			fmt.Println(err)
			RenderTemplate(w, r, "users/edit", map[string]interface{}{
				"Pagetitle": "EditUser",
				"User":      user,
				"Error":     err.Error(),
			})
			return
		}
		Logf(FatalLevel, "Error updating user: %s\n", err)
	}

	err = GlobalUserStore.Save(user)
	if err != nil {
		Logf(FatalLevel, "Error updating user in Global user store: %s\n", err)
	}

	http.Redirect(w, r, "/users/"+user.ID+"?flash=user+updated", http.StatusFound)
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
		"Pagetitle": "ListUsers",
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

	format := fmt.Sprint(r.URL.Query()["format"])
	switch format {
	case "[csv]":
		var row []string
		items := [][]string{
			{"ID", "Username", "Email", "Sessions"},
		}

		w.Header().Set("Content-Type", "text/csv")
		w.Header().Set("Content-Disposition", "attachment;filename=users.csv")
		w.WriteHeader(http.StatusOK)
		writer := csv.NewWriter(w)
		users, _ = GlobalUserStore.All()
		for _, u := range users {
			var sessions []string
			for _, s := range u.Sessions {
				sessions = append(sessions, s.ID)
			}
			row = []string{u.ID, u.Username, u.Email, strings.Join(sessions, "\n")}
			items = append(items, row)
		}
		if err := writer.WriteAll(items); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	case "[yaml]":
		w.Header().Set("Content-Type", "text/yaml")
		w.WriteHeader(http.StatusOK)
		writer := yaml.NewEncoder(w)
		if err := writer.Encode(users); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	case "[xml]":
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(http.StatusOK)
		writer := xml.NewEncoder(w)
		writer.Indent("", "    ")
		if err := writer.Encode(users); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	case "[json]":
		fallthrough
	default:
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		writer := json.NewEncoder(w)
		writer.SetIndent("", "    ")
		if err := writer.Encode(users); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

func HandleUserDELETEv1(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	user, _ := GlobalUserStore.Find(params.ByName("id"))
	if user == nil {
		log.Println("No user found to delete")
		w.WriteHeader(http.StatusNotFound)
		return
	}
	if RequestUser(r).ID == user.ID || RequestUser(r).Username == "admin" {
		for _, session := range user.Sessions {
			err := GlobalSessionStore.Delete(&session)
			if err != nil {
				log.Println("Unable to delete session", session, ":", err)
			}
		}
		userconf, _ := GlobalUserConfigStore.Find(user.ID)
		if userconf != nil {
			err := GlobalUserConfigStore.Delete(userconf)
			if err != nil {
				log.Println("Unable to delete user", user, ":", err)
			}
		}
		err := GlobalUserStore.Delete(user)
		if err != nil {
			log.Println("Unable to delete user", user, ":", err)
		}
	} else {
		log.Println("Access forbidden:", RequestUser(r).ID, "!=", user.ID, "|| admin !=", user.Username)
		w.WriteHeader(http.StatusForbidden)
	}
	w.WriteHeader(http.StatusNoContent)
}

/****************************************
***  Storage Backends                 ***
*****************************************/

// UserStore is an abstraction interface to allow multiple data sources to save user info to
type UserStore interface {
	Find(string) (*User, error)
	All() ([]User, error)
	FindByEmail(string) (*User, error)
	FindByUsername(string) (*User, error)
	Save(*User) error
	Delete(*User) error
}

// GlobalUserStore is the Global Database of users
var GlobalUserStore UserStore

/**********************************
***  File User Store            ***
***********************************/

// FileUserStore is an implementation of UserStore to save user data to the filesystem
type FileUserStore struct {
	filename string
	Users    map[string]User
}

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

// All returns  a list of all users, except the HashedPassword field
func (store FileUserStore) All() ([]User, error) {
	var userlist []User
	for _, v := range store.Users {
		v.HashedPassword = ""
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

func (store FileUserStore) Delete(user *User) error {
	delete(store.Users, user.ID)
	contents, err := yaml.Marshal(store)
	if err != nil {
		return err
	}

	return os.WriteFile(store.filename, contents, 0660)
}

/**********************************
***  DB User Store              ***
***********************************/

// DBUserStore is an implementation of UserStore to save user data in the database
type DBUserStore struct {
	db *sql.DB
}

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
		Logf(FatalLevel, "Unable to create users table in database: %s\n", err)
	}

	_, err = GlobalPostgresDB.Exec(`
CREATE INDEX IF NOT EXISTS username_idx ON users( username );`)
	if err != nil {
		Logf(FatalLevel, "Unable to create users table username index in database: %s\n", err)
	}

	_, err = GlobalPostgresDB.Exec(`
CREATE INDEX IF NOT EXISTS email_idx ON users( email );`)
	if err != nil {
		Logf(FatalLevel, "Unable to create users table email index in database: %s\n", err)
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

// All returns  a list of all users, except the HashedPassword field
func (store DBUserStore) All() ([]User, error) {
	rows, err := store.db.Query(
		`
		SELECT id, username, email
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

func (store DBUserStore) Delete(user *User) error {
	row, err := store.db.Exec(
		`
		DELETE FROM users
		WHERE id = $1`,
		user.ID,
	)
	if err != nil {
		log.Println(row.RowsAffected())
	}
	return err
}
