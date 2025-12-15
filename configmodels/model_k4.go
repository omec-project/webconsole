package configmodels

import "time"

type K4 struct {
	K4       string `json:"k4" bson:"k4"`
	K4_SNO   byte   `json:"k4_sno" bson:"k4_sno"`
	K4_Label string `json:"key_label,omitempty" bson:"key_label,omitempty"`
	K4_Type  string `json:"key_type,omitempty" bson:"key_type,omitempty"`
	// Creation timestamp in RFC3339
	TimeCreated time.Time `json:"time_created"`
	// Update timestamp in RFC3339
	TimeUpdated time.Time `json:"time_updated"`
}
