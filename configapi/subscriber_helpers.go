package configapi

import (
	"context"

	"github.com/omec-project/openapi/models"
	"github.com/omec-project/webconsole/backend/logger"
	"github.com/omec-project/webconsole/configmodels"
	"github.com/omec-project/webconsole/dbadapter"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

func getDeletedImsisList(group, prevGroup *configmodels.DeviceGroups) (dimsis []string) {
	if prevGroup == nil {
		return
	}

	if group == nil {
		return prevGroup.Imsis
	}

	for _, pimsi := range prevGroup.Imsis {
		var found bool
		for _, imsi := range group.Imsis {
			if pimsi == imsi {
				found = true
			}
		}

		if !found {
			dimsis = append(dimsis, pimsi)
		}
	}

	return
}

func removeSubscriberEntriesRelatedToDeviceGroups(mcc, mnc, imsi string, sessionRunner dbadapter.SessionRunner) error {
	filterImsiOnly := bson.M{"ueId": "imsi-" + imsi}
	filter := bson.M{"ueId": "imsi-" + imsi, "servingPlmnId": mcc + mnc}

	ctx := context.TODO()
	err := sessionRunner(ctx, func(sc mongo.SessionContext) error {
		// AM policy
		err := dbadapter.CommonDBClient.RestfulAPIDeleteOneWithContext(ctx, amPolicyDataColl, filterImsiOnly)
		if err != nil {
			logger.DbLog.Errorf("failed to delete AM policy data for IMSI %s: %v", imsi, err)
			return err
		}
		// SM policy
		err = dbadapter.CommonDBClient.RestfulAPIDeleteOneWithContext(ctx, smPolicyDataColl, filterImsiOnly)
		if err != nil {
			logger.DbLog.Errorf("failed to delete SM policy data for IMSI %s: %v", imsi, err)
			return err
		}
		// AM data
		err = dbadapter.CommonDBClient.RestfulAPIDeleteOneWithContext(ctx, amDataColl, filter)
		if err != nil {
			logger.DbLog.Errorf("failed to delete AM data for IMSI %s: %v", imsi, err)
			return err
		}
		// SM data
		err = dbadapter.CommonDBClient.RestfulAPIDeleteOneWithContext(ctx, smDataColl, filter)
		if err != nil {
			logger.DbLog.Errorf("failed to delete SM data for IMSI %s: %v", imsi, err)
			return err
		}
		// SMF selection
		err = dbadapter.CommonDBClient.RestfulAPIDeleteOneWithContext(ctx, smfSelDataColl, filter)
		if err != nil {
			logger.DbLog.Errorf("failed to delete SMF selection data for IMSI %s: %v", imsi, err)
			return err
		}
		return nil
	})
	if err != nil {
		logger.DbLog.Errorf("failed to delete subscriber entries related to device groups for IMSI %s: %v", imsi, err)
		return err
	}
	logger.DbLog.Debugf("succeeded to delete subscriber entries related to device groups for IMSI %s", imsi)
	return nil
}

func handleSubscriberDelete(imsi string) error {
	rwLock.Lock()
	defer rwLock.Unlock()
	err := subscriberAuthData.SubscriberAuthenticationDataDelete(imsi)
	if err != nil {
		logger.DbLog.Errorln("SubscriberAuthDataDelete error:", err)
		return err
	}
	logger.DbLog.Debugf("successfully processed subscriber delete for IMSI: %s", imsi)
	return nil
}

func handleSubscriberPut(imsi string, authSubData *models.AuthenticationSubscription) error {
	rwLock.Lock()
	defer rwLock.Unlock()
	err := subscriberAuthData.SubscriberAuthenticationDataUpdate(imsi, authSubData)
	if err != nil {
		logger.DbLog.Errorln("Subscriber Authentication Data Update Error:", err)
		return err
	}
	logger.DbLog.Debugf("successfully processed subscriber update for IMSI: %s", imsi)
	return nil
}

func handleSubscriberPost(imsi string, authSubData *models.AuthenticationSubscription) error {
	rwLock.Lock()
	defer rwLock.Unlock()
	err := subscriberAuthData.SubscriberAuthenticationDataCreate(imsi, authSubData)
	if err != nil {
		logger.DbLog.Errorln("Subscriber Authentication Data Create Error:", err)
		return err
	}
	logger.DbLog.Debugf("successfully processed subscriber post for IMSI: %s", imsi)
	return nil
}
