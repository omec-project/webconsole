/**
 * JavaScript equivalent of Go structs from model_device_groups_ip_domain_expanded.go
 */

import { DeviceGroupsIpDomainExpandedUeDnnQos } from "./model_device_groups_ip_domain_expanded_ue_dnn_qos";

export class DeviceGroupsIpDomainExpanded {
  constructor() {
    this.dnn = "";
    this["ue-ip-pool"] = "";
    this["dns-primary"] = "";
    this["dns-secondary"] = "";
    this.mtu = 0;
    /** @type {DeviceGroupsIpDomainExpandedUeDnnQos|null} */
    this["ue-dnn-qos"] = null;
  }
}