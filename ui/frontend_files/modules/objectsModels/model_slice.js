/**
 * JavaScript equivalent of Go structs from model_slice.go
 */

import { SliceSliceId } from "./model_slice_slice_id";
import { SliceSiteInfo } from "./model_slice_site_info";
import { SliceApplicationFilteringRules } from "./model_application_filtering_rules";

export class Slice {
  constructor() {
    this.sliceName = "";
    /** @type {SliceSliceId|null} */
    this.sliceId = {};
    this.siteDeviceGroup = [];
    /** @type {SliceSiteInfo|null} */
    this.siteInfo = {};
    /** @type {Array<SliceApplicationFilteringRules>|null} */
    this.applicationFilteringRules = [];
  }
}