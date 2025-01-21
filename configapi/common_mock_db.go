package configapi

import (
	"context"
	"errors"

	"github.com/omec-project/webconsole/dbadapter"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MockSession struct {
	mongo.Session
}

func (m *MockSession) StartTransaction(opts ...*options.TransactionOptions) error {
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

func (db *MockMongoClientDBError) RestfulAPIGetOne(coll string, filter bson.M) (map[string]interface{}, error) {
	return nil, errors.New("DB error")
}

func (db *MockMongoClientDBError) RestfulAPIGetMany(coll string, filter bson.M) ([]map[string]interface{}, error) {
	return nil, errors.New("DB error")
}

func (db *MockMongoClientDBError) RestfulAPIPutOneWithContext(collName string, filter bson.M, putData map[string]interface{}, context context.Context) (bool, error) {
	return false, errors.New("DB error")
}

func (db *MockMongoClientDBError) RestfulAPIDeleteOneWithContext(collName string, filter bson.M, context context.Context) error {
	return errors.New("DB error")
}

func (db *MockMongoClientDBError) RestfulAPIJSONPatchWithContext(collName string, filter bson.M, patchJSON []byte, context context.Context) error {
	return errors.New("DB error")
}

func (db *MockMongoClientDBError) RestfulAPIPost(collName string, filter bson.M, postData map[string]interface{}) (bool, error) {
	return false, errors.New("DB error")
}

func (db *MockMongoClientDBError) RestfulAPIPostMany(collName string, filter bson.M, postDataArray []interface{}) error {
	return errors.New("DB error")
}

func (db *MockMongoClientDBError) RestfulAPICount(collName string, filter bson.M) (int64, error) {
	return 0, errors.New("DB error")
}

func (m *MockMongoClientDBError) StartSession() (mongo.Session, error) {
	return &MockSession{}, nil
}

type MockMongoClientEmptyDB struct {
	dbadapter.DBInterface
}

func (db *MockMongoClientEmptyDB) RestfulAPIGetOne(collName string, filter bson.M) (map[string]interface{}, error) {
	return map[string]interface{}{}, nil
}

func (db *MockMongoClientEmptyDB) RestfulAPIGetMany(coll string, filter bson.M) ([]map[string]interface{}, error) {
	var results []map[string]interface{}
	return results, nil
}

func (db *MockMongoClientEmptyDB) RestfulAPIPutOneWithContext(collName string, filter bson.M, putData map[string]interface{}, context context.Context) (bool, error) {
	return false, nil
}

func (db *MockMongoClientEmptyDB) RestfulAPIPost(collName string, filter bson.M, postData map[string]interface{}) (bool, error) {
	return true, nil
}

func (db *MockMongoClientEmptyDB) RestfulAPIPostMany(collName string, filter bson.M, postDataArray []interface{}) error {
	return nil
}

func (db *MockMongoClientEmptyDB) RestfulAPIDeleteOneWithContext(collName string, filter bson.M, context context.Context) error {
	return nil
}

func (db *MockMongoClientEmptyDB) RestfulAPIJSONPatchWithContext(collName string, filter bson.M, patchJSON []byte, context context.Context) error {
	return nil
}

func (db *MockMongoClientEmptyDB) RestfulAPICount(collName string, filter bson.M) (int64, error) {
	return 0, nil
}

func (m *MockMongoClientEmptyDB) StartSession() (mongo.Session, error) {
	return &MockSession{}, nil
}

type MockMongoClientDuplicateCreation struct {
	dbadapter.DBInterface
}

func (db *MockMongoClientDuplicateCreation) RestfulAPIGetMany(coll string, filter bson.M) ([]map[string]interface{}, error) {
	var results []map[string]interface{}
	return results, nil
}

func (db *MockMongoClientDuplicateCreation) RestfulAPIPostMany(collName string, filter bson.M, postDataArray []interface{}) error {
	return errors.New("E11000")
}

func (db *MockMongoClientDuplicateCreation) RestfulAPICount(collName string, filter bson.M) (int64, error) {
	return 1, nil
}

func (m *MockMongoClientDuplicateCreation) StartSession() (mongo.Session, error) {
	return &MockSession{}, nil
}
