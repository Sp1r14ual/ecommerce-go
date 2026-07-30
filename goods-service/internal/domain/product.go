package domain

import "go.mongodb.org/mongo-driver/bson/primitive"

type Product struct {
	// bson:"_id,omitempty" говорит драйверу Mongo использовать это поле как главный ключ _id
	// и сгенерировать его автоматически, если мы при создании его не передали
	ID          primitive.ObjectID `bson:"_id,omitempty"`
	Name        string             `bson:"name"`
	Description string             `bson:"description"`
	Price       int64              `bson:"price"`
}
