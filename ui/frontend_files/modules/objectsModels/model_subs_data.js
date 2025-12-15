/**
 * JavaScript equivalent of Go structs from model_subs_data.go
 */

import { FlowRule } from "./model_flow_rule";

// TODO: implement models
// AuthenticationSubscription        models.AuthenticationSubscription          `json:"AuthenticationSubscription"`
// AccessAndMobilitySubscriptionData models.AccessAndMobilitySubscriptionData   `json:"AccessAndMobilitySubscriptionData"`
// SessionManagementSubscriptionData []models.SessionManagementSubscriptionData `json:"SessionManagementSubscriptionData"`
// SmfSelectionSubscriptionData      models.SmfSelectionSubscriptionData        `json:"SmfSelectionSubscriptionData"`
// AmPolicyData                      models.AmPolicyData                        `json:"AmPolicyData"`
// SmPolicyData                      models.SmPolicyData  
export const SubsData = () => {
  return {
    plmnID: "",
    ueId: "",
    AuthenticationSubscription: {},
    AccessAndMobilitySubscriptionData: {},
    SessionManagementSubscriptionData: [],
    SmfSelectionSubscriptionData: {},
    AmPolicyData: {},
    SmPolicyData: {},
    FlowRules: [] // FlowRule|null
  };
};

export const SubsOverrideData = () => {
  return {
    "plmnID": "",
    "opc": "",
    "key": "",
    "sequenceNumber": "",
    "k4_sno": null,
    "encryptionAlgorithm": null
  };
};
