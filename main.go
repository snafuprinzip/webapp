package main

import (
	"flag"
	"github.com/julienschmidt/httprouter"
	"log"
	"net/http"
	"os"
	"path"
)

const Debug = true

// Creates a new.html router
func NewRouter() *httprouter.Router {
	router := httprouter.New()
	router.NotFound = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
	return router
}

func CreateStores() {
	// setup global user store
	if Config.DBConnector == "" || Config.DBConnector == "files" {
		// DBConnector isn't set or set to files, so we use the filesystem storage backend
		// and write yaml files to the data directory
		userstore, err := NewFileUserStore(path.Join(Config.DataDirectory, "users.yaml"))
		if err != nil {
			log.Fatalf("Error creating user store: %s\n", err)
		}
		globalUserStore = userstore

		sessionstore, err := NewFileSessionStore(path.Join(Config.DataDirectory, "sessions.yaml"))
		if err != nil {
			log.Fatalf("Error creating session store: %s\n", err)
		}
		globalSessionStore = sessionstore
	} else { // DBConnector is set, so we use the database backend
		globalUserStore = NewDBUserStore()
		globalSessionStore = NewDBSessionStore()
	}
	log.Println("Backend Storages created")

}

func main() {
	var configfile string

	// get command line arguments
	flag.StringVar(&configfile, "config", "./config.yaml", "Path to configuration file")
	flag.Parse()

	// read configuration from configfile
	if err := ReadConfig(configfile); err != nil {
		if !os.IsNotExist(err) { // Error for non-existent config file has already been covered in ReadConfig
			log.Printf("%s\n", err)
		}
	}

	// setup database
	db, err := NewMySQLDB(Config.DBConnector)
	if err != nil {
		log.Fatalf("Error creating new Database: %s\n", err)
	}
	globalMySQLDB = db

	// Create Data Stores
	CreateStores()

	// setup the public multiplexer
	router := NewRouter()
	router.Handle("GET", "/", HandleHome)

	if Config.OpenRegistration {
		router.Handle("GET", "/register", HandleUserNew)
		router.Handle("POST", "/register", HandleUserCreate)
	}
	router.Handle("GET", "/login", HandleSessionNew)
	router.Handle("POST", "/login", HandleSessionCreate)
	router.ServeFiles("/assets/*filepath", http.Dir("assets/"))

	// setup the mux for authenticated users
	secureRouter := NewRouter()
	secureRouter.Handle("GET", "/signout", HandleSessionDestroy)
	secureRouter.Handle("GET", "/account", HandleUserEdit)
	secureRouter.Handle("POST", "/account", HandleUserUpdate)

	// add middleware handlers
	middleware := Middleware{}
	middleware.Add(router)
	middleware.Add(http.HandlerFunc(RequireLogin))
	middleware.Add(secureRouter)

	// listen and serve
	log.Println("starting listener")
	log.Fatal(http.ListenAndServe(Config.BindAddress, middleware))
}
