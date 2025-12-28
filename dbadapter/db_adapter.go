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
	"go.mongodb.org/mongo-driver/mongo/options"
)

type DBInterface interface {
	RestfulAPIGetOne(collName string, filter bson.M) (map[string]any, error)
	RestfulAPIGetMany(collName string, filter bson.M) ([]map[string]any, error)
	RestfulAPIPutOneTimeout(collName string, filter bson.M, putData map[string]any, timeout int32, timeField string) bool
	RestfulAPIPutOne(collName string, filter bson.M, putData map[string]any) (bool, error)
	RestfulAPIPutOneWithContext(context context.Context, collName string, filter bson.M, putData map[string]any) (bool, error)
	RestfulAPIPutOneNotUpdate(collName string, filter bson.M, putData map[string]any) (bool, error)
	RestfulAPIPutMany(collName string, filterArray []primitive.M, putDataArray []map[string]any) error
	RestfulAPIDeleteOne(collName string, filter bson.M) error
	RestfulAPIDeleteOneWithContext(context context.Context, collName string, filter bson.M) error
	RestfulAPIDeleteMany(collName string, filter bson.M) error
	RestfulAPIMergePatch(collName string, filter bson.M, patchData map[string]any) error
	RestfulAPIJSONPatch(collName string, filter bson.M, patchJSON []byte) error
	RestfulAPIJSONPatchWithContext(context context.Context, collName string, filter bson.M, patchJSON []byte) error
	RestfulAPIJSONPatchExtend(collName string, filter bson.M, patchJSON []byte, dataName string) error
	RestfulAPIPost(collName string, filter bson.M, postData map[string]any) (bool, error)
	RestfulAPIPostWithContext(context context.Context, collName string, filter bson.M, postData map[string]any) (bool, error)
	RestfulAPIPostMany(collName string, filter bson.M, postDataArray []any) error
	RestfulAPIPostManyWithContext(context context.Context, collName string, filter bson.M, postDataArray []any) error
	RestfulAPICount(collName string, filter bson.M) (int64, error)
	RestfulAPIPullOne(collName string, filter bson.M, putData map[string]any) error
	RestfulAPIPullOneWithContext(context context.Context, collName string, filter bson.M, putData map[string]any) error
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

func GetSessionRunner(client DBInterface) SessionRunner {
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
				abortErr := session.AbortTransaction(sc)
				logger.DbLog.Warnf("failed to abort transaction: %v", abortErr)
				return err
			}
			return session.CommitTransaction(sc)
		})
	}
}

type PatchOperation struct {
	Value any    `json:"value,omitempty"`
	Op    string `json:"op"`
	Path  string `json:"path"`
}

type OptConfig struct {
	MaxPoolSize uint64
	MinPoolSize uint64
}

func setDBClient(url, dbname string, optConfig OptConfig) (DBInterface, error) {
	mClient, errConnect := mongoapi.NewMongoClient(url, dbname)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	opts := options.Client().ApplyURI(url).
		SetMaxPoolSize(optConfig.MaxPoolSize).
		SetMinPoolSize(optConfig.MinPoolSize)

	client, err := mongo.Connect(ctx, opts)
	if err != nil {
		return nil, err
	}
	err = mClient.Client.Disconnect(context.Background())
	if err != nil {
		return nil, err
	}
	mClient.Client = client
	if errConnect != nil {
		return nil, errConnect
	}

	return &MongoDBClient{*mClient}, nil
}

