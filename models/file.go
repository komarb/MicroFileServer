package models

import "go.mongodb.org/mongo-driver/bson/primitive"

type File struct {
	ID           primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	EncodedFile	 string             `json:"reportSender"`
}
