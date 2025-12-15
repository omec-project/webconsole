/**
 * JavaScript equivalent of Go structs from model_slice_qos.go
 */

export class SliceQos {
  constructor() {
    this.uplink = 0;               // uplink data rate in bps
    this.downlink = 0;             // downlink data rate in bps
    this.bitrateUnit = "";     // data rate unit for uplink and downlink
    this.traffiClass = "";    // QCI/QFI for the traffic
  }
}
