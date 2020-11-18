package server

import (
	"MicroFileServer/config"
	"context"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"net/http"
	"time"
)

type App struct {
	Router *mux.Router
	DB *mongo.Client
}

var collection *mongo.Collection
var db *mongo.Database
var cfg *config.Config

func (a *App) Init(config *config.Config) {
	cfg = config
	log.Info("Little Big File Server is starting up!")
	DBUri := "mongodb://" + cfg.DB.Host + ":" + cfg.DB.DBPort
	log.WithField("dburi", DBUri).Info("Current database URI: ")
	client, err := mongo.NewClient(options.Client().ApplyURI(DBUri))
	if err != nil {
		log.WithFields(log.Fields{
			"function" : "mongo.NewClient",
			"error"	:	err,
			"db_uri":	DBUri,
		},
		).Fatal("Failed to create new MongoDB client")
	}

	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	err = client.Connect(ctx)
	if err != nil {
		log.WithFields(log.Fields{
			"function" : "mongo.Connect",
			"error"	:	err},
		).Fatal("Failed to connect to MongoDB")
	}

	ctx, _ = context.WithTimeout(context.Background(), 10*time.Second)
	err = client.Ping(ctx, nil)
	if err != nil {
		log.WithFields(log.Fields{
			"function" : "mongo.Ping",
			"error"	:	err},
		).Fatal("Failed to ping MongoDB")
	}
	log.Info("Connected to MongoDB!")
	log.WithFields(log.Fields{
		"db_name" : cfg.DB.DBName,
		"collection_name" : cfg.DB.CollectionName,
	}).Info("Database information: ")
	log.WithField("testMode", cfg.App.TestMode).Info("Let's check if test mode is on...")

	collection = client.Database(cfg.DB.DBName).Collection(cfg.DB.CollectionName)
	db = client.Database(cfg.DB.DBName)
	a.Router = mux.NewRouter().PathPrefix(cfg.App.PathPrefix).Subrouter()
	a.setRouters()
}

func (a *App) setRouters() {
	public := a.Router.PathPrefix("/download").Subrouter()
	public.Use(loggingMiddleware)
	public.HandleFunc("/{id}", downloadFile).Methods("GET")


	private := a.Router.PathPrefix("/files").Subrouter()
	if cfg.App.TestMode {
		private.Use(testAuthMiddleware)
	} else {
		private.Use(authMiddleware)
	}
	private.HandleFunc("/upload", uploadFile).Methods("POST", "OPTIONS")
	private.HandleFunc("/{id}", deleteFile).Methods("DELETE", "OPTIONS")
	private.HandleFunc("/{id}", getFile).Methods("GET", "OPTIONS")
	private.HandleFunc("", getFilesListForUser).Methods("GET", "OPTIONS").Queries("user","{user}")
	private.HandleFunc("", getFilesList).Methods("GET", "OPTIONS").Queries("sorted_by", "{sortVar}")
}

func (a *App) Run(addr string) {
	log.WithField("port", cfg.App.AppPort).Info("Starting server on port:")
	log.Info("\n\nNow handling routes!")

	err := http.ListenAndServe(addr, a.Router)
	if err != nil {
		log.WithFields(log.Fields{
			"function" : "http.ListenAndServe",
			"error"	:	err},
		).Fatal("Failed to run a server!")
	}
}

func init() {
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp: true,
	})
}

