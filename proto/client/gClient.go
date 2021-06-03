package client

import (
	context "context"
	protos "github.com/badhrinathpa/webconsole/proto/omec5gconfig"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
	"log"
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

type ConfigReq struct {
	suppNssaiPlmnList SupportedNssaiInPlmnList
	SuppPlmnList      SupportedPlmnList
}

type ConfigResp struct {
	suppNssaiPlmnList SupportedNssaiInPlmnList
	SuppPlmnList      SupportedPlmnList
}

type ConfigClient struct {
	Client  protos.ConfigServiceClient
	Conn    *grpc.ClientConn
	Version uint32
}

func CreateChannel(host string, timeout uint32) (*ConfigClient, error) {
	log.Println("create config client")
	// Second, check to see if we can reuse the gRPC connection for a new P4RT client
	conn, err := GetConnection(host)
	if err != nil {
		log.Println("grpc connection failed")
		return nil, err
	}

	client := &ConfigClient{
		Client: protos.NewConfigServiceClient(conn),
		Conn:   conn,
	}

	return client, nil
}

var kacp = keepalive.ClientParameters{
	Time:                10 * time.Second, // send pings every 10 seconds if there is no activity
	Timeout:             time.Second,      // wait 1 second for ping ack before considering the connection dead
	PermitWithoutStream: true,             // send pings even without active streams
}

var retryPolicy = `{
		"methodConfig": [{
		  "name": [{"service": "grpc.Config"}],
		  "waitForReady": true,
		  "retryPolicy": {
			  "MaxAttempts": 4,
			  "InitialBackoff": ".01s",
			  "MaxBackoff": ".01s",
			  "BackoffMultiplier": 1.0,
			  "RetryableStatusCodes": [ "UNAVAILABLE" ]
		  }}]}`

func GetConnection(host string) (conn *grpc.ClientConn, err error) {
	/* get connection */
	log.Println("Get connection.")
	conn, err = grpc.Dial(host, grpc.WithInsecure(), grpc.WithKeepaliveParams(kacp), grpc.WithDefaultServiceConfig(retryPolicy))
	if err != nil {
		log.Println("grpc dial err: ", err)
		return nil, err
	}
	//defer conn.Close()
	return conn, err
}

func (c *ConfigClient) WriteConfig(cReq *ConfigReq) error {
	log.Println("Write config request")
	wreq := &protos.WriteRequest{}
	wcfg := &protos.Config{}
	wreq.WriteConfig = wcfg
	suppPlmnList := &protos.SupportedPlmnList{}
	for _, pl := range cReq.SuppPlmnList.PlmnIdList {
		log.Println("mcc: ", pl.MCC)
		log.Println("mnc: ", pl.MNC)
		plmnId := &protos.PlmnId{
			Mcc: pl.MCC,
			Mnc: pl.MNC,
		}
		suppPlmnList.PlmnIds = append(suppPlmnList.PlmnIds, plmnId)
	}
	wcfg.SuppPlmnList = suppPlmnList
	ret, err := c.Client.Write(context.Background(), wreq)
	if ((ret != nil) && (ret.WriteStatus == protos.Status_FAILURE)) || err != nil {
		return err
	}

	return nil
}

func (c *ConfigClient) ReadConfig(cRes *ConfigResp) error {
	log.Println("Read request")
	rreq := &protos.ReadRequest{}
	rsp, err := c.Client.Read(context.Background(), rreq)
	if err != nil {
		return err
	}

	rcfg := rsp.ReadConfig
	suppPlmnList := rcfg.SuppPlmnList
	for _, pl := range suppPlmnList.PlmnIds {
		log.Println("mcc: ", pl.Mcc)
		log.Println("mnc: ", pl.Mnc)
		plmnId := PlmnId{
			MCC: pl.GetMcc(),
			MNC: pl.GetMnc(),
		}
		cRes.SuppPlmnList.PlmnIdList = append(cRes.SuppPlmnList.PlmnIdList, plmnId)
	}
	return nil
}
