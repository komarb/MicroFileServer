package server

import (
	"MicroFileServer/models"
	"bytes"
	"context"
	"encoding/json"
	"github.com/gabriel-vasile/mimetype"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/gridfs"
	"go.mongodb.org/mongo-driver/mongo/options"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

func downloadFile(w http.ResponseWriter, r *http.Request) {
	var downloadedFile models.File

	data := mux.Vars(r)
	objID, err := primitive.ObjectIDFromHex(data["id"])
	if err != nil {
		http.NotFound(w, r)
		return
	}
	filter := bson.M{"_id": objID}
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	err = collection.FindOne(ctx, filter).Decode(&downloadedFile)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	fileName := downloadedFile.FileName
	buf := bytes.NewBuffer(nil)
	bucket, _ := gridfs.NewBucket(db)
	dStream, err := bucket.DownloadToStream(objID, buf)
	if err != nil {
		log.WithFields(log.Fields{
			"function" : "bucket.DownloadToStream",
			"handler" : "downloadFile",
			"error"	:	err,
		},
		).Fatal("DB interaction resulted in error, shutting down...")
	}
	log.WithFields(log.Fields{
		"fileSize" : dStream,
	},
	).Info("File size to download: ")

	mime := mimetype.Detect(buf.Bytes())
	if !strings.Contains(mime.String(), "video") && !strings.Contains(mime.String(), "audio") {
		w.Header().Set("Content-Type", mime.String())
		w.Header().Set("Content-Disposition", "attachment; filename=\""+fileName+"\"")
	}
	http.ServeContent(w, r, fileName, time.Now(), bytes.NewReader(buf.Bytes()))
}

func uploadFile(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	r.Body = http.MaxBytesReader(w, r.Body, cfg.App.MaxFileSize * 1024 * 1024)
	data, handler, err := r.FormFile("uploadingForm")
	if err != nil {
		w.Write([]byte("File is not appropriate!"))
		w.WriteHeader(400)
		log.WithFields(log.Fields{
			"err" : err,
			"maxSizeInMB" : cfg.App.MaxFileSize,
		},
		).Info("File is not appropriate!")
		return
	}
	defer data.Close()
	fileBytes, err := ioutil.ReadAll(data)
	if err != nil {
		log.WithFields(log.Fields{
			"function" : "ioutil.ReadAll(data)",
			"handler" : "uploadFile",
			"error"	:	err,
		},
		).Warn("Can't read file data!")
		return
	}
	desc := r.FormValue("fileDescription")

	gridFSOptions := options.GridFSUpload()
	gridFSOptions.SetMetadata(bson.M{"fileSender" : Claims.Sub, "fileDescription" : desc})
	bucket, err := gridfs.NewBucket(db)
	if err != nil {
		log.WithFields(log.Fields{
			"function" : "gridfs.NewBucket(db)",
			"handler" : "uploadFile",
			"error"	:	err,
		},
		).Warn("Can't create new bucket!")
		return
	}
	uploadStream, err := bucket.OpenUploadStream(handler.Filename, gridFSOptions)
	if err != nil {
		log.WithFields(log.Fields{
			"function" : "OpenUploadStream(handler.Filename, gridFSOptions)",
			"handler" : "uploadFile",
			"error"	:	err,
		},
		).Warn("Can't open upload stream!")
		return
	}

	fileSize, err := uploadStream.Write(fileBytes)
	if err != nil {
		log.WithFields(log.Fields{
			"function" : "uploadStream.Write(fileBytes)",
			"handler" : "uploadFile",
			"error"	:	err,
		},
		).Warn("Can't write to upload stream!")
		return
	}
	log.WithFields(log.Fields{
		"fileSize" : fileSize,
	},
	).Info("Write file to DB was successful. File size: ")

	fileID := uploadStream.FileID
	uploadStream.Close()
	var file models.File
	filter := bson.M{"_id" : fileID}
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	err = collection.FindOne(ctx, filter).Decode(&file)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	json.NewEncoder(w).Encode(file)
}

func deleteFile(w http.ResponseWriter, r *http.Request) {
	var requiredFile models.File
	data := mux.Vars(r)

	objID, err := primitive.ObjectIDFromHex(data["id"])
	if err != nil {
		http.NotFound(w, r)
		return
	}
	filter := bson.M{"_id": objID}
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	err = collection.FindOne(ctx, filter).Decode(&requiredFile)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if requiredFile.Metadata.FileSender == Claims.Sub || isAdmin() {
		bucket, err := gridfs.NewBucket(db)
		if err != nil {
			log.WithFields(log.Fields{
				"function" : "gridfs.NewBucket(db)",
				"handler" : "deleteFile",
				"error"	:	err,
			},
			).Warn("Can't create new bucket!")
			return
		}
		err = bucket.Delete(objID)

		if err != nil {
			log.WithFields(log.Fields{
				"function" : "bucket.Delete",
				"handler" : "deleteFile",
				"error"	:	err,
			},
			).Fatal("DB interaction resulted in error, shutting down...")
		}
		w.Write([]byte("Successfully deleted file!"))
		w.WriteHeader(200)
	} else {
		w.WriteHeader(403)
		return
	}
}

func getFilesList(w http.ResponseWriter, r *http.Request) {
	files := make([]models.File, 0)
	var filter bson.M

	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	switch {
	case isAdmin():
		filter = bson.M{}
	case isUser():
		filter = bson.M{
			"metadata.fileSender": Claims.Sub,
		}
	}
	data := mux.Vars(r)
	sortVar := data["var"]
	findOptions := options.Find()
	switch sortVar {
	case "name":
		findOptions.SetSort(bson.M{"metadata.fileSender": 1})
	case "date":
		findOptions.SetSort(bson.M{"uploadDate": 1})
	}
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	cur, err := collection.Find(ctx, filter, findOptions)
	if err != nil {
		log.WithFields(log.Fields{
			"function" : "mongo.Find",
			"handler" : "getFilesList",
			"error"	:	err,
		},
		).Fatal("DB interaction resulted in error, shutting down...")
	}
	ctx, _ = context.WithTimeout(context.Background(), 10*time.Second)
	defer cur.Close(ctx)
	ctx, _ = context.WithTimeout(context.Background(), 10*time.Second)
	err = cur.All(ctx, &files)
	if err != nil {
		log.WithFields(log.Fields{
			"function" : "mongo.All",
			"handler" : "getFilesList",
			"error"	:	err,
		},
		).Fatal("DB interaction resulted in error, shutting down...")
	}
	json.NewEncoder(w).Encode(files)
}

func getFilesListForUser(w http.ResponseWriter, r *http.Request) {
	files := make([]models.File, 0)

	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	data := mux.Vars(r)
	user := data["user"]
	if user == Claims.Sub || isAdmin() {
		filter := bson.M{
			"metadata.fileSender" : user,
		}
		findOptions := options.Find()
		findOptions.SetSort(bson.M{"uploadDate": 1})
		ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
		cur, err := collection.Find(ctx, filter, findOptions)
		if err != nil {
			log.WithFields(log.Fields{
				"function" : "mongo.Find",
				"handler" : "getFilesList",
				"error"	:	err,
			},
			).Fatal("DB interaction resulted in error, shutting down...")
		}
		ctx, _ = context.WithTimeout(context.Background(), 10*time.Second)
		defer cur.Close(ctx)
		ctx, _ = context.WithTimeout(context.Background(), 10*time.Second)
		err = cur.All(ctx, &files)
		if err != nil {
			log.WithFields(log.Fields{
				"function" : "mongo.All",
				"handler" : "getFilesList",
				"error"	:	err,
			},
			).Fatal("DB interaction resulted in error, shutting down...")
		}
		json.NewEncoder(w).Encode(files)
	} else {
		w.WriteHeader(403)
		return
	}
}

func getFile(w http.ResponseWriter, r *http.Request) {
	var file models.File

	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	data := mux.Vars(r)
	objID, err := primitive.ObjectIDFromHex(data["id"])
	if err != nil {
		http.NotFound(w, r)
		return
	}
	filter := bson.M{"_id" : objID}
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	err = collection.FindOne(ctx, filter).Decode(&file)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	json.NewEncoder(w).Encode(file)
}