func ConnectMongo(url string, dbname string, client *DBInterface, opts OptConfig) {
	ticker := time.NewTicker(2 * time.Second)
	defer func() { ticker.Stop() }()
	timer := time.After(180 * time.Second)
ConnectMongo:
	for {
		var err error
		*client, err = setDBClient(url, dbname, opts)
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
	checkReplica := factory.WebUIConfig.Configuration.Mongodb.CheckReplica

	// enabled check replica set step, focus on dev
	if !checkReplica {
		logger.DbLog.Infoln("replicaset is not necessary, mongodb config is correct, connect is success")
		return nil
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
		"enableAuth", factory.WebUIConfig.Configuration.EnableAuthentication)

	ConnectMongo(mongodb.Url, mongodb.Name, &CommonDBClient, OptConfig{
		MaxPoolSize: uint64(mongodb.DefaultConns),
		MinPoolSize: 10,
	})
	logger.InitLog.Infow("Connected to common database",
		"url", mongodb.Url,
		"dbName", mongodb.Name)

	if err := CheckTransactionsSupport(&CommonDBClient); err != nil {
		logger.DbLog.Errorw("failed to connect to MongoDB client", mongodb.Name, "error", err)
		return err
	}

	ConnectMongo(mongodb.AuthUrl, mongodb.AuthKeysDbName, &AuthDBClient, OptConfig{
		MaxPoolSize: uint64(mongodb.AuthConns),
		MinPoolSize: 10,
	})
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

	if factory.WebUIConfig.Configuration.EnableAuthentication {
		ConnectMongo(mongodb.WebuiDBUrl, mongodb.WebuiDBName, &WebuiDBClient, OptConfig{
			MaxPoolSize: uint64(mongodb.WebuiDbConns),
			MinPoolSize: 10,
		})
		if resp, err := WebuiDBClient.CreateIndex(configmodels.UserAccountDataColl, "username"); !resp || err != nil {
			logger.InitLog.Errorf("error initializing webuiDB %v", err)
			return err
		}
	}

	logger.InitLog.Info("MongoDB initialization completed successfully")
	return nil
}

func (db *MongoDBClient) RestfulAPIGetOne(collName string, filter bson.M) (map[string]any, error) {
	return db.MongoClient.RestfulAPIGetOne(collName, filter)
}

func (db *MongoDBClient) RestfulAPIGetMany(collName string, filter bson.M) ([]map[string]any, error) {
	return db.MongoClient.RestfulAPIGetMany(collName, filter)
}

func (db *MongoDBClient) RestfulAPIPutOneTimeout(collName string, filter bson.M, putData map[string]any, timeout int32, timeField string) bool {
	return db.MongoClient.RestfulAPIPutOneTimeout(collName, filter, putData, timeout, timeField)
}

func (db *MongoDBClient) RestfulAPIPutOne(collName string, filter bson.M, putData map[string]any) (bool, error) {
	return db.MongoClient.RestfulAPIPutOne(collName, filter, putData)
}

func (db *MongoDBClient) RestfulAPIPutOneWithContext(context context.Context, collName string, filter bson.M, putData map[string]any) (bool, error) {
	return db.MongoClient.RestfulAPIPutOneWithContext(context, collName, filter, putData)
}

func (db *MongoDBClient) RestfulAPIPutOneNotUpdate(collName string, filter bson.M, putData map[string]any) (bool, error) {
	return db.MongoClient.RestfulAPIPutOneNotUpdate(collName, filter, putData)
}

func (db *MongoDBClient) RestfulAPIPutMany(collName string, filterArray []primitive.M, putDataArray []map[string]any) error {
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

func (db *MongoDBClient) RestfulAPIMergePatch(collName string, filter bson.M, patchData map[string]any) error {
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

func (db *MongoDBClient) RestfulAPIPost(collName string, filter bson.M, postData map[string]any) (bool, error) {
	return db.MongoClient.RestfulAPIPost(collName, filter, postData)
}

func (db *MongoDBClient) RestfulAPIPostWithContext(context context.Context, collName string, filter bson.M, postData map[string]any) (bool, error) {
	return db.MongoClient.RestfulAPIPostWithContext(context, collName, filter, postData)
}

func (db *MongoDBClient) RestfulAPIPostMany(collName string, filter bson.M, postDataArray []any) error {
	return db.MongoClient.RestfulAPIPostMany(collName, filter, postDataArray)
}

func (db *MongoDBClient) RestfulAPIPostManyWithContext(context context.Context, collName string, filter bson.M, postDataArray []any) error {
	return db.MongoClient.RestfulAPIPostManyWithContext(context, collName, filter, postDataArray)
}

func (db *MongoDBClient) RestfulAPICount(collName string, filter bson.M) (int64, error) {
	return db.MongoClient.RestfulAPICount(collName, filter)
}

func (db *MongoDBClient) RestfulAPIPullOne(collName string, filter bson.M, putData map[string]any) error {
	return db.MongoClient.RestfulAPIPullOne(collName, filter, putData)
}

func (db *MongoDBClient) RestfulAPIPullOneWithContext(context context.Context, collName string, filter bson.M, putData map[string]any) error {
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
