package webapp

import (
	"database/sql"
	"encoding/json"
	"github.com/julienschmidt/httprouter"
	"gopkg.in/yaml.v3"
	"log"
	"net/http"
	"os"
)

type UserConfig struct {
	UserID   string `json:"userID" yaml:"userID"`
	Language string `json:"language" yaml:"language"`
	DarkMode bool   `json:"darkMode" yaml:"darkMode"`
}

// NewUserConfig creates a new UserConfig
func NewUserConfig(userid, language string, darkmode bool) (UserConfig, error) {
	userconfig := UserConfig{
		UserID:   userid,
		Language: language,
		DarkMode: darkmode,
	}

	return userconfig, nil
}

// FindUserConfig returns the user config with the given userid if found
func FindUserConfig(userid string) (*UserConfig, error) {
	// create dummy user to return username if login fails
	out := &UserConfig{
		UserID: userid,
	}

	// find user or return dummy with error message if it fails
	existingUserConfig, err := GlobalUserConfigStore.Find(userid)
	if err != nil {
		return out, err
	}
	if existingUserConfig == nil {
		return out, nil
	}

	return existingUserConfig, nil
}

// UpdateUserConfig updates the UserConfig's email address and, if the current password matches, the password
func UpdateUserConfig(userconfig *UserConfig, language string, darkmode bool) (UserConfig, error) {
	// make a shallow copy of the user and set email
	out := *userconfig
	out.Language = language
	out.DarkMode = darkmode

	// update userconfig
	userconfig.Language = language
	userconfig.DarkMode = darkmode

	return out, nil
}

// GetLanguage returns the language from the lang url parameter, the users config for the given userid or from
// the current request r or "en" if both are not found
func GetLanguage(userid string, r *http.Request, params httprouter.Params) string {
	// if lang is set with the url it takes precedence
	lang := params.ByName("lang")
	if lang != "" {
		return lang
	}

	// get user
	var uid string
	if userid != "" {
		uid = userid
	} else {
		u := RequestUser(r)
		if u != nil {
			uid = u.ID
		}
	}

	// get user config
	if uid != "" {
		set, err := GlobalUserConfigStore.Find(uid)
		if err != nil {
			log.Println("Can't find userconfig of user", uid, "in global user config store:", err)
		}
		if set != nil {
			lang = set.Language
		}
	}

	if lang != "" {
		return lang
	}

	return "en"
}

/****************************************
***  Handler                          ***
*****************************************/

// HandleUserConfigEdit shows the account information page to change email or password
// (GET /account)
func HandleUserConfigEdit(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	userconfig, err := GlobalUserConfigStore.Find(params.ByName("userid"))
	if err != nil {
		*userconfig, _ = NewUserConfig(params.ByName("userid"), "en", false)
	}
	RenderTemplate(w, r, "userconfigs/edit", map[string]interface{}{
		"Pagetitle":  "EditSettings",
		"UserConfig": userconfig,
	})
}

// HandleUserConfigUpdate updates the user information with the new email or password information
// from the account information page
// (POST /account)
func HandleUserConfigUpdate(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	user := RequestUser(r)
	if user == nil {
		log.Fatal("User from session not found")
	}

	currentUserconfig, err := GlobalUserConfigStore.Find(params.ByName("userid"))
	if currentUserconfig == nil {
		conf, _ := NewUserConfig(user.ID, "en", false)
		currentUserconfig = &conf
	}
	language := r.FormValue("language")
	var darkmode bool

	if r.FormValue("darkmode") == "dark" {
		darkmode = true
	}

	userconfig, err := UpdateUserConfig(currentUserconfig, language, darkmode)
	if err != nil {
		if IsValidationError(err) {
			RenderTemplate(w, r, "userconfigs/edit", map[string]interface{}{
				"Pagetitle":  "EditSettings",
				"UserConfig": userconfig,
				"Error":      err.Error(),
			})
			return
		}
		log.Fatalf("Error updating user config: %s\n", err)
	}

	err = GlobalUserConfigStore.Save(currentUserconfig)
	if err != nil {
		log.Fatalf("Error updating user config in Global user config store: %s\n", err)
	}

	http.Redirect(w, r, "/?flash=settings+updated", http.StatusFound)
}

