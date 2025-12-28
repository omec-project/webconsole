/**
 * JavaScript equivalent of Go structs from model_flow_rule.go
 */

export const FlowRule = () => {
  return {
    precedence: 0,
    action: "",
    srcIp: "",
    dstIp: "",
    srcPort: 0,
    dstPort: 0,
    proto: ""
  };
};