/**
 * JavaScript equivalent of Go structs from model_site.go
 */

export const SiteInfo = () => {
  return {
    "site-name": "",     // Name of the site
    "plmn": {            // PLMN information
      "mcc": "",         // Mobile Country Code
      "mnc": ""          // Mobile Network Code
    },
    "gNodeBs": [],       // List of gNodeBs
    "upf": {}            // UPF configuration
  };
};

export const SitesList = () => {
  return [];  // Array of site names
};
