/**
 * JavaScript equivalent of Go structs from model_slice_apn_ambr_qos.go
 */

export class ApnAmbrQosInfo {
  constructor() {
    this.uplink = 0;
    this.downlink = 0;
    this.bitrateUnit = "";
    this.trafficClass = "";
  }
}