func HandleUserConfigGETv1(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	userid := params.ByName("id")
	if userid == "" {
		userid = RequestUser(r).ID
	}

	userconfig, err := GlobalUserConfigStore.Find(userid)
	if err != nil {
		*userconfig, _ = NewUserConfig(userid, "en", false)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	writer := json.NewEncoder(w)
	writer.SetIndent("", "    ")
	if err := writer.Encode(userconfig); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

/****************************************
***  Storage Backends                 ***
*****************************************/

// UserConfigStore is an abstraction interface to allow multiple data sources to save user info to
type UserConfigStore interface {
	Find(string) (*UserConfig, error)
	Save(*UserConfig) error
	Delete(config *UserConfig) error
}

// GlobalUserConfigStore is the Global Database of users
var GlobalUserConfigStore UserConfigStore

/**********************************
***  File UserConfig Store      ***
***********************************/

// FileUserConfigStore is an implementation of UserConfigStore to save user data to the filesystem
type FileUserConfigStore struct {
	filename    string
	UserConfigs map[string]UserConfig
}

// NewFileUserConfigStore creates a new FileUserConfigStore under the given filename
func NewFileUserConfigStore(filename string) (*FileUserConfigStore, error) {
	store := &FileUserConfigStore{
		UserConfigs: map[string]UserConfig{},
		filename:    filename,
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

// Save adds a new user and saves the GlobalUserConfigStore to the Filesystem
func (store FileUserConfigStore) Save(userconfig *UserConfig) error {
	store.UserConfigs[userconfig.UserID] = *userconfig

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

func (store FileUserConfigStore) All() ([]UserConfig, error) {
	var userlist []UserConfig
	for _, v := range store.UserConfigs {
		userlist = append(userlist, v)
	}
	return userlist, nil
}

// Find returns the userconfig with the given userid if found
func (store FileUserConfigStore) Find(userid string) (*UserConfig, error) {
	userconfig, ok := store.UserConfigs[userid]
	if ok {
		return &userconfig, nil
	}
	return nil, nil
}

func (store *FileUserConfigStore) Delete(userconf *UserConfig) error {
	delete(store.UserConfigs, userconf.UserID)
	contents, err := yaml.Marshal(store)
	if err != nil {
		return err
	}

	return os.WriteFile(store.filename, contents, 0660)
}

/**********************************
***  DB UserConfig Store              ***
***********************************/

// DBUserConfigStore is an implementation of UserConfigStore to save user data in the database
type DBUserConfigStore struct {
	db *sql.DB
}

func NewDBUserConfigStore() UserConfigStore {
	_, err := GlobalPostgresDB.Exec(`
CREATE TABLE IF NOT EXISTS userconfigs (
  userid varchar(255) NOT NULL DEFAULT '',
  language varchar(2) NOT NULL DEFAULT '',
  darkmode boolean NOT NULL DEFAULT FALSE,
  PRIMARY KEY (userid)
);
`)
	if err != nil {
		log.Fatalf("Unable to create userconfigs table in database: %s\n", err)
	}

	return &DBUserConfigStore{
		db: GlobalPostgresDB,
	}
}

func (store DBUserConfigStore) Save(userconfig *UserConfig) error {
	_, err := store.db.Exec(
		`
	INSERT INTO userconfigs
	    (userid, language, darkmode)
	    VALUES ($1, $2, $3)
	    ON CONFLICT (userid) DO UPDATE SET language=excluded.language, darkmode=excluded.darkmode`,
		userconfig.UserID,
		userconfig.Language,
		userconfig.DarkMode,
	)
	return err
}

func (store DBUserConfigStore) Find(userid string) (*UserConfig, error) {
	row := store.db.QueryRow(
		`
		SELECT userid, language, darkmode
		FROM userconfigs
		WHERE userid = $1`,
		userid,
	)

	userconfig := UserConfig{}
	err := row.Scan(
		&userconfig.UserID,
		&userconfig.Language,
		&userconfig.DarkMode,
	)
	// return nil and no error when the Scan returns no findings
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &userconfig, err
}

func (store DBUserConfigStore) Delete(userconfig *UserConfig) error {
	row, err := store.db.Exec(
		`
		DELETE FROM userconfigs
		WHERE userid = $1`,
		userconfig.UserID,
	)
	if err != nil {
		log.Println(row)
	}
	return err
}
