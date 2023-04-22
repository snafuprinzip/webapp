package webapp

import (
	"database/sql"
	"github.com/julienschmidt/httprouter"
	"gopkg.in/yaml.v3"
	"log"
	"net/http"
	"os"
)

type UserConfig struct {
	UserID   string
	Language string
	DarkMode bool
}

// UserConfigStore is an abstraction interface to allow multiple data sources to save user info to
type UserConfigStore interface {
	Find(string) (*UserConfig, error)
	Save(*UserConfig) error
}

// FileUserConfigStore is an implementation of UserConfigStore to save user data to the filesystem
type FileUserConfigStore struct {
	filename    string
	UserConfigs map[string]UserConfig
}

// DBUserConfigStore is an implementation of UserConfigStore to save user data in the database
type DBUserConfigStore struct {
	db *sql.DB
}

// GlobalUserConfigStore is the Global Database of users
var GlobalUserConfigStore UserConfigStore

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
		"Pagetitle":  "Edit Config",
		"UserConfig": userconfig,
	})
}

// HandleUserConfigUpdate updates the user information with the new email or password information
// from the account information page
// (POST /account)
func HandleUserConfigUpdate(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	currentUserconfig, err := GlobalUserConfigStore.Find(params.ByName("userid"))
	language := r.FormValue("language")
	var darkmode bool

	if r.FormValue("darkmode") == "true" {
		darkmode = true
	}

	userconfig, err := UpdateUserConfig(currentUserconfig, language, darkmode)
	if err != nil {
		if IsValidationError(err) {
			RenderTemplate(w, r, "users/edit", map[string]interface{}{
				"Pagetitle":  "Edit UserConfig",
				"UserConfig": userconfig,
				"Error":      err.Error(),
			})
			return
		}
		log.Fatalf("Error updating user: %s\n", err)
	}

	err = GlobalUserConfigStore.Save(currentUserconfig)
	if err != nil {
		log.Fatalf("Error updating user in Global user store: %s\n", err)
	}

	http.Redirect(w, r, "/account?flash=user+updated", http.StatusFound)
}

/****************************************
***  Storage Backends                 ***
*****************************************/

/**********************************
***  File UserConfig Store            ***
***********************************/

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

/**********************************
***  DB UserConfig Store              ***
***********************************/

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
	    ON CONFLICT DO UPDATE SET userid=$1, language=$2, darkmode=$3`,
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
