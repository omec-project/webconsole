/**
 * JavaScript equivalent of Go structs from model_device_groups.go
 */

import { DeviceGroupsIpDomainExpanded } from "./model_device_groups_ip_domain_expanded";

export class DeviceGroups {
  constructor() {
    this["group-name"] = "";
    this.imsis = [];
    this["site-info"] = "";
    this["ip-domain-name"] = "";
    /** @type {DeviceGroupsIpDomainExpanded|null} */
    this["ip-domain-expanded"] = {};
  }
}
