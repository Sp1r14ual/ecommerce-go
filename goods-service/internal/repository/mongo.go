package repository

import (
	"context"

	"github.com/Sp1r14ual/ecommerce-go/goods-service/internal/domain"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type GoodsRepo struct {
	collection *mongo.Collection
}

func NewGoodsRepo(db *mongo.Database) *GoodsRepo {
	// В Монго нет таблиц, есть "Коллекции" (collections)
	return &GoodsRepo{
		collection: db.Collection("products"),
	}
}

func (r *GoodsRepo) Create(ctx context.Context, p *domain.Product) (string, error) {
	// InsertOne вставляет документ в базу
	result, err := r.collection.InsertOne(ctx, p)
	if err != nil {
		return "", err
	}

	// Mongo возвращает сгенерированный ObjectID. Приводим его к строке
	oid := result.InsertedID.(primitive.ObjectID).Hex()
	return oid, nil
}

func (r *GoodsRepo) ListAll(ctx context.Context) ([]*domain.Product, error) {
	// bson.M{} означает пустой фильтр (найти всё)
	cursor, err := r.collection.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var products []*domain.Product
	// Декодируем результаты поиска в наш слайс структур
	if err = cursor.All(ctx, &products); err != nil {
		return nil, err
	}

	return products, nil
}

func (r *GoodsRepo) Get(ctx context.Context, id string) (*domain.Product, error) {
	// Конвертируем строку с ID в формат ObjectID для Mongo
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}

	var p domain.Product
	// Ищем один документ по _id
	err = r.collection.FindOne(ctx, bson.M{"_id": oid}).Decode(&p)
	return &p, err
}
