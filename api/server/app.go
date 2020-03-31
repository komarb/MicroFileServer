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

	// Create db connect
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	err = client.Connect(ctx)
	if err != nil {
		log.WithFields(log.Fields{
			"function" : "mongo.Connect",
			"error"	:	err},
		).Fatal("Failed to connect to MongoDB")
	}

	// Check the connection
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
	a.Router = mux.NewRouter()
	a.setRouters()
}

func (a *App) setRouters() {
	/*if cfg.App.TestMode {
		a.Router.Use(testAuthMiddleware)
	} else {
		a.Router.Use(authMiddleware)
	}*/
	a.Router.HandleFunc("/download/{id}", downloadFile).Methods("GET")
	a.Router.HandleFunc("/upload", uploadFile).Methods("POST")
	a.Router.HandleFunc("/files/{id}", deleteFile).Methods("DELETE")
	a.Router.HandleFunc("/files", getFilesListForUser).Methods("GET").Queries("user","{user}")
	a.Router.HandleFunc("/files", getFilesList).Methods("GET")

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

