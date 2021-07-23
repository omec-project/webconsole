// SPDX-FileCopyrightText: 2021 Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0
// SPDX-License-Identifier: LicenseRef-ONF-Member-Only-1.0

package server

import (
	context "context"
	"github.com/omec-project/webconsole/backend/logger"
	"github.com/omec-project/webconsole/configmodels"
	protos "github.com/omec-project/webconsole/proto/sdcoreConfig"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
	"net"
	"time"
)

var grpcLog *logrus.Entry

func init() {
	grpcLog = logger.GrpcLog
}

type ServerConfig struct {
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

func StartServer(host string, confServ *ConfigServer,
	configMsgChan chan *configmodels.ConfigMessage, subsChannel chan *SubsUpdMsg) {
	grpcLog.Println("Start grpc config server")

	go configHandler(configMsgChan, subsChannel)

	lis, err := net.Listen("tcp", host)
	if err != nil {
		grpcLog.Fatalf("failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer(grpc.KeepaliveEnforcementPolicy(kaep), grpc.KeepaliveParams(kasp))
	protos.RegisterConfigServiceServer(grpcServer, confServ)
	if err = grpcServer.Serve(lis); err != nil {
		grpcLog.Fatalf("failed to serve: %v", err)
	}
	grpcLog.Infoln("Completed grpc server goroutine")
}

func (c *ConfigServer) GetNetworkSlice(ctx context.Context, rReq *protos.NetworkSliceRequest) (*protos.NetworkSliceResponse, error) {
	grpcLog.Infof("Network Slice config req:Client %v, rst counter %v\n", rReq.ClientId, rReq.RestartCounter)
	client, created := getClient(rReq.ClientId)
	var reqMsg clientReqMsg
	reqMsg.networkSliceReqMsg = rReq
	reqMsg.grpcRspMsg = make(chan *clientRspMsg)
	// Post the message on client handler & wait to get response
	if created == true {
		reqMsg.newClient = true
	}
	client.tempGrpcReq <- &reqMsg
	rResp := <-reqMsg.grpcRspMsg
	client.clientLog.Infoln("Received response message from client FSM")
	return rResp.networkSliceRspMsg, nil
}
