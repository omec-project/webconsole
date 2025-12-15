/**
 * JavaScript equivalent of Go structs from model_slice_site_info.go
 */

import { SliceSiteInfoPlmn } from "./model_slice_site_info_plmn";
import { SliceSiteInfoGNodeBs } from "./model_slice_site_info_g_node_bs";

export class SliceSiteInfo {
  constructor() {
    this.siteName = "";  // Unique name per Site
    /** @type {SliceSiteInfoPlmn|null} */
    this.plmn = {};
    /** @type {Array<SliceSiteInfoGNodeBs>|null} */
    this.gNodeBs = [];
    this.upf = {};           // UPF which belong to this slice
  }
}
