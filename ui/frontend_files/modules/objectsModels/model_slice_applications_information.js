/**
 * JavaScript equivalent of Go structs from model_slice_applications_information.go
 */

export class SliceApplicationsInformation {
  constructor() {
    this.appName = "";     // Single App or group of application identification
    this.endpoint = "";        // Single IP or network
    this.startPort = 0;    // port range start
    this.endPort = 0;      // port range end
    this.protocol = 0;
  }
}