// SPDX-License-Identifier: Apache-2.0
// Copyright 2025 Canonical Ltd.

package configapi

import (
	"context"
	"errors"

	"github.com/omec-project/webconsole/dbadapter"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type MockSession struct {
	mongo.Session
}

func (m *MockSession) WithTransaction(ctx context.Context, fn func(ctx context.Context) (interface{}, error), opts ...options.Lister[options.TransactionOptions]) (interface{}, error) {
	return fn(ctx)
}

func (m *MockSession) WithSession(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}

func (m *MockSession) StartTransaction(opts ...options.Lister[options.TransactionOptions]) error {
	return nil
}

func (m *MockSession) AbortTransaction(ctx context.Context) error {
	return nil
}

func (m *MockSession) CommitTransaction(ctx context.Context) error {
	return nil
}

func (m *MockSession) EndSession(ctx context.Context) {}

type MockMongoClientDBError struct {
	dbadapter.DBInterface
}

func (db *MockMongoClientDBError) RestfulAPIGetOne(coll string, filter bson.M) (map[string]any, error) {
	return nil, errors.New("DB error")
}

func (db *MockMongoClientDBError) RestfulAPIGetMany(coll string, filter bson.M) ([]map[string]any, error) {
	return nil, errors.New("DB error")
}

func (db *MockMongoClientDBError) RestfulAPIPutOneWithContext(context context.Context, collName string, filter bson.M, putData map[string]any) (bool, error) {
	return false, errors.New("DB error")
}

func (db *MockMongoClientDBError) RestfulAPIDeleteOneWithContext(context context.Context, collName string, filter bson.M) error {
	return errors.New("DB error")
}

func (db *MockMongoClientDBError) RestfulAPIJSONPatchWithContext(context context.Context, collName string, filter bson.M, patchJSON []byte) error {
	return errors.New("DB error")
}

func (db *MockMongoClientDBError) RestfulAPIPost(collName string, filter bson.M, postData map[string]any) (bool, error) {
	return false, errors.New("DB error")
}

func (db *MockMongoClientDBError) RestfulAPIPostWithContext(context context.Context, collName string, filter bson.M, postData map[string]any) (bool, error) {
	return false, errors.New("DB error")
}

func (db *MockMongoClientDBError) RestfulAPIPostMany(collName string, filter bson.M, postDataArray []any) error {
	return errors.New("DB error")
}

func (db *MockMongoClientDBError) RestfulAPIPostManyWithContext(context context.Context, collName string, filter bson.M, postDataArray []any) error {
	return errors.New("DB error")
}

func (db *MockMongoClientDBError) RestfulAPICount(collName string, filter bson.M) (int64, error) {
	return 0, errors.New("DB error")
}

func (m *MockMongoClientDBError) StartSession() (dbadapter.DBSession, error) {
	return nil, nil
}

func (db *MockMongoClientDBError) RestfulAPIPostOnDB(ctx context.Context, dbName string, collName string, filter bson.M, postData map[string]any) (bool, error) {
	return false, errors.New("DB error")
}

func (db *MockMongoClientDBError) RestfulAPIPutOneOnDB(ctx context.Context, dbName string, collName string, filter bson.M, putData map[string]any) (bool, error) {
	return false, errors.New("DB error")
}

func (db *MockMongoClientDBError) RestfulAPIDeleteOneOnDB(ctx context.Context, dbName string, collName string, filter bson.M) error {
	return errors.New("DB error")
}

type MockMongoClientEmptyDB struct {
	dbadapter.DBInterface
}

func (db *MockMongoClientEmptyDB) RestfulAPIGetOne(collName string, filter bson.M) (map[string]any, error) {
	return map[string]any{}, nil
}

func (db *MockMongoClientEmptyDB) RestfulAPIGetMany(coll string, filter bson.M) ([]map[string]any, error) {
	var results []map[string]any
	return results, nil
}

func (db *MockMongoClientEmptyDB) RestfulAPIPutOneWithContext(context context.Context, collName string, filter bson.M, putData map[string]any) (bool, error) {
	return false, nil
}

func (db *MockMongoClientEmptyDB) RestfulAPIPost(collName string, filter bson.M, postData map[string]any) (bool, error) {
	return true, nil
}

func (db *MockMongoClientEmptyDB) RestfulAPIPostWithContext(context context.Context, collName string, filter bson.M, postData map[string]any) (bool, error) {
	return true, nil
}

func (db *MockMongoClientEmptyDB) RestfulAPIPostMany(collName string, filter bson.M, postDataArray []any) error {
	return nil
}

func (db *MockMongoClientEmptyDB) RestfulAPIPostManyWithContext(context context.Context, collName string, filter bson.M, postDataArray []any) error {
	return nil
}

func (db *MockMongoClientEmptyDB) RestfulAPIDeleteOneWithContext(context context.Context, collName string, filter bson.M) error {
	return nil
}

func (db *MockMongoClientEmptyDB) RestfulAPIJSONPatchWithContext(context context.Context, collName string, filter bson.M, patchJSON []byte) error {
	return nil
}

func (db *MockMongoClientEmptyDB) RestfulAPICount(collName string, filter bson.M) (int64, error) {
	return 0, nil
}

func (m *MockMongoClientEmptyDB) StartSession() (dbadapter.DBSession, error) {
	return nil, nil
}

func (m *MockMongoClientEmptyDB) RestfulAPIDeleteOne(coll string, filter bson.M) error {
	return nil
}

func (m *MockMongoClientEmptyDB) Client() *mongo.Client {
	return nil
}

func (db *MockMongoClientEmptyDB) RestfulAPIPostOnDB(ctx context.Context, dbName string, collName string, filter bson.M, postData map[string]any) (bool, error) {
	return true, nil
}

func (db *MockMongoClientEmptyDB) RestfulAPIPutOneOnDB(ctx context.Context, dbName string, collName string, filter bson.M, putData map[string]any) (bool, error) {
	return false, nil
}

func (db *MockMongoClientEmptyDB) RestfulAPIDeleteOneOnDB(ctx context.Context, dbName string, collName string, filter bson.M) error {
	return nil
}
