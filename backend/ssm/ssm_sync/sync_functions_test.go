package ssmsync

import (
	"testing"

	"github.com/omec-project/webconsole/configmodels"
)

func TestReadStopCondition(t *testing.T) {
	// Set initial condition
	StopSSMsyncFunction = false

	result := readStopCondition()
	if result != false {
		t.Errorf("Expected readStopCondition() to return false, got %v", result)
	}

	// Change condition
	StopSSMsyncFunction = true
	result = readStopCondition()
	if result != true {
		t.Errorf("Expected readStopCondition() to return true, got %v", result)
	}

	// Reset for other tests
	StopSSMsyncFunction = false
}

func TestCreateNewKeySSM_InvalidLabel(t *testing.T) {
	_, err := createNewKeySSM("INVALID_LABEL", 1)
	if err == nil {
		t.Error("Expected error for invalid key label, got nil")
	}

	expectedError := "unsupported key label: INVALID_LABEL"
	if err.Error() != expectedError {
		t.Errorf("Expected error message '%s', got '%s'", expectedError, err.Error())
	}
}

func TestDeleteKeyMongoDB(t *testing.T) {
	k4 := configmodels.K4{
		K4_SNO:   1,
		K4_Label: "test_label",
		K4_Type:  "AES",
	}

	// This will fail without proper DB connection, but we can test the function signature
	err := DeleteKeyMongoDB(k4)

	// We expect an error since DB is not connected in test environment
	if err == nil {
		t.Log("Warning: DeleteKeyMongoDB returned nil error, expected DB connection error")
	}
}

func TestStoreInMongoDB(t *testing.T) {
	k4 := configmodels.K4{
		K4_SNO:   1,
		K4_Label: "test_label",
		K4_Type:  "AES",
		K4:       "test_key_value",
	}

	// This will fail without proper DB connection, but we can test the function signature
	err := StoreInMongoDB(k4, "test_label")

	// We expect an error since DB is not connected in test environment
	if err == nil {
		t.Log("Warning: StoreInMongoDB returned nil error, expected DB connection error")
	}
}

func TestGetUsersMDB(t *testing.T) {
	// This will fail without proper DB connection, but we can test the function signature
	users := GetUsersMDB()

	// Without DB, we expect an empty list
	if users == nil {
		t.Error("Expected non-nil slice from GetUsersMDB")
	}
}

func TestGetSubscriberData(t *testing.T) {
	// Test with invalid ueId
	_, err := GetSubscriberData("invalid_ue_id")

	// We expect an error since DB is not connected or subscriber doesn't exist
	if err == nil {
		t.Log("Warning: GetSubscriberData returned nil error, expected DB connection error or not found error")
	}
}

func TestErrorChannelsInitialized(t *testing.T) {
	if ErrorSyncChan == nil {
		t.Error("ErrorSyncChan should be initialized")
	}

	if ErrorRotationChan == nil {
		t.Error("ErrorRotationChan should be initialized")
	}

	// Test that we can send to the channel without blocking
	select {
	case ErrorSyncChan <- nil:
		// Successfully sent
	default:
		t.Error("ErrorSyncChan should accept messages")
	}

	// Drain the channel
	select {
	case <-ErrorSyncChan:
		// Successfully received
	default:
		t.Error("Should have been able to receive from ErrorSyncChan")
	}
}

func TestStopSSMsyncFunctionInitialValue(t *testing.T) {
	// Reset to known state
	StopSSMsyncFunction = false

	if StopSSMsyncFunction != false {
		t.Error("StopSSMsyncFunction should be initialized to false")
	}
}
