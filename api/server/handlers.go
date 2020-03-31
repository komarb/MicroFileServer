package server

import (
	"MicroFileServer/logging"
	"MicroFileServer/models"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/gridfs"
	"go.mongodb.org/mongo-driver/mongo/options"
	"io/ioutil"
	"net/http"
	"os"
	"time"
)



func downloadFile(w http.ResponseWriter, r *http.Request) {
	var downloadedFile models.File
	var buf bytes.Buffer

	subClaim, err := getClaim(r, "sub")
	if err != nil {
		logging.AuthError(w, err, "getClaim(sub)")
		return
	}
	data := mux.Vars(r)

	if subClaim != data["id"] {
		w.WriteHeader(403)
		return
	}

	objID, err := primitive.ObjectIDFromHex(string(data["id"]))
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
	w.Header().Set("Content-Disposition", "attachment; filename=\""+fileName+"\"")
	bucket, _ := gridfs.NewBucket(db)

	dStream, err := bucket.DownloadToStream(objID, &buf)
	if err != nil {
		log.WithFields(log.Fields{
			"function" : "bucket.DownloadToStream",
			"handler" : "downloadFile",
			"error"	:	err,
		},
		).Fatal("DB interaction resulted in error, shutting down...")
	}
	fmt.Printf("File size to download: %v\n", dStream)
	ioutil.WriteFile(fileName, buf.Bytes(), 0600)
}



func uploadFile(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	r.Body = http.MaxBytesReader(w, r.Body, 30 * 1024 * 1024)
	data, handler, err := r.FormFile("uploadingForm")
	if err != nil {
		w.Write([]byte("File is too big! (Max size: 30MB)"))
		return
	}

	defer data.Close()
	fileBytes, err := ioutil.ReadAll(data)
	if err != nil {
		log.Fatal(err)
	}
	subClaim, err := getClaim(r, "sub")
	if err != nil {
		logging.AuthError(w, err, "getClaim (sub)")
		return
	}

	desc := r.FormValue("fileDescription")

	gridFSOptions := options.GridFSUpload()
	gridFSOptions.SetMetadata(bson.M{"fileSender" : subClaim, "fileDescription" : desc})
	bucket, err := gridfs.NewBucket(db)
	if err != nil {
		log.Fatal(err)
	}
	uploadStream, err := bucket.OpenUploadStream(handler.Filename, gridFSOptions)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer uploadStream.Close()

	fileSize, err := uploadStream.Write(fileBytes)
	if err != nil {
		log.Fatal(err)
	}
	log.Infof("Write file to DB was successful. File size: %dM\n", fileSize)
	w.Write([]byte("Successfully uploaded file!"))

	/*objID, err := primitive.ObjectIDFromHex(fmt.Sprintf("%v", uploadStream.FileID))
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

	json.NewEncoder(w).Encode(file)*/
}



func deleteFile(w http.ResponseWriter, r *http.Request) {
	var requiredFile models.File
	data := mux.Vars(r)

	objID, err := primitive.ObjectIDFromHex(string(data["id"]))
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
	bucket, err := gridfs.NewBucket(db)
	if err != nil {
		log.Fatal(err)
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
	w.WriteHeader(200)
	w.Write([]byte("Successfully deleted file!"))
}

func getFilesList(w http.ResponseWriter, r *http.Request) {
	files := make([]models.File, 0)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	cur, err := collection.Find(ctx, bson.M{})
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

	filter := bson.M{
		"metadata.fileSender" : user,
	}

	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	cur, err := collection.Find(ctx, filter)
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

func getFile(w http.ResponseWriter, r *http.Request) {
	var file models.File
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	data := mux.Vars(r)

	objID, err := primitive.ObjectIDFromHex(string(data["id"]))
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