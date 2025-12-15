/**
 * JavaScript equivalent of Go structs from config_msg.go
 */

import { DeviceGroups } from "./model_device_groups";
import { Slice } from "./model_slice";

// Constants matching Go enums
export const OperationType = {
  POST: 0,
  PUT: 1,
  DELETE: 2
};

export const GroupType = {
  DEVICE_GROUP: 0,
  NETWORK_SLICE: 1,
  SUB_DATA: 2
};

/**
 * ConfigMessage represents configuration messages for different entities
 */
export class ConfigMessage {
  constructor() {
    /** @type {DeviceGroups|null} */
    this.DevGroup = null;       // DeviceGroups
    /** @type {Slice|null} */
    this.Slice = null;          // Slice
    // TODO: implement the object *models.AuthenticationSubscription
    this.AuthSubData = null;    // AuthenticationSubscription
    this.DevGroupName = '';
    this.SliceName = '';
    this.Imsi = '';
    this.MsgType = 0;
    this.MsgMethod = 0;
  }
}

/**
 * Represents a slice with its attached device groups
 */
export class SliceConfigSnapshot {
  constructor() {
    /** @type {Slice|null} */
    this.SliceMsg = null;      // Slice
    /** @type {Array<DeviceGroups>|null} */
    this.DevGroup = [];        // Array of DeviceGroups
  }
}

/**
 * Represents a device group with its slice name
 */
export class DevGroupConfigSnapshot {
  constructor() {
    /** @type {DeviceGroups|null} */
    this.DevGroup = null;      // DeviceGroups
    this.SliceName = '';
  }
}
