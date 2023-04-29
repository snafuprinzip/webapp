package main

import (
	"flag"
	"github.com/julienschmidt/httprouter"
	"github.com/snafuprinzip/webapp"
	"log"
	"net/http"
	"os"
	"path"
)

//const Debug = true

// NewRouter creates a new html router
func NewRouter() *httprouter.Router {
	router := httprouter.New()
	router.NotFound = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
	return router
}

func main() {
	var configfile string

	// get command line arguments
	flag.StringVar(&configfile, "config", "./config/config.yaml", "Path to configuration file")
	flag.Parse()

	// read configuration from configfile
	if err := webapp.ReadConfig(configfile); err != nil {
		if !os.IsNotExist(err) { // Error for non-existent config file has already been covered in ReadConfig
			log.Printf("%s\n", err)
		}
	}

	webapp.NewApp("WebApp")

	// Create Data Stores
	// setup Global user store
	if webapp.Config.DBConnector == "" || webapp.Config.DBConnector == "files" {
		// DBConnector isn't set or set to files, so we use the filesystem storage backend
		// and write yaml files to the data directory
		userstore, err := webapp.NewFileUserStore(path.Join(webapp.Config.DataDirectory, "users.yaml"))
		if err != nil {
			log.Fatalf("Error creating user store: %s\n", err)
		}
		webapp.GlobalUserStore = userstore

		userconfigstore, err := webapp.NewFileUserConfigStore(path.Join(webapp.Config.DataDirectory, "userconfigs.yaml"))
		if err != nil {
			log.Fatalf("Error creating userconfigs store: %s\n", err)
		}
		webapp.GlobalUserConfigStore = userconfigstore

		sessionstore, err := webapp.NewFileSessionStore(path.Join(webapp.Config.DataDirectory, "sessions.yaml"))
		if err != nil {
			log.Fatalf("Error creating session store: %s\n", err)
		}
		webapp.GlobalSessionStore = sessionstore
	} else { // DBConnector is set, so we use the database backend
		// setup database
		db, err := webapp.NewPostgresDB(webapp.Config.DBConnector)
		defer db.Close()

		if err != nil {
			log.Fatalf("Error creating new Database: %s\n", err)
		}
		webapp.GlobalPostgresDB = db

		webapp.GlobalUserStore = webapp.NewDBUserStore()
		webapp.GlobalUserConfigStore = webapp.NewDBUserConfigStore()
		webapp.GlobalSessionStore = webapp.NewDBSessionStore()
	}
	log.Println("Backend Storages created")

	// Create Admin account if needed
	webapp.CreateAdminAccount()
	webapp.ReadPagetitles()

	// setup the public multiplexer
	router := NewRouter()
	router.GET("/", webapp.HandleHome)

	if webapp.Config.OpenRegistration {
		router.GET("/register", webapp.HandleUserNew)
		router.POST("/register", webapp.HandleUserCreate)
	}
	router.GET("/login", webapp.HandleSessionNew)
	router.POST("/login", webapp.HandleSessionCreate)
	router.ServeFiles("/assets/*filepath", http.Dir("assets/"))

	// set up the mux for authenticated users
	secureRouter := NewRouter()
	secureRouter.GET("/signout", webapp.HandleSessionDestroy)
	secureRouter.GET("/account", webapp.HandleUserEdit)
	secureRouter.POST("/account", webapp.HandleUserUpdate)
	secureRouter.GET("/settings", webapp.HandleUserConfigEdit)
	secureRouter.POST("/settings", webapp.HandleUserConfigUpdate)
	secureRouter.GET("/api/v1/settings", webapp.HandleUserConfigGETv1)

	adminRouter := NewRouter()
	adminRouter.GET("/users", webapp.HandleUsersIndex)
	adminRouter.GET("/api/v1/users", webapp.HandleUsersGETv1)
	adminRouter.DELETE("/api/v1/users/:id", webapp.HandleUserDELETEv1)
	adminRouter.GET("/api/v1/settings/:id", webapp.HandleUserConfigGETv1)

	// add middleware handlers
	middleware := webapp.Middleware{}
	middleware.Add(router)
	middleware.Add(http.HandlerFunc(webapp.RequireLogin))
	middleware.Add(secureRouter)
	middleware.Add(http.HandlerFunc(webapp.RequireAdmin))
	middleware.Add(adminRouter)

	// listen and serve
	log.Println("starting listener")
	log.Fatal(http.ListenAndServe(webapp.Config.BindAddress, middleware))
}
