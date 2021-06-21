package server

import (
	context "context"
	protos "github.com/omec-project/webconsole/proto/sdcoreConfig"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
	"log"
	"net"
	"time"
)

type PlmnId struct {
	MCC string
	MNC string
}

type SupportedPlmnList struct {
	PlmnIdList []PlmnId
}

type Nssai struct {
	sst string
	sd  string
}

type SupportedNssaiList struct {
	NssaiList []Nssai
}

type SupportedNssaiInPlmnList struct {
	Plmn       PlmnId
	SnssaiList SupportedNssaiList
}

type ServerConfig struct {
	suppNssaiPlmnList SupportedNssaiInPlmnList
	SuppPlmnList      SupportedPlmnList
}

type ConfigServer struct {
	protos.ConfigServiceServer
	serverCfg ServerConfig
	Version   uint32
}

var kaep = keepalive.EnforcementPolicy{
	MinTime:             15 * time.Second, // If a client pings more than once every 5 seconds, terminate the connection
	PermitWithoutStream: true,             // Allow pings even when there are no active streams
}

var kasp = keepalive.ServerParameters{
	Time:    30 * time.Second, // Ping the client if it is idle for 5 seconds to ensure the connection is still active
	Timeout: 5 * time.Second,  // Wait 1 second for the ping ack before assuming the connection is dead
}

func StartServer(host string, confServ *ConfigServer) {
	log.Println("start config server")
	lis, err := net.Listen("tcp", host)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer(grpc.KeepaliveEnforcementPolicy(kaep), grpc.KeepaliveParams(kasp))
	protos.RegisterConfigServiceServer(grpcServer, confServ)
	if err = grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

func (c *ConfigServer) Read(ctx context.Context, rReq *protos.ReadRequest) (*protos.ReadResponse, error) {
	log.Println("Handle Read config request")
	rResp := &protos.ReadResponse{}
	rCfg := &protos.Config{}
	rResp.ReadConfig = rCfg
	suppPlmnList := &protos.SupportedPlmnList{}
	plmnId1 := &protos.PlmnId{
		Mcc: "305",
		Mnc: "11",
	}
	plmnId2 := &protos.PlmnId{
		Mcc: "208",
		Mnc: "93",
	}
	suppPlmnList.PlmnIds = append(suppPlmnList.PlmnIds, plmnId1)
	suppPlmnList.PlmnIds = append(suppPlmnList.PlmnIds, plmnId2)
	rCfg.SuppPlmnList = suppPlmnList
	return rResp, nil
}

func (c *ConfigServer) Write(ctx context.Context, wReq *protos.WriteRequest) (*protos.WriteResponse, error) {
	log.Println("Handle write request")
	wResp := &protos.WriteResponse{}
	wResp.WriteStatus = protos.Status_SUCCESS

	wCfg := wReq.WriteConfig
	suppPlmnList := wCfg.SuppPlmnList
	for _, pl := range suppPlmnList.PlmnIds {
		log.Println("mcc: ", pl.Mcc)
		log.Println("mnc: ", pl.Mnc)
		plmnId := PlmnId{
			MCC: pl.GetMcc(),
			MNC: pl.GetMnc(),
		}
		c.serverCfg.SuppPlmnList.PlmnIdList = append(c.serverCfg.SuppPlmnList.PlmnIdList, plmnId)
	}
	return wResp, nil
}
