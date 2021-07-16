package server

import (
	"github.com/omec-project/webconsole/backend/logger"
	"github.com/omec-project/webconsole/configmodels"
	"github.com/sirupsen/logrus"
)

var configLog *logrus.Entry

func init() {
	configLog = logger.ConfigLog
}

var slicesConfigSnapshot map[string]*configmodels.Slice
var devgroupsConfigSnapshot map[string]*configmodels.DeviceGroups

func init() {
	slicesConfigSnapshot = make(map[string]*configmodels.Slice)
	devgroupsConfigSnapshot = make(map[string]*configmodels.DeviceGroups)
}

func configHandler(configMsgChan chan *configmodels.ConfigMessage) {
	for {
		configLog.Infoln("Waiting for configuration event ")
		select {
		case configMsg := <-configMsgChan:
			if configMsg.MsgMethod == configmodels.Post_op || configMsg.MsgMethod == configmodels.Put_op {
				configLog.Infoln("Received msg from configApi package ", configMsg)
				// update config snapshot
				if configMsg.DevGroup != nil {
					configLog.Infoln("Received msg from configApi package for Device Group ", configMsg.DevGroupName)
					devgroupsConfigSnapshot[configMsg.DevGroupName] = configMsg.DevGroup
				}

				if configMsg.Slice != nil {
					configLog.Infoln("Received msg from configApi package for Slice ", configMsg.SliceName)
					slicesConfigSnapshot[configMsg.SliceName] = configMsg.Slice
				}

				// loop through all clients and send this message to all clients
				if len(clientNFPool) == 0 {
					configLog.Infoln("No client available. No need to send config")
				}
				for _, client := range clientNFPool {
					client.outStandingPushConfig <- configMsg
				}
			} else {
				configLog.Infoln("Received delete msg from configApi package ", configMsg)
				// update config snapshot
				if configMsg.DevGroup == nil {
					configLog.Infoln("Received msg from configApi package to delete Device Group ", configMsg.DevGroupName)
					devgroupsConfigSnapshot[configMsg.DevGroupName] = nil
				}

				if configMsg.Slice == nil {
					configLog.Infoln("Received msg from configApi package to delete Slice ", configMsg.SliceName)
					slicesConfigSnapshot[configMsg.SliceName] = nil
				}

				// loop through all clients and send this message to all clients
				if len(clientNFPool) == 0 {
					configLog.Infoln("No client available. No need to send config")
				}
				for _, client := range clientNFPool {
					client.outStandingPushConfig <- configMsg
				}
			}
		}
	}
}
