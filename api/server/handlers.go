package server

import (
	"MicroFileServer/models"
	"bytes"
	"context"
	"fmt"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/gridfs"
	"io/ioutil"
	"net/http"
	"os"
	"time"
)



func downloadFile(w http.ResponseWriter, r *http.Request) {
	var downloadedFile models.File
	var buf bytes.Buffer
	data := mux.Vars(r)

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
	filyBytes, err := ioutil.ReadAll(data)
	if err != nil {
		log.Fatal(err)
	}
	bucket, err := gridfs.NewBucket(db)
	if err != nil {
		log.Fatal(err)
	}

	uploadStream, err := bucket.OpenUploadStream(handler.Filename)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer uploadStream.Close()

	fileSize, err := uploadStream.Write(filyBytes)
	if err != nil {
		log.Fatal(err)
	}
	log.Info("Write file to DB was successful. File size: %d M\n", fileSize)
	w.Write([]byte("Successfully uploaded file!"))
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
