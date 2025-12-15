/**
 * JavaScript equivalent of Go structs from model_network_slice.go
 */

import { ApplicationFilteringRule } from "./model_application_filtering_rules.js";

export const NetworkSlice = () => {
  return {
    "slice-name": "",                       // Name of the slice
    "slice-id": {                           // Slice ID information
      "sst": "",                            // Slice Service Type
      "sd": ""                              // Slice Differentiator
    },
    "site-device-group": [],                // List of device groups for this slice
    "site-info": {                          // Site information
      "site-name": "",                      // Name of the site
      "plmn": {                             // PLMN information
        "mcc": "",                          // Mobile Country Code
        "mnc": ""                           // Mobile Network Code
      },
      "gNodeBs": [],                        // List of gNodeBs
      "upf": {}                             // UPF configuration
    },
    "application-filtering-rules": []       // List of application filtering rules
  };
};

export const NetworkSlicesList = () => {
  return [];  // Array of network slice names
};

export const GNodeB = () => {
  return {
    "name": "",  // Name of the gNodeB
    "tac": 0     // Tracking Area Code
  };
};
