/**
 * JavaScript equivalent of Go structs from model_device_groups_ip_domain_expanded_ue_dnn_qos.go
 */

import { TrafficClassInfo } from "./model_traffic_class";

export const DeviceGroupsIpDomainExpandedUeDnnQos = () => {
  return {
    "dnn-mbr-uplink": 0,         // uplink data rate
    "dnn-mbr-downlink": 0,       // downlink data rate
    "bitrate-unit": "",         // data rate unit for uplink and downlink
    "traffic-class": null,      // QCI/QFI for the traffic
  };
};