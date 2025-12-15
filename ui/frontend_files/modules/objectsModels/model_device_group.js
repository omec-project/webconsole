/**
 * JavaScript equivalent of Go structs from model_device_group.go
 */

import { IpDomainExpanded } from "./model_ip_domain.js";

export const DeviceGroup = () => {
  return {
    "group-name": "",        // Name of the device group
    "imsis": [],             // List of IMSIs belonging to this group
    "ip-domain-name": "",    // IP domain name
    "ip-domain-expanded": IpDomainExpanded(),  // Expanded IP domain configuration
    "site-info": ""          // Site information
  };
};

export const DeviceGroupsList = () => {
  return [];  // Array of device group names
};
