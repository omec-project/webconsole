package configapi

import (
	"encoding/json"
	"os"
	"os/exec"
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

type MockMongoPost struct {
	dbadapter.DBInterface
}

type MockMongoDeleteOne struct {
	dbadapter.DBInterface
}

type MockMongoGetOneNil struct {
	dbadapter.DBInterface
}

type MockMongoGetManyNil struct {
	dbadapter.DBInterface
}

type MockMongoGetManyGroups struct {
	dbadapter.DBInterface
}

type MockMongoGetManySlices struct {
	dbadapter.DBInterface
}

type MockMongoDeviceGroupGetOne struct {
	dbadapter.DBInterface
	testGroup configmodels.DeviceGroups
}

type MockMongoSliceGetOne struct {
	dbadapter.DBInterface
	testSlice configmodels.Slice
}

type MockMongoSubscriberGetOne struct {
	dbadapter.DBInterface
	testSubscriber bson.M
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

func (m *MockMongoGetOneNil) RestfulAPIGetOne(collName string, filter bson.M) (map[string]interface{}, error) {
	var value map[string]interface{}
	return value, nil
}

func (m *MockMongoDeviceGroupGetOne) RestfulAPIGetOne(collName string, filter bson.M) (map[string]interface{}, error) {
	var previousGroupBson bson.M
	previousGroup, err := json.Marshal(m.testGroup)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(previousGroup, &previousGroupBson)
	if err != nil {
		return nil, err
	}
	return previousGroupBson, nil
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
	cs := []string{"-test.run=TestExecCommandHelper", "--", "ENTER YOUR COMMAND HERE"}
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
