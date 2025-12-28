/**
 * JavaScript equivalent of Go structs from model_inventory.go
 */

export const GNB_DATA_COLL = "webconsoleData.snapshots.gnbData";
export const UPF_DATA_COLL = "webconsoleData.snapshots.upfData";

export const Gnb = () => {
  return {
    name: "",
    tac: 0
  };
};

export const PostGnbRequest = () => {
  return {
    name: "",
    tac: 0
  };
};

export const PutGnbRequest = () => {
  return {
    tac: 0
  };
};

export const Upf = () => {
  return {
    hostname: "",
    port: ""
  };
};

export const PostUpfRequest = (hostname = "", port = "") => {
  return {
    hostname,
    port
  };
};

export const PutUpfRequest = () => {
  return {
    port: ""
  };
};
