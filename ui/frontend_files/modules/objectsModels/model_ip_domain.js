/**
 * JavaScript equivalent of Go structs from model_ip_domain.go
 */

export const IpDomain = () => {
  return {
    dnn: "",
    ueIpPool: "",
    dnsPrimary: "",
    dnsSecondary: "",
    mtu: 1500
  };
};

export const IpDomainExpanded = () => {
  return {
    dnn: "",             // Data Network Name
    "ue-ip-pool": "",    // UE IP Pool in CIDR notation
    "dns-primary": "",   // Primary DNS server
    "dns-secondary": "", // Secondary DNS server
    mtu: 1500,           // Maximum Transmission Unit
    "ue-dnn-qos": null   // QoS information for this DNN
  };
};
