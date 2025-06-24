package configapi

import (
	"encoding/json"
	"go.mongodb.org/mongo-driver/mongo"
	"os"
	"os/exec"
	"reflect"
	"testing"

	"github.com/omec-project/webconsole/backend/factory"
	"github.com/omec-project/webconsole/configmodels"
	"github.com/omec-project/webconsole/dbadapter"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

var (
	execCommandTimesCalled = 0
	postData               []map[string]interface{}
	deleteData             []map[string]interface{}
)

type MockMongoGetOneNil struct {
	dbadapter.DBInterface
}

func (m *MockMongoGetOneNil) RestfulAPIGetOne(collName string, filter bson.M) (map[string]interface{}, error) {
	var value map[string]interface{}
	return value, nil
}

type MockMongoSliceGetOne struct {
	dbadapter.DBInterface
	testSlice configmodels.Slice
}

func (m *MockMongoSliceGetOne) RestfulAPIGetOne(collName string, filter bson.M) (map[string]interface{}, error) {
	var previousSliceBson bson.M
	previousSlice, err := json.Marshal(m.testSlice)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(previousSlice, &previousSliceBson)
	if err != nil {
		return nil, err
	}
	return previousSliceBson, nil
}

type MockMongoSubscriberGetOne struct {
	dbadapter.DBInterface
	testSubscriber bson.M
}

func (m *MockMongoSubscriberGetOne) RestfulAPIGetOne(collName string, filter bson.M) (map[string]interface{}, error) {
	var previousSubscriberBson bson.M
	previousSubscriber, err := json.Marshal(m.testSubscriber)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(previousSubscriber, &previousSubscriberBson)
	if err != nil {
		return nil, err
	}
	return previousSubscriberBson, nil
}

func mockExecCommand(command string, args ...string) *exec.Cmd {
	cs := []string{"-test.run=TestExecCommandHelper", "--", "YOUR COMMAND"}
	cs = append(cs, args...)
	cmd := exec.Command(os.Args[0], cs...)
	execCommandTimesCalled += 1
	return cmd
}

func Test_sendPebbleNotification_on_when_handleNetworkSlicePost(t *testing.T) {
	execCommand = mockExecCommand
	defer func() { execCommand = exec.Command }()

	origSync := syncSliceDeviceGroupSubscribers
	syncSliceDeviceGroupSubscribers = func(_, _ *configmodels.Slice) error { return nil }
	defer func() { syncSliceDeviceGroupSubscribers = origSync }()

	numPebbleNotificationsSent := execCommandTimesCalled

	slice := networkSlice("slice1")
	var prevSlice *configmodels.Slice = nil

	factory.WebUIConfig.Configuration.SendPebbleNotifications = true
	dbadapter.CommonDBClient = &MockMongoPost{}

	err := handleNetworkSlicePost(&slice, prevSlice)
	if err != nil {
		t.Errorf("Could not handle network slice post: %v", err)
	}
	if execCommandTimesCalled != numPebbleNotificationsSent+1 {
		t.Errorf("Unexpected number of Pebble notifications: %v. Should be: %v", execCommandTimesCalled, numPebbleNotificationsSent+1)
	}
}

func Test_sendPebbleNotification_off_when_handleNetworkSlicePost(t *testing.T) {
	execCommand = mockExecCommand
	defer func() { execCommand = exec.Command }()
	execCommandTimesCalled = 0

	origSync := syncSliceDeviceGroupSubscribers
	syncSliceDeviceGroupSubscribers = func(_, _ *configmodels.Slice) error { return nil }
	defer func() { syncSliceDeviceGroupSubscribers = origSync }()

	factory.WebUIConfig = &factory.Config{
		Configuration: &factory.Configuration{
			SendPebbleNotifications: false,
		},
	}

	slice := &configmodels.Slice{SliceName: "slice1"}
	prevSlice := &configmodels.Slice{}

	dbadapter.CommonDBClient = &MockMongoPost{}

	err := handleNetworkSlicePost(slice, prevSlice)
	if err != nil {
		t.Errorf("handleNetworkSlicePost returned error: %v", err)
	}

	if execCommandTimesCalled != 0 {
		t.Errorf("Expected 0 Pebble notifications, but got %v", execCommandTimesCalled)
	}
}

type MockMongoPost struct {
	dbadapter.DBInterface
}

func (m *MockMongoPost) RestfulAPIPost(coll string, filter primitive.M, data map[string]interface{}) (bool, error) {
	params := map[string]interface{}{
		"coll":   coll,
		"filter": filter,
		"data":   data,
	}
	postData = append(postData, params)
	return true, nil
}

func (m *MockMongoPost) RestfulAPIGetMany(collName string, filter bson.M) ([]map[string]interface{}, error) {
	return []map[string]interface{}{}, nil
}

type MockCombinedDB struct {
	dbadapter.DBInterface
	testSlice configmodels.Slice
}

func (m *MockCombinedDB) RestfulAPIPost(coll string, filter primitive.M, data map[string]interface{}) (bool, error) {
	params := map[string]interface{}{
		"coll":   coll,
		"filter": filter,
		"data":   data,
	}
	postData = append(postData, params)
	return true, nil
}

func (m *MockCombinedDB) RestfulAPIGetMany(collName string, filter bson.M) ([]map[string]interface{}, error) {
	return []map[string]interface{}{}, nil
}

func structToMap(obj interface{}) (map[string]interface{}, error) {
	var result map[string]interface{}
	data, err := json.Marshal(obj)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(data, &result)
	if err != nil {
		return nil, err
	}
	return result, nil
}
func (m *MockCombinedDB) RestfulAPIGetOne(collName string, filter bson.M) (map[string]interface{}, error) {
	return structToMap(m.testSlice)
}

func (m *MockCombinedDB) Client() *mongo.Client {
	return nil
}

func Test_handleNetworkSlicePost(t *testing.T) {
	networkSlices := []configmodels.Slice{networkSlice("slice1"), networkSlice("slice2"),
		networkSlice("slice_no_gnodeb"), networkSlice("slice_no_device_groups")}
	networkSlices[2].SiteInfo.GNodeBs = []configmodels.SliceSiteInfoGNodeBs{}
	networkSlices[3].SiteDeviceGroup = []string{}
	factory.WebUIConfig.Configuration.Mode5G = true

	for _, testSlice := range networkSlices {
		ts := testSlice // capture loop variable
		t.Run(ts.SliceName, func(t *testing.T) {
			postData = make([]map[string]interface{}, 0)
			mock := &MockCombinedDB{
				testSlice: ts,
			}
			dbadapter.CommonDBClient = mock

			postErr := handleNetworkSlicePost(&testSlice, nil)

			if postErr != nil {
				t.Errorf("Could not handle network slice post: %v", postErr)
			}

			if len(postData) == 0 {
				t.Fatal("No post operation was recorded")
			}

			expected_collection := sliceDataColl
			if postData[0]["coll"] != expected_collection {
				t.Errorf("Expected collection %v, got %v", expected_collection, postData[0]["coll"])
			}

			expected_filter := bson.M{"slice-name": testSlice.SliceName}
			if !reflect.DeepEqual(postData[0]["filter"], expected_filter) {
				t.Errorf("Expected filter %v, got %v", expected_filter, postData[0]["filter"])
			}

			var resultSlice configmodels.Slice
			result := postData[0]["data"].(map[string]interface{})
			bytes, err := json.Marshal(result)
			if err != nil {
				t.Errorf("Could not marshal result map: %v", err)
			}
			err = json.Unmarshal(bytes, &resultSlice)
			if err != nil {
				t.Errorf("Could not unmarshal result %v", result)
			}
			if !reflect.DeepEqual(resultSlice, testSlice) {
				t.Errorf("Expected slice %v, got %v", testSlice, resultSlice)
			}
		})
	}
}

type MockMongoDeleteOne struct {
	dbadapter.DBInterface
}

func (m *MockMongoDeleteOne) RestfulAPIDeleteOne(coll string, filter primitive.M) error {
	params := map[string]interface{}{
		"coll":   coll,
		"filter": filter,
	}
	deleteData = append(deleteData, params)
	return nil
}

func (m *MockMongoDeleteOne) RestfulAPIGetOne(coll string, filter bson.M) (map[string]interface{}, error) {
	return map[string]interface{}{}, nil
}

func Test_handleNetworkSlicePost_alreadyExists(t *testing.T) {
	networkSlices := []configmodels.Slice{networkSlice("slice1"), networkSlice("slice2"),
		networkSlice("slice_no_gnodeb"), networkSlice("slice_no_device_groups")}
	networkSlices[2].SiteInfo.GNodeBs = []configmodels.SliceSiteInfoGNodeBs{}
	networkSlices[3].SiteDeviceGroup = []string{}
	factory.WebUIConfig.Configuration.Mode5G = true

	for _, testSlice := range networkSlices {
		ts := testSlice

		t.Run(ts.SliceName, func(t *testing.T) {
			postData = make([]map[string]interface{}, 0)

			mock := &MockCombinedDB{testSlice: ts}
			dbadapter.CommonDBClient = mock

			err := handleNetworkSlicePost(&ts, &ts)

			if err != nil {
				t.Fatalf("handleNetworkSlicePost returned error: %v", err)
			}

			if len(postData) == 0 {
				t.Fatal("Expected a post operation but none was recorded")
			}

			if postData[0]["coll"] != sliceDataColl {
				t.Errorf("Expected collection %v, got %v", sliceDataColl, postData[0]["coll"])
			}

			expectedFilter := bson.M{"slice-name": ts.SliceName}
			if !reflect.DeepEqual(postData[0]["filter"], expectedFilter) {
				t.Errorf("Expected filter %v, got %v", expectedFilter, postData[0]["filter"])
			}

			result := postData[0]["data"].(map[string]interface{})
			bytes, err := json.Marshal(result)
			if err != nil {
				t.Fatalf("Failed to marshal result data: %v", err)
			}
			var resultSlice configmodels.Slice
			if err := json.Unmarshal(bytes, &resultSlice); err != nil {
				t.Fatalf("Failed to unmarshal result data: %v", err)
			}
			if !reflect.DeepEqual(resultSlice, ts) {
				t.Errorf("Expected slice %v, got %v", ts, resultSlice)
			}
		})
	}
}
