package client

import {
	"log"
	context "context"
	"errors"
	protos "github.com/badhrinathpa/webconsole/proto/omec5gconfig"
}

type PlmnId struct {
	mcc string
	mnc string
}

type SupportedPlmnList struct {
	PlmnIdList []PlmnId
}

type SupportedNssaiList struct {
	NssaiList []Nssai
}

type SupportedNssaiInPlmnList struct {
	Plmn  PlmnId
	SnssaiList SupportedNssaiList
}

type ConfigReq struct {
    suppNssaiPlmnList SupportedNssaiInPlmnList
    suppPlmnList      SupportedPlmnList
}

type ConfigResp struct {
    suppNssaiPlmnList SupportedNssaiInPlmnList
    suppPlmnList      SupportedPlmnList
}

type ConfigClient struct {
    Client protos.ConfigServiceClient
    Conn   *grpc.ClientConn
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
        Client:   p4.NewConfigServiceClient(conn),
        Conn:     conn,
    }

    return client, nil
}

func GetConnection(host string) (conn *grpc.ClientConn, err error) {
    /* get connection */
    log.Println("Get connection.")
    conn, err = grpc.Dial(host, grpc.WithInsecure())
    if err != nil {
        log.Println("grpc dial err: ", err)
        return nil, err
    }
    return
}

func (c *ConfigClient) WriteConfig(cReq *ConfigReq) error {
	log.Println("Write config request")
    wreq := &protos.WriteRequest{}
    wcfg := &protos.Config{}
    wreq.WriteConfig = wcfg
    suppPlmnList := &protos.SupportedPlmnList{}
	for _, pl := range cReq.suppPlmnList.PlmnIdList {
		log.Println("mcc: ", pl.mcc)
		log.Println("mnc: ", pl.mnc)
		plmnId := &protos.PlmnId {
			Mcc: pl.mcc,
			Mnc: pl.mnc
		}
		suppPlmnList.PlmnIds = append(suppPlmnList.PlmnIds, plmnId)
	}
    wcfg.SuppPlmnList = suppPlmnList
	ret, err := c.Client.Write(context.Background(), wreq)
    if ((ret == protos.Status_FAILURE) || err != nil) {
        return err
    }

    return nil
}

func (c *ConfigClient) ReadConfig(cRes *ConfigResp) error {
	log.Println("Read request")
    rreq := &protos.ReadRequest{}
	rsp, err := c.Client.Read(context.Background(), rreq)
    if ((ret == protos.Status_FAILURE) || err != nil) {
        return err
    }

    rcfg := &rsp.ReadConfig
    suppPlmnList := rcfg.SuppPlmnList
	for _, pl := range suppPlmnList.PlmnIds {
		log.Println("mcc: ", pl.Mcc)
		log.Println("mnc: ", pl.Mnc)
		plmnId := &PlmnId {
			mcc: pl.GetMcc(),
			mnc: pl.GetMnc()
		}
		cRes.suppPlmnList.PlmnIdList = append(cRes.suppPlmnList.PlmnIdList, plmnId)
	}
	return nil
}
