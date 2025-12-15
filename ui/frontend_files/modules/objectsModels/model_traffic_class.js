/**
 * JavaScript equivalent of Go structs from model_traffic_class.go
 */

export const TrafficClassInfo = () => {
  return {
    name: "",       // Traffic class name
    qci: 0,         // QCI/5QI/QFI
    arp: 0,         // Traffic class priority
    pdb: 0,         // Packet Delay Budget
    pelr: 0,        // Packet Error Loss Rate
  };
};
