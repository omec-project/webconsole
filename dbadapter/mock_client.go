package dbadapter

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// MockDBClient is a mock implementation of the database client for testing
type MockDBClient struct {
	Docs                   []map[string]any
	GetManyFn              func(collName string, filter bson.M) ([]map[string]any, error)
	GetOneFn               func(collName string, filter bson.M) (map[string]any, error)
	PostFn                 func(collName string, filter bson.M, postData map[string]any) (bool, error)
	PostWithContextFn      func(ctx context.Context, collName string, filter bson.M, postData map[string]any) (bool, error)
	PostManyFn             func(collName string, filter bson.M, postDataArray []any) error
	PostManyWithContextFn  func(ctx context.Context, collName string, filter bson.M, postDataArray []any) error
	PutOneFn               func(collName string, filter bson.M, putData map[string]any) (bool, error)
	PutOneWithContextFn    func(ctx context.Context, collName string, filter bson.M, putData map[string]any) (bool, error)
	PutOneTimeoutFn        func(collName string, filter bson.M, putData map[string]any, timeout int32, timeField string) bool
	PutOneNotUpdateFn      func(collName string, filter bson.M, putData map[string]any) (bool, error)
	PutManyFn              func(collName string, filterArray []primitive.M, putDataArray []map[string]any) error
	DeleteOneFn            func(collName string, filter bson.M) error
	DeleteOneWithContextFn func(ctx context.Context, collName string, filter bson.M) error
	DeleteManyFn           func(collName string, filter bson.M) error
	MergePatchFn           func(collName string, filter bson.M, patchData map[string]any) error
	JSONPatchFn            func(collName string, filter bson.M, patchJSON []byte) error
	JSONPatchWithContextFn func(ctx context.Context, collName string, filter bson.M, patchJSON []byte) error
	JSONPatchExtendFn      func(collName string, filter bson.M, patchJSON []byte, dataName string) error
	CountFn                func(collName string, filter bson.M) (int64, error)
	PullOneFn              func(collName string, filter bson.M, putData map[string]any) error
	PullOneWithContextFn   func(ctx context.Context, collName string, filter bson.M, putData map[string]any) error
	CreateIndexFn          func(collName string, keyField string) (bool, error)
	StartSessionFn         func() (mongo.Session, error)
	SupportsTransactionsFn func() (bool, error)
}

// RestfulAPIGetMany implements the mock version of GetMany
func (m *MockDBClient) RestfulAPIGetMany(collName string, filter bson.M) ([]map[string]any, error) {
	if m.GetManyFn != nil {
		return m.GetManyFn(collName, filter)
	}
	return nil, nil
}

// RestfulAPIGetOne implements the mock version of GetOne
func (m *MockDBClient) RestfulAPIGetOne(collName string, filter bson.M) (map[string]any, error) {
	if m.GetOneFn != nil {
		return m.GetOneFn(collName, filter)
	}
	return nil, nil
}

// RestfulAPIPost implements the mock version of Post
func (m *MockDBClient) RestfulAPIPost(collName string, filter bson.M, postData map[string]any) (bool, error) {
	if m.PostFn != nil {
		return m.PostFn(collName, filter, postData)
	}
	return false, nil
}

// RestfulAPIPostWithContext implements the mock version of PostWithContext
func (m *MockDBClient) RestfulAPIPostWithContext(ctx context.Context, collName string, filter bson.M, postData map[string]any) (bool, error) {
	if m.PostWithContextFn != nil {
		return m.PostWithContextFn(ctx, collName, filter, postData)
	}
	return false, nil
}

// RestfulAPIPostMany implements the mock version of PostMany
func (m *MockDBClient) RestfulAPIPostMany(collName string, filter bson.M, postDataArray []any) error {
	if m.PostManyFn != nil {
		return m.PostManyFn(collName, filter, postDataArray)
	}
	return nil
}

// RestfulAPIPostManyWithContext implements the mock version of PostManyWithContext
func (m *MockDBClient) RestfulAPIPostManyWithContext(ctx context.Context, collName string, filter bson.M, postDataArray []any) error {
	if m.PostManyWithContextFn != nil {
		return m.PostManyWithContextFn(ctx, collName, filter, postDataArray)
	}
	return nil
}

// RestfulAPIPutOne implements the mock version of PutOne
func (m *MockDBClient) RestfulAPIPutOne(collName string, filter bson.M, putData map[string]any) (bool, error) {
	if m.PutOneFn != nil {
		return m.PutOneFn(collName, filter, putData)
	}
	return false, nil
}

// RestfulAPIPutOneTimeout implements the mock version of PutOneTimeout
func (m *MockDBClient) RestfulAPIPutOneTimeout(collName string, filter bson.M, putData map[string]any, timeout int32, timeField string) bool {
	if m.PutOneTimeoutFn != nil {
		return m.PutOneTimeoutFn(collName, filter, putData, timeout, timeField)
	}
	return true
}

