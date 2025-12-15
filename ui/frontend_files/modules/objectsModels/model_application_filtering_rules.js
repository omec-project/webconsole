/**
 * JavaScript equivalent of Go structs from model_application_filtering_rules.go
 */

export const ApplicationFilteringRule = () => {
  return {
    "application-id": "",    // ID of the application
    "endpoint-fqdn": "",     // Endpoint fully qualified domain name
    "endpoint-ip": "",       // Endpoint IP address
    "endpoint-port": 0,      // Endpoint port
    "protocol": "",          // Protocol (TCP, UDP, etc)
    "traffic-class": "",      // Traffic class name for this rule
    "endPort": 0,
    "appMbrUplink": 0,
    "appMbrDownlink": 0,
    "bitrateUnit": "",
    "trafficClass": null,
    "ruleTrigger": ""
  };
};
