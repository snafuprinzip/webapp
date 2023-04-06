package main

import (
	"database/sql"
	"github.com/go-yaml/yaml"
	"github.com/julienschmidt/httprouter"
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
	sessionDuration   = 3 * 24 * time.Hour
	sessionCookieName = "WebApp"
	sessionIDLength   = 20
)

type SessionStore interface {
	Find(string) (*Session, error)
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

var globalSessionStore SessionStore // Session Database

func NewSession(w http.ResponseWriter) *Session {
	expiry := time.Now().Add(sessionDuration)

	session := &Session{
		ID:     GenerateID("sess", sessionIDLength),
		Expiry: expiry,
	}

	cookie := http.Cookie{
		Name:    sessionCookieName,
		Value:   session.ID,
		Expires: expiry,
	}

	http.SetCookie(w, &cookie)
	return session
}

func RequestSession(r *http.Request) *Session {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil {
		return nil
	}

	session, err := globalSessionStore.Find(cookie.Value)
	if err != nil {
		log.Fatalf("Error accessing global session store: %s\n", err)
	}

	if session == nil {
		return nil
	}

	if session.Expired() {
		globalSessionStore.Delete(session)
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

	user, err := globalUserStore.Find(session.UserID)
	if err != nil {
		if err != sql.ErrNoRows {
			log.Fatalf("Error accessing global user store: %s\n", err)
		}
	}

	return user
}

func RequireLogin(w http.ResponseWriter, r *http.Request) {
	// Let request pass if user is found
	if RequestUser(r) != nil {
		return
	}

	query := url.Values{}
	query.Add("next", url.QueryEscape(r.URL.String()))

	http.Redirect(w, r, "/login?"+query.Encode(), http.StatusFound)
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
		err := globalSessionStore.Delete(session)
		if err != nil {
			log.Fatalf("Error deleting session from glocal session store: %s\n", err)
		}
	}
	RenderTemplate(w, r, "sessions/destroy", nil)
}

func HandleSessionNew(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	next := r.URL.Query().Get("next")
	RenderTemplate(w, r, "sessions/new", map[string]interface{}{
		"Next": next,
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
				"User":  user,
				"Error": err,
				"Next":  next,
			})
			return
		}
		log.Fatalf("Error finding user/password combination: %s\n", err)
	}

	// find an existing session for the now authenticated user or create a new one
	session := FindOrCreateSession(w, r)
	session.UserID = user.ID
	err = globalSessionStore.Save(session)
	if err != nil {
		log.Fatalf("Error adding new session to global session store: %s\n", err)
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
	schema, err := os.ReadFile("sql/sessions_table.sql")
	if err != nil {
		log.Fatalf("Unable to read sql schema for the sessions table: %s\n", err)
	}

	_, err = globalMySQLDB.Exec(string(schema))
	if err != nil {
		log.Fatalf("Unable to create schema in database: %s\n", err)
	}

	return &DBSessionStore{
		db: globalMySQLDB,
	}
}

func (store DBSessionStore) Save(session *Session) error {
	_, err := store.db.Exec(
		`
	REPLACE INTO sessions
	    (id, userid, expiry)
	    VALUES (?, ?, ?)`,
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
		WHERE id = ?`,
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

func (store DBSessionStore) Delete(session *Session) error {
	row, err := store.db.Exec(
		`
		DELETE FROM sessions
		WHERE id = ?`,
		session.ID,
	)
	if err != nil {
		log.Println(row)
	}

	return err
}
