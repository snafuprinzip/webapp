package webapp

import (
	"database/sql"
	"github.com/julienschmidt/httprouter"
	"gopkg.in/yaml.v3"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"
)

type Session struct {
	ID     string
	UserID string
	Expiry time.Time
}

const (
	sessionDuration = 3 * 24 * time.Hour
	sessionIDLength = 20
)

type SessionStore interface {
	Find(string) (*Session, error)
	FindByUser(string) ([]Session, error)
	Save(*Session) error
	Delete(*Session) error
}

type FileSessionStore struct {
	filename string
	Sessions map[string]Session
}

type DBSessionStore struct {
	db *sql.DB
}

var GlobalSessionStore SessionStore // Session Database

func NewSession(w http.ResponseWriter) *Session {
	expiry := time.Now().Add(sessionDuration)

	session := &Session{
		ID:     GenerateID("sess", sessionIDLength),
		Expiry: expiry,
	}

	cookie := http.Cookie{
		Name:    appName,
		Value:   session.ID,
		Expires: expiry,
	}

	http.SetCookie(w, &cookie)
	return session
}

func RequestSession(r *http.Request) *Session {
	cookie, err := r.Cookie(appName)
	if err != nil {
		return nil
	}

	session, err := GlobalSessionStore.Find(cookie.Value)
	if err != nil {
		log.Fatalf("Error accessing Global session store: %s\n", err)
	}

	if session == nil {
		return nil
	}

	if session.Expired() {
		err = GlobalSessionStore.Delete(session)
		if err != nil {
			log.Println("Unable to delete session from global session store:", err)
		}
		return nil
	}

	return session
}

func (s *Session) Expired() bool {
	return s.Expiry.Before(time.Now())
}

func RequestUser(r *http.Request) *User {
	session := RequestSession(r)
	if session == nil || session.UserID == "" {
		return nil
	}

	user, err := GlobalUserStore.Find(session.UserID)
	if err != nil {
		if err != sql.ErrNoRows {
			log.Fatalf("Error accessing Global user store: %s\n", err)
		}
	}

	return user
}

// IsAdmin checks if the current user is logged in with an admin account
func IsAdmin(r *http.Request) bool {
	user := RequestUser(r)
	if user != nil && user.ID == "admin" && user.Username == "admin" {
		return true
	}
	return false
}

// RequireLogin checks if the user is logged in
func RequireLogin(w http.ResponseWriter, r *http.Request) {
	// Let request pass if user is found
	if RequestUser(r) != nil {
		return
	}

	query := url.Values{}
	query.Add("next", url.QueryEscape(r.URL.String()))

	http.Redirect(w, r, "/login?"+query.Encode(), http.StatusFound)
}

// RequireAdmin checks if the user is logged in with an admin account
func RequireAdmin(w http.ResponseWriter, r *http.Request) {
	// Let request pass if admin user is found
	if IsAdmin(r) {
		return
	}

	query := url.Values{}
	query.Add("next", url.QueryEscape(r.URL.String()))

	http.Redirect(w, r, "/?"+query.Encode(), http.StatusFound)
}

func FindOrCreateSession(w http.ResponseWriter, r *http.Request) *Session {
	session := RequestSession(r)
	if session == nil {
		session = NewSession(w)
	}
	return session
}

/****************************************
***  Handler                          ***
*****************************************/

func HandleSessionDestroy(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	session := RequestSession(r)
	if session != nil {
		err := GlobalSessionStore.Delete(session)
		if err != nil {
			log.Fatalf("Error deleting session from glocal session store: %s\n", err)
		}
	}
	RenderTemplate(w, r, "sessions/destroy", nil)
}

func HandleSessionNew(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	next := r.URL.Query().Get("next")
	RenderTemplate(w, r, "sessions/new", map[string]interface{}{
		"Pagetitle": "Login",
		"Next":      next,
	})
}

func HandleSessionCreate(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	// extract form values
	username := r.FormValue("username")
	password := r.FormValue("password")
	next := r.FormValue("next")

	// find user or show login form and error message
	user, err := FindUser(username, password)
	if err != nil {
		if IsValidationError(err) {
			RenderTemplate(w, r, "sessions/new", map[string]interface{}{
				"Pagetitle": "Login",
				"User":      user,
				"Error":     err,
				"Next":      next,
			})
			return
		}
		log.Fatalf("Error finding user/password combination: %s\n", err)
	}

	// find an existing session for the now authenticated user or create a new one
	session := FindOrCreateSession(w, r)
	session.UserID = user.ID
	err = GlobalSessionStore.Save(session)
	if err != nil {
		log.Fatalf("Error adding new session to Global session store: %s\n", err)
	}

	if next == "" {
		next = "/"
	}

	http.Redirect(w, r, next+"?flash=Angemeldet", http.StatusFound)
}

