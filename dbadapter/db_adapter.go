// SPDX-FileCopyrightText: 2024 Open Networking Foundation <info@opennetworking.org>
// SPDX-FileCopyrightText: 2019 free5GC.org
// SPDX-FileCopyrightText: 2024 Canonical Ltd
//
// SPDX-License-Identifier: Apache-2.0
package dbadapter

import (
	"time"

	"github.com/omec-project/util/mongoapi"
	"github.com/omec-project/webconsole/backend/logger"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type DBInterface interface {
	RestfulAPIGetOne(collName string, filter bson.M) (map[string]interface{}, error)
	RestfulAPIGetMany(collName string, filter bson.M) ([]map[string]interface{}, error)
	RestfulAPIPutOneTimeout(collName string, filter bson.M, putData map[string]interface{}, timeout int32, timeField string) bool
	RestfulAPIPutOne(collName string, filter bson.M, putData map[string]interface{}) (bool, error)
	RestfulAPIPutOneNotUpdate(collName string, filter bson.M, putData map[string]interface{}) (bool, error)
	RestfulAPIPutMany(collName string, filterArray []primitive.M, putDataArray []map[string]interface{}) error
	RestfulAPIDeleteOne(collName string, filter bson.M) error
	RestfulAPIDeleteMany(collName string, filter bson.M) error
	RestfulAPIMergePatch(collName string, filter bson.M, patchData map[string]interface{}) error
	RestfulAPIJSONPatch(collName string, filter bson.M, patchJSON []byte) error
	RestfulAPIJSONPatchExtend(collName string, filter bson.M, patchJSON []byte, dataName string) error
	RestfulAPIPost(collName string, filter bson.M, postData map[string]interface{}) (bool, error)
	RestfulAPIPostMany(collName string, filter bson.M, postDataArray []interface{}) error
	RestfulAPICount(collName string, filter bson.M) (int64, error)
	CreateIndex(collName string, keyField string) (bool, error)
}

var (
	CommonDBClient DBInterface
	AuthDBClient   DBInterface
	WebuiDBClient  DBInterface
)

type MongoDBClient struct {
	mongoapi.MongoClient
}

func setDBClient(url, dbname string) (DBInterface, error) {
	mClient, errConnect := mongoapi.NewMongoClient(url, dbname)
	if mClient.Client != nil {
		return mClient, nil
	}
	return nil, errConnect
}

func ConnectMongo(url string, dbname string, client *DBInterface) {
	ticker := time.NewTicker(2 * time.Second)
	defer func() { ticker.Stop() }()
	timer := time.After(180 * time.Second)
ConnectMongo:
	for {
		var err error
		*client, err = setDBClient(url, dbname)
		if err == nil {
			break ConnectMongo
		}
		select {
		case <-ticker.C:
			continue
		case <-timer:
			logger.DbLog.Errorln("timed out while connecting to MongoDB in 3 minutes")
			return
		}
	}
	logger.DbLog.Infoln("connected to MongoDB")
}

func (db *MongoDBClient) RestfulAPIGetOne(collName string, filter bson.M) (map[string]interface{}, error) {
	return db.MongoClient.RestfulAPIGetOne(collName, filter)
}

func (db *MongoDBClient) RestfulAPIGetMany(collName string, filter bson.M) ([]map[string]interface{}, error) {
	return db.MongoClient.RestfulAPIGetMany(collName, filter)
}

func (db *MongoDBClient) RestfulAPIPutOneTimeout(collName string, filter bson.M, putData map[string]interface{}, timeout int32, timeField string) bool {
	return db.MongoClient.RestfulAPIPutOneTimeout(collName, filter, putData, timeout, timeField)
}

func (db *MongoDBClient) RestfulAPIPutOne(collName string, filter bson.M, putData map[string]interface{}) (bool, error) {
	return db.MongoClient.RestfulAPIPutOne(collName, filter, putData)
}

func (db *MongoDBClient) RestfulAPIPutOneNotUpdate(collName string, filter bson.M, putData map[string]interface{}) (bool, error) {
	return db.MongoClient.RestfulAPIPutOneNotUpdate(collName, filter, putData)
}

func (db *MongoDBClient) RestfulAPIPutMany(collName string, filterArray []primitive.M, putDataArray []map[string]interface{}) error {
	return db.MongoClient.RestfulAPIPutMany(collName, filterArray, putDataArray)
}

func (db *MongoDBClient) RestfulAPIDeleteOne(collName string, filter bson.M) error {
	return db.MongoClient.RestfulAPIDeleteOne(collName, filter)
}

func (db *MongoDBClient) RestfulAPIDeleteMany(collName string, filter bson.M) error {
	return db.MongoClient.RestfulAPIDeleteMany(collName, filter)
}

func (db *MongoDBClient) RestfulAPIMergePatch(collName string, filter bson.M, patchData map[string]interface{}) error {
	return db.MongoClient.RestfulAPIMergePatch(collName, filter, patchData)
}

func (db *MongoDBClient) RestfulAPIJSONPatch(collName string, filter bson.M, patchJSON []byte) error {
	return db.MongoClient.RestfulAPIJSONPatch(collName, filter, patchJSON)
}

func (db *MongoDBClient) RestfulAPIJSONPatchExtend(collName string, filter bson.M, patchJSON []byte, dataName string) error {
	return db.MongoClient.RestfulAPIJSONPatchExtend(collName, filter, patchJSON, dataName)
}

func (db *MongoDBClient) RestfulAPIPost(collName string, filter bson.M, postData map[string]interface{}) (bool, error) {
	return db.MongoClient.RestfulAPIPost(collName, filter, postData)
}

func (db *MongoDBClient) RestfulAPIPostMany(collName string, filter bson.M, postDataArray []interface{}) error {
	return db.MongoClient.RestfulAPIPostMany(collName, filter, postDataArray)
}

func (db *MongoDBClient) RestfulAPICount(collName string, filter bson.M) (int64, error) {
	return db.MongoClient.RestfulAPICount(collName, filter)
}

func (db *MongoDBClient) CreateIndex(collName string, keyField string) (bool, error) {
	return db.MongoClient.CreateIndex(collName, keyField)
}
