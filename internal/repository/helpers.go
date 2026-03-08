package repository

import (
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func mongoArrayFilters(filters ...bson.M) *options.UpdateOptions {
	af := make([]interface{}, len(filters))
	for i, f := range filters {
		af[i] = f
	}
	return options.Update().SetArrayFilters(options.ArrayFilters{Filters: af})
}
