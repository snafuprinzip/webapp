package main

import (
	"flag"
	"github.com/julienschmidt/httprouter"
	"github.com/snafuprinzip/webappskeleton"
	"github.com/snafuprinzip/webappskeleton/webapp"
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
	// setup Global user store
	if webappskeleton.Config.DBConnector == "" || webappskeleton.Config.DBConnector == "files" {
		// DBConnector isn't set or set to files, so we use the filesystem storage backend
		// and write yaml files to the data directory
		userstore, err := webapp.NewFileUserStore(path.Join(webappskeleton.Config.DataDirectory, "users.yaml"))
		if err != nil {
			log.Fatalf("Error creating user store: %s\n", err)
		}
		webapp.GlobalUserStore = userstore

		sessionstore, err := webapp.NewFileSessionStore(path.Join(webappskeleton.Config.DataDirectory, "sessions.yaml"))
		if err != nil {
			log.Fatalf("Error creating session store: %s\n", err)
		}
		webapp.GlobalSessionStore = sessionstore
	} else { // DBConnector is set, so we use the database backend
		// setup database
		db, err := webapp.NewMySQLDB(webappskeleton.Config.DBConnector)
		if err != nil {
			log.Fatalf("Error creating new Database: %s\n", err)
		}
		webapp.GlobalMySQLDB = db

		webapp.GlobalUserStore = webapp.NewDBUserStore()
		webapp.GlobalSessionStore = webapp.NewDBSessionStore()
	}
	log.Println("Backend Storages created")

}

func main() {
	var configfile string

	// get command line arguments
	flag.StringVar(&configfile, "config", "./config.yaml", "Path to configuration file")
	flag.Parse()

	// read configuration from configfile
	if err := webappskeleton.ReadConfig(configfile); err != nil {
		if !os.IsNotExist(err) { // Error for non-existent config file has already been covered in ReadConfig
			log.Printf("%s\n", err)
		}
	}

	// Create Data Stores
	CreateStores()

	// setup the public multiplexer
	router := NewRouter()
	router.Handle("GET", "/", webapp.HandleHome)

	if webappskeleton.Config.OpenRegistration {
		router.Handle("GET", "/register", webapp.HandleUserNew)
		router.Handle("POST", "/register", webapp.HandleUserCreate)
	}
	router.Handle("GET", "/login", webapp.HandleSessionNew)
	router.Handle("POST", "/login", webapp.HandleSessionCreate)
	router.ServeFiles("/assets/*filepath", http.Dir("assets/"))

	// setup the mux for authenticated users
	secureRouter := NewRouter()
	secureRouter.Handle("GET", "/signout", webapp.HandleSessionDestroy)
	secureRouter.Handle("GET", "/account", webapp.HandleUserEdit)
	secureRouter.Handle("POST", "/account", webapp.HandleUserUpdate)

	// add middleware handlers
	middleware := webapp.Middleware{}
	middleware.Add(router)
	middleware.Add(http.HandlerFunc(webapp.RequireLogin))
	middleware.Add(secureRouter)

	// listen and serve
	log.Println("starting listener")
	log.Fatal(http.ListenAndServe(webappskeleton.Config.BindAddress, middleware))
}
