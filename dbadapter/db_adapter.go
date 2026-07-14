// Copyright (C) 2026 Intel Corporation
// SPDX-FileCopyrightText: 2024 Canonical Ltd
// SPDX-FileCopyrightText: 2024 Open Networking Foundation <info@opennetworking.org>
// SPDX-FileCopyrightText: 2019 free5GC.org
// SPDX-License-Identifier: Apache-2.0
package dbadapter

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/omec-project/util/mongoapi"
	"github.com/omec-project/webconsole/backend/factory"
	"github.com/omec-project/webconsole/backend/logger"
	"github.com/omec-project/webconsole/configmodels"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type DBSession interface {
	EndSession(ctx context.Context)
	WithTransaction(ctx context.Context, fn func(ctx context.Context) (any, error), opts ...options.Lister[options.TransactionOptions]) (any, error)
	StartTransaction(opts ...options.Lister[options.TransactionOptions]) error
	AbortTransaction(ctx context.Context) error
	CommitTransaction(ctx context.Context) error
	WithSession(ctx context.Context, fn func(context.Context) error) error
}

type MongoDBSession struct {
	session *mongo.Session
}

func (s *MongoDBSession) EndSession(ctx context.Context) {
	s.session.EndSession(ctx)
}

func (s *MongoDBSession) WithTransaction(ctx context.Context, fn func(ctx context.Context) (any, error), opts ...options.Lister[options.TransactionOptions]) (any, error) {
	return s.session.WithTransaction(ctx, fn, opts...)
}

func (s *MongoDBSession) StartTransaction(opts ...options.Lister[options.TransactionOptions]) error {
	return s.session.StartTransaction(opts...)
}

func (s *MongoDBSession) AbortTransaction(ctx context.Context) error {
	return s.session.AbortTransaction(ctx)
}

func (s *MongoDBSession) CommitTransaction(ctx context.Context) error {
	return s.session.CommitTransaction(ctx)
}

func (s *MongoDBSession) WithSession(ctx context.Context, fn func(context.Context) error) error {
	return mongo.WithSession(ctx, s.session, fn)
}

type DBInterface interface {
	RestfulAPIGetOne(collName string, filter bson.M) (map[string]interface{}, error)
	RestfulAPIGetMany(collName string, filter bson.M) ([]map[string]interface{}, error)
	RestfulAPIPutOneTimeout(collName string, filter bson.M, putData map[string]interface{}, timeout int32, timeField string) bool
	RestfulAPIPutOne(collName string, filter bson.M, putData map[string]interface{}) (bool, error)
	RestfulAPIPutOneWithContext(context context.Context, collName string, filter bson.M, putData map[string]interface{}) (bool, error)
	RestfulAPIPutOneNotUpdate(collName string, filter bson.M, putData map[string]interface{}) (bool, error)
	RestfulAPIPutMany(collName string, filterArray []bson.M, putDataArray []map[string]interface{}) error
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
	StartSession() (DBSession, error)
	SupportsTransactions() (bool, error)
	RestfulAPIPostOnDB(ctx context.Context, dbName string, collName string, filter bson.M, postData map[string]interface{}) (bool, error)
	RestfulAPIPutOneOnDB(ctx context.Context, dbName string, collName string, filter bson.M, putData map[string]interface{}) (bool, error)
	RestfulAPIDeleteOneOnDB(ctx context.Context, dbName string, collName string, filter bson.M) error
}