// RestfulAPIPutOneWithContext implements the mock version of PutOneWithContext
func (m *MockDBClient) RestfulAPIPutOneWithContext(ctx context.Context, collName string, filter bson.M, putData map[string]any) (bool, error) {
	if m.PutOneWithContextFn != nil {
		return m.PutOneWithContextFn(ctx, collName, filter, putData)
	}
	return false, nil
}

// RestfulAPIPutOneNotUpdate implements the mock version of PutOneNotUpdate
func (m *MockDBClient) RestfulAPIPutOneNotUpdate(collName string, filter bson.M, putData map[string]any) (bool, error) {
	if m.PutOneNotUpdateFn != nil {
		return m.PutOneNotUpdateFn(collName, filter, putData)
	}
	return false, nil
}

// RestfulAPIPutMany implements the mock version of PutMany
func (m *MockDBClient) RestfulAPIPutMany(collName string, filterArray []primitive.M, putDataArray []map[string]any) error {
	if m.PutManyFn != nil {
		return m.PutManyFn(collName, filterArray, putDataArray)
	}
	return nil
}

// RestfulAPIDeleteOne implements the mock version of DeleteOne
func (m *MockDBClient) RestfulAPIDeleteOne(collName string, filter bson.M) error {
	if m.DeleteOneFn != nil {
		return m.DeleteOneFn(collName, filter)
	}
	return nil
}

// RestfulAPIDeleteOneWithContext implements the mock version of DeleteOneWithContext
func (m *MockDBClient) RestfulAPIDeleteOneWithContext(ctx context.Context, collName string, filter bson.M) error {
	if m.DeleteOneWithContextFn != nil {
		return m.DeleteOneWithContextFn(ctx, collName, filter)
	}
	return nil
}

// RestfulAPIDeleteMany implements the mock version of DeleteMany
func (m *MockDBClient) RestfulAPIDeleteMany(collName string, filter bson.M) error {
	if m.DeleteManyFn != nil {
		return m.DeleteManyFn(collName, filter)
	}
	return nil
}

// RestfulAPIMergePatch implements the mock version of MergePatch
func (m *MockDBClient) RestfulAPIMergePatch(collName string, filter bson.M, patchData map[string]any) error {
	if m.MergePatchFn != nil {
		return m.MergePatchFn(collName, filter, patchData)
	}
	return nil
}

// RestfulAPIJSONPatch implements the mock version of JSONPatch
func (m *MockDBClient) RestfulAPIJSONPatch(collName string, filter bson.M, patchJSON []byte) error {
	if m.JSONPatchFn != nil {
		return m.JSONPatchFn(collName, filter, patchJSON)
	}
	return nil
}

// RestfulAPIJSONPatchWithContext implements the mock version of JSONPatchWithContext
func (m *MockDBClient) RestfulAPIJSONPatchWithContext(ctx context.Context, collName string, filter bson.M, patchJSON []byte) error {
	if m.JSONPatchWithContextFn != nil {
		return m.JSONPatchWithContextFn(ctx, collName, filter, patchJSON)
	}
	return nil
}

// RestfulAPIJSONPatchExtend implements the mock version of JSONPatchExtend
func (m *MockDBClient) RestfulAPIJSONPatchExtend(collName string, filter bson.M, patchJSON []byte, dataName string) error {
	if m.JSONPatchExtendFn != nil {
		return m.JSONPatchExtendFn(collName, filter, patchJSON, dataName)
	}
	return nil
}

// RestfulAPICount implements the mock version of Count
func (m *MockDBClient) RestfulAPICount(collName string, filter bson.M) (int64, error) {
	if m.CountFn != nil {
		return m.CountFn(collName, filter)
	}
	return 0, nil
}

// RestfulAPIPullOne implements the mock version of PullOne
func (m *MockDBClient) RestfulAPIPullOne(collName string, filter bson.M, putData map[string]any) error {
	if m.PullOneFn != nil {
		return m.PullOneFn(collName, filter, putData)
	}
	return nil
}

// RestfulAPIPullOneWithContext implements the mock version of PullOneWithContext
func (m *MockDBClient) RestfulAPIPullOneWithContext(ctx context.Context, collName string, filter bson.M, putData map[string]any) error {
	if m.PullOneWithContextFn != nil {
		return m.PullOneWithContextFn(ctx, collName, filter, putData)
	}
	return nil
}

// CreateIndex implements the mock version of CreateIndex
func (m *MockDBClient) CreateIndex(collName string, keyField string) (bool, error) {
	if m.CreateIndexFn != nil {
		return m.CreateIndexFn(collName, keyField)
	}
	return true, nil
}

// StartSession implements the mock version of StartSession
func (m *MockDBClient) StartSession() (mongo.Session, error) {
	if m.StartSessionFn != nil {
		return m.StartSessionFn()
	}
	return nil, nil
}

// SupportsTransactions implements the mock version of SupportsTransactions
func (m *MockDBClient) SupportsTransactions() (bool, error) {
	if m.SupportsTransactionsFn != nil {
		return m.SupportsTransactionsFn()
	}
	return true, nil
}