/****************************************
***  Storage Backends                 ***
*****************************************/

/**********************************
***  File Session Store         ***
***********************************/

func NewFileSessionStore(name string) (*FileSessionStore, error) {
	store := &FileSessionStore{
		Sessions: map[string]Session{},
		filename: name,
	}

	contents, err := os.ReadFile(name)
	if err != nil {
		if os.IsNotExist(err) {
			return store, nil
		}
		return nil, err
	}

	err = yaml.Unmarshal(contents, store)
	if err != nil {
		return nil, err
	}
	return store, err
}

func (s *FileSessionStore) Find(id string) (*Session, error) {
	session, exists := s.Sessions[id]
	if !exists {
		return nil, nil
	}
	return &session, nil
}

func (s *FileSessionStore) FindByUser(userid string) ([]Session, error) {
	var sessions []Session
	for _, session := range s.Sessions {
		log.Printf("session.UserID (%s) == userid (%s)\n", session.UserID, userid)
		if session.UserID == userid {
			sessions = append(sessions, session)
		}
	}
	return sessions, nil
}

func (s *FileSessionStore) Save(session *Session) error {
	s.Sessions[session.ID] = *session
	contents, err := yaml.Marshal(s)
	if err != nil {
		return err
	}

	return os.WriteFile(s.filename, contents, 0660)
}

func (s *FileSessionStore) Delete(session *Session) error {
	delete(s.Sessions, session.ID)
	contents, err := yaml.Marshal(s)
	if err != nil {
		return err
	}

	return os.WriteFile(s.filename, contents, 0660)
}

/**********************************
***  DB Session Store           ***
***********************************/

func NewDBSessionStore() SessionStore {
	_, err := GlobalPostgresDB.Exec(`
CREATE TABLE IF NOT EXISTS sessions (
  id varchar(255) NOT NULL DEFAULT '',
  userid varchar(255) NOT NULL DEFAULT '',
  expiry timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (id)
);
`)
	if err != nil {
		log.Fatalf("Unable to create sessions table in database: %s\n", err)
	}

	_, err = GlobalPostgresDB.Exec(`
CREATE INDEX IF NOT EXISTS userid_idx ON sessions( userid );`)
	if err != nil {
		log.Fatalf("Unable to create userid index in sessions table of the database: %s\n", err)
	}

	return &DBSessionStore{
		db: GlobalPostgresDB,
	}
}

func (store DBSessionStore) Save(session *Session) error {
	_, err := store.db.Exec(
		`
	INSERT INTO sessions
	    (id, userid, expiry)
	    VALUES ($1, $2, $3)
	    ON CONFLICT (id)
	    DO UPDATE SET id=$1, userid=$2, expiry=$3`,
		session.ID,
		session.UserID,
		session.Expiry,
	)
	return err
}

func (store DBSessionStore) Find(id string) (*Session, error) {
	row := store.db.QueryRow(
		`
		SELECT id, userid, expiry
		FROM sessions
		WHERE id = $1`,
		id,
	)

	session := Session{}
	err := row.Scan(
		&session.ID,
		&session.UserID,
		&session.Expiry,
	)
	// return nil and no error when the Scan returns no findings
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &session, err
}

func (store DBSessionStore) FindByUser(userid string) ([]Session, error) {
	rows, err := store.db.Query(
		`
		SELECT id, userid, expiry
		FROM sessions
		WHERE userid = $1
		`,
		userid,
	)
	if err != nil {
		return nil, err
	}

	var sessions []Session
	for rows.Next() {
		session := Session{}
		err := rows.Scan(
			&session.ID,
			&session.UserID,
			&session.Expiry,
		)
		if err != nil {
			return nil, err
		}

		sessions = append(sessions, session)
	}

	return sessions, nil

}

func (store DBSessionStore) Delete(session *Session) error {
	row, err := store.db.Exec(
		`
		DELETE FROM sessions
		WHERE id = $1`,
		session.ID,
	)
	if err != nil {
		log.Println(row)
	}

	return err
}