type indexCreator interface {
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
type SessionRunner func(ctx context.Context, fn func(sc context.Context) error) error

func GetSessionRunner(client DBInterface) SessionRunner {
	return func(ctx context.Context, fn func(sc context.Context) error) error {
		session, err := client.StartSession()
		if err != nil {
			return err
		}
		if session == nil {
			return fn(ctx)
		}
		defer session.EndSession(ctx)
		return session.WithSession(ctx, func(sc context.Context) error {
			_, err = session.WithTransaction(sc, func(sc context.Context) (interface{}, error) {
				return nil, fn(sc)
			})
			return err
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
	if errConnect != nil {
		return nil, errConnect
	}
	return &MongoDBClient{*mClient}, nil
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
			return fmt.Errorf("timed out while waiting for replica set or sharded config to be set in MongoDB")
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

	if err := createIndexWithRetry(CommonDBClient, configmodels.UpfDataColl, "hostname", 180*time.Second, 2*time.Second); err != nil {
		logger.InitLog.Errorf("error creating UPF index in commonDB %v", err)
		return err
	}
	if err := createIndexWithRetry(CommonDBClient, configmodels.GnbDataColl, "name", 180*time.Second, 2*time.Second); err != nil {
		logger.InitLog.Errorf("error creating gNB index in commonDB %v", err)
		return err
	}

	if factory.WebUIConfig.Configuration.EnableAuthentication {
		ConnectMongo(mongodb.WebuiDBUrl, mongodb.WebuiDBName, &WebuiDBClient)
		if err := createIndexWithRetry(WebuiDBClient, configmodels.UserAccountDataColl, "username", 180*time.Second, 2*time.Second); err != nil {
			logger.InitLog.Errorf("error initializing webuiDB %v", err)
			return err
		}
	}

	logger.InitLog.Info("MongoDB initialization completed successfully")
	return nil
}

func createIndexWithRetry(client indexCreator, collName string, keyField string, timeout, retryInterval time.Duration) error {
	if client == nil {
		return fmt.Errorf("mongoDB client has not been initialized")
	}

	ticker := time.NewTicker(retryInterval)
	defer ticker.Stop()
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	for {
		resp, err := client.CreateIndex(collName, keyField)
		if err == nil && resp {
			return nil
		}

		if err == nil && !resp {
			err = fmt.Errorf("CreateIndex returned false for %s.%s", collName, keyField)
		}

		if !isRetryableIndexError(err) {
			return err
		}

		logger.InitLog.Warnw("retrying MongoDB index creation",
			"collection", collName,
			"keyField", keyField,
			"error", err)

		select {
		case <-ticker.C:
			continue
		case <-timer.C:
			return fmt.Errorf("timed out creating index for %s.%s: %w", collName, keyField, err)
		}
	}
}

func isRetryableIndexError(err error) bool {
	if err == nil {
		return false
	}

	var commandErr mongo.CommandError
	if errors.As(err, &commandErr) {
		if commandErr.HasErrorLabel("RetryableWriteError") || commandErr.HasErrorLabel("TransientTransactionError") {
			return true
		}
	}

	var writeException mongo.WriteException
	if errors.As(err, &writeException) {
		if writeException.HasErrorLabel("RetryableWriteError") || writeException.HasErrorLabel("TransientTransactionError") {
			return true
		}
		if writeException.WriteConcernError != nil && isRetryableMongoMessage(writeException.WriteConcernError.Message) {
			return true
		}
		for _, writeErr := range writeException.WriteErrors {
			if isRetryableMongoMessage(writeErr.Message) {
				return true
			}
		}
	}

	var serverErr mongo.ServerError
	if errors.As(err, &serverErr) {
		if serverErr.HasErrorLabel("RetryableWriteError") || serverErr.HasErrorLabel("TransientTransactionError") {
			return true
		}
	}

	return isRetryableMongoMessage(err.Error())
}

func isRetryableMongoMessage(message string) bool {
	lowerMsg := strings.ToLower(message)
	transientFragments := []string{
		"interruptedatshutdown",
		"interrupted at shutdown",
		"notwritableprimary",
		"not primary",
		"node is recovering",
		"primary stepped down",
		"connection",
		"server selection",
		"topology",
		"context deadline exceeded",
		"election",
	}

	for _, fragment := range transientFragments {
		if strings.Contains(lowerMsg, fragment) {
			return true
		}
	}

	return false
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

func (db *MongoDBClient) RestfulAPIPutMany(collName string, filterArray []bson.M, putDataArray []map[string]interface{}) error {
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

func (db *MongoDBClient) StartSession() (DBSession, error) {
	session, err := db.MongoClient.StartSession()
	if err != nil || session == nil {
		return nil, err
	}
	return &MongoDBSession{session: session}, nil
}

func (db *MongoDBClient) SupportsTransactions() (bool, error) {
	return db.MongoClient.SupportsTransactions()
}

func (db *MongoDBClient) RestfulAPIPostOnDB(ctx context.Context, dbName string, collName string, filter bson.M, postData map[string]interface{}) (bool, error) {
	collection := db.Client.Database(dbName).Collection(collName)
	var existing bson.M
	err := collection.FindOne(ctx, filter).Decode(&existing)
	if err != nil && err != mongo.ErrNoDocuments {
		return false, fmt.Errorf("RestfulAPIPostOnDB FindOne err: %w", err)
	}
	if existing != nil {
		if _, err := collection.UpdateOne(ctx, filter, bson.M{"$set": postData}); err != nil {
			return false, fmt.Errorf("RestfulAPIPostOnDB UpdateOne err: %w", err)
		}
		return true, nil
	}
	if _, err := collection.InsertOne(ctx, postData); err != nil {
		return false, fmt.Errorf("RestfulAPIPostOnDB InsertOne err: %w", err)
	}
	return false, nil
}

func (db *MongoDBClient) RestfulAPIPutOneOnDB(ctx context.Context, dbName string, collName string, filter bson.M, putData map[string]interface{}) (bool, error) {
	collection := db.Client.Database(dbName).Collection(collName)
	var existing bson.M
	err := collection.FindOne(ctx, filter).Decode(&existing)
	if err != nil && err != mongo.ErrNoDocuments {
		return false, fmt.Errorf("RestfulAPIPutOneOnDB FindOne err: %w", err)
	}
	if existing != nil {
		if _, err := collection.UpdateOne(ctx, filter, bson.M{"$set": putData}); err != nil {
			return false, fmt.Errorf("RestfulAPIPutOneOnDB UpdateOne err: %w", err)
		}
		return true, nil
	}
	if _, err := collection.InsertOne(ctx, putData); err != nil {
		return false, fmt.Errorf("RestfulAPIPutOneOnDB InsertOne err: %w", err)
	}
	return false, nil
}

func (db *MongoDBClient) RestfulAPIDeleteOneOnDB(ctx context.Context, dbName string, collName string, filter bson.M) error {
	collection := db.Client.Database(dbName).Collection(collName)
	if _, err := collection.DeleteOne(ctx, filter); err != nil {
		return fmt.Errorf("RestfulAPIDeleteOneOnDB err: %w", err)
	}
	return nil
}
