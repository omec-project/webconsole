// SPDX-FileCopyrightText: 2024 Open Networking Foundation <info@opennetworking.org>
// SPDX-FileCopyrightText: 2019 free5GC.org
// SPDX-FileCopyrightText: 2024 Canonical Ltd
//
// SPDX-License-Identifier: Apache-2.0
package dbadapter

import (
	"context"
	"fmt"
	"time"

	"github.com/omec-project/util/mongoapi"
	"github.com/omec-project/webconsole/backend/factory"
	"github.com/omec-project/webconsole/backend/logger"
	"github.com/omec-project/webconsole/configmodels"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type DBInterface interface {
	RestfulAPIGetOne(collName string, filter bson.M) (map[string]interface{}, error)
	RestfulAPIGetMany(collName string, filter bson.M) ([]map[string]interface{}, error)
	RestfulAPIPutOneTimeout(collName string, filter bson.M, putData map[string]interface{}, timeout int32, timeField string) bool
	RestfulAPIPutOne(collName string, filter bson.M, putData map[string]interface{}) (bool, error)
	RestfulAPIPutOneWithContext(context context.Context, collName string, filter bson.M, putData map[string]interface{}) (bool, error)
	RestfulAPIPutOneNotUpdate(collName string, filter bson.M, putData map[string]interface{}) (bool, error)
	RestfulAPIPutMany(collName string, filterArray []primitive.M, putDataArray []map[string]interface{}) error
	RestfulAPIDeleteOne(collName string, filter bson.M) error
	RestfulAPIDeleteOneWithContext(context context.Context, collName string, filter bson.M) error
	RestfulAPIDeleteMany(collName string, filter bson.M) error
	RestfulAPIMergePatch(collName string, filter bson.M, patchData map[string]interface{}) error
	RestfulAPIJSONPatch(collName string, filter bson.M, patchJSON []byte) error
	RestfulAPIJSONPatchWithContext(context context.Context, collName string, filter bson.M, patchJSON []byte) error
	RestfulAPIJSONPatchExtend(collName string, filter bson.M, patchJSON []byte, dataName string) error
	RestfulAPIPost(collName string, filter bson.M, postData map[string]interface{}) (bool, error)
	RestfulAPIPostWithContext(context context.Context, collName string, filter bson.M, postData map[string]interface{}) (bool, error)
	RestfulAPIPostMany(collName string, filter bson.M, postDataArray []interface{}) error
	RestfulAPIPostManyWithContext(context context.Context, collName string, filter bson.M, postDataArray []interface{}) error
	RestfulAPICount(collName string, filter bson.M) (int64, error)
	RestfulAPIPullOne(collName string, filter bson.M, putData map[string]interface{}) error
	RestfulAPIPullOneWithContext(context context.Context, collName string, filter bson.M, putData map[string]interface{}) error
	CreateIndex(collName string, keyField string) (bool, error)
	StartSession() (mongo.Session, error)
	SupportsTransactions() (bool, error)
}

var (
	CommonDBClient DBInterface
	AuthDBClient   DBInterface
	WebuiDBClient  DBInterface
)

type MongoDBClient struct {
	mongoapi.MongoClient
}

type SessionRunner func(ctx context.Context, fn func(sc mongo.SessionContext) error) error

func RealSessionRunner(client *mongo.Client) SessionRunner {
	return func(ctx context.Context, fn func(sc mongo.SessionContext) error) error {
		session, err := client.StartSession()
		if err != nil {
			return err
		}
		defer session.EndSession(ctx)
		return mongo.WithSession(ctx, session, func(sc mongo.SessionContext) error {
			if err := session.StartTransaction(); err != nil {
				return err
			}
			if err := fn(sc); err != nil {
				_ = session.AbortTransaction(sc)
				return err
			}
			return session.CommitTransaction(sc)
		})
	}
}

type PatchOperation struct {
	Value interface{} `json:"value,omitempty"`
	Op    string      `json:"op"`
	Path  string      `json:"path"`
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

func CheckTransactionsSupport(client *DBInterface) error {
	if client == nil || *client == nil {
		return fmt.Errorf("mongoDB client has not been initialized")
	}
	ticker := time.NewTicker(60 * time.Second)
	defer func() { ticker.Stop() }()
	timer := time.After(180 * time.Second)
	logger.DbLog.Infoln("checking for replica set or sharded config in MongoDB...")
	for {
		supportsTransactions, err := (*client).SupportsTransactions()
		if err != nil {
			logger.DbLog.Warnw("could not verify replica set or sharded status", "error", err)
		}
		if supportsTransactions {
			break
		}
		select {
		case <-ticker.C:
			// Continue to check after each tick
		case <-timer:
			return fmt.Errorf("timed out while waiting for Replica Set or sharded config to be set in MongoDB")
		}
	}
	logger.DbLog.Infoln("mongoDB support of transactions verified")
	return nil
}

func InitMongoDB() error {
	if factory.WebUIConfig.Configuration == nil {
		return fmt.Errorf("configuration is nil")
	}

	mongodb := factory.WebUIConfig.Configuration.Mongodb
	logger.InitLog.Infow("MongoDB configuration loaded",
		"mode5G", factory.WebUIConfig.Configuration.Mode5G,
		"enableAuth", factory.WebUIConfig.Configuration.EnableAuthentication)

	if factory.WebUIConfig.Configuration.Mode5G {
		ConnectMongo(mongodb.Url, mongodb.Name, &CommonDBClient)
		logger.InitLog.Infow("Connected to common database",
			"url", mongodb.Url,
			"dbName", mongodb.Name)

		if err := CheckTransactionsSupport(&CommonDBClient); err != nil {
			logger.DbLog.Errorw("failed to connect to MongoDB client", mongodb.Name, "error", err)
			return err
		}

		ConnectMongo(mongodb.AuthUrl, mongodb.AuthKeysDbName, &AuthDBClient)
		logger.InitLog.Infow("Connected to auth database",
			"url", mongodb.AuthUrl,
			"dbName", mongodb.AuthKeysDbName)

		if resp, err := CommonDBClient.CreateIndex(configmodels.UpfDataColl, "hostname"); !resp || err != nil {
			logger.InitLog.Errorf("error creating UPF index in commonDB %v", err)
			return err
		}
		if resp, err := CommonDBClient.CreateIndex(configmodels.GnbDataColl, "name"); !resp || err != nil {
			logger.InitLog.Errorf("error creating gNB index in commonDB %v", err)
			return err
		}
	}
	if factory.WebUIConfig.Configuration.EnableAuthentication {
		ConnectMongo(mongodb.WebuiDBUrl, mongodb.WebuiDBName, &WebuiDBClient)
		if resp, err := WebuiDBClient.CreateIndex(configmodels.UserAccountDataColl, "username"); !resp || err != nil {
			logger.InitLog.Errorf("error initializing webuiDB %v", err)
			return err
		}
	}

	logger.InitLog.Info("MongoDB initialization completed successfully")
	return nil
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

func (db *MongoDBClient) RestfulAPIPutOneWithContext(context context.Context, collName string, filter bson.M, putData map[string]interface{}) (bool, error) {
	return db.MongoClient.RestfulAPIPutOneWithContext(context, collName, filter, putData)
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

func (db *MongoDBClient) RestfulAPIDeleteOneWithContext(context context.Context, collName string, filter bson.M) error {
	return db.MongoClient.RestfulAPIDeleteOneWithContext(context, collName, filter)
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

func (db *MongoDBClient) RestfulAPIJSONPatchWithContext(context context.Context, collName string, filter bson.M, patchJSON []byte) error {
	return db.MongoClient.RestfulAPIJSONPatchWithContext(context, collName, filter, patchJSON)
}

func (db *MongoDBClient) RestfulAPIJSONPatchExtend(collName string, filter bson.M, patchJSON []byte, dataName string) error {
	return db.MongoClient.RestfulAPIJSONPatchExtend(collName, filter, patchJSON, dataName)
}

func (db *MongoDBClient) RestfulAPIPost(collName string, filter bson.M, postData map[string]interface{}) (bool, error) {
	return db.MongoClient.RestfulAPIPost(collName, filter, postData)
}

func (db *MongoDBClient) RestfulAPIPostWithContext(context context.Context, collName string, filter bson.M, postData map[string]interface{}) (bool, error) {
	return db.MongoClient.RestfulAPIPostWithContext(context, collName, filter, postData)
}

func (db *MongoDBClient) RestfulAPIPostMany(collName string, filter bson.M, postDataArray []interface{}) error {
	return db.MongoClient.RestfulAPIPostMany(collName, filter, postDataArray)
}

func (db *MongoDBClient) RestfulAPIPostManyWithContext(context context.Context, collName string, filter bson.M, postDataArray []interface{}) error {
	return db.MongoClient.RestfulAPIPostManyWithContext(context, collName, filter, postDataArray)
}

func (db *MongoDBClient) RestfulAPICount(collName string, filter bson.M) (int64, error) {
	return db.MongoClient.RestfulAPICount(collName, filter)
}

func (db *MongoDBClient) RestfulAPIPullOne(collName string, filter bson.M, putData map[string]interface{}) error {
	return db.MongoClient.RestfulAPIPullOne(collName, filter, putData)
}

func (db *MongoDBClient) RestfulAPIPullOneWithContext(context context.Context, collName string, filter bson.M, putData map[string]interface{}) error {
	return db.MongoClient.RestfulAPIPullOneWithContext(context, collName, filter, putData)
}

func (db *MongoDBClient) CreateIndex(collName string, keyField string) (bool, error) {
	return db.MongoClient.CreateIndex(collName, keyField)
}

func (db *MongoDBClient) StartSession() (mongo.Session, error) {
	return db.MongoClient.StartSession()
}

func (db *MongoDBClient) SupportsTransactions() (bool, error) {
	return db.MongoClient.SupportsTransactions()
}
