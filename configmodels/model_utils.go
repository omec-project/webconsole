// SPDX-FileCopyrightText: 2024 Open Networking Foundation <info@opennetworking.org>
// SPDX-FileCopyrightText: 2024 Canonical Ltd
//
// SPDX-License-Identifier: Apache-2.0

package configmodels

import (
	"encoding/json"

	"github.com/omec-project/webconsole/backend/logger"
	"go.mongodb.org/mongo-driver/bson"
)

func ToBsonM(data any) (ret bson.M) {
	tmp, err := json.Marshal(data)
	if err != nil {
		logger.DbLog.Errorln("could not marshal data")
		return nil
	}
	err = json.Unmarshal(tmp, &ret)
	if err != nil {
		logger.DbLog.Errorln("could not unmarshal data")
		return nil
	}
	return ret
}

func MapToByte(data map[string]any) (ret []byte) {
	ret, err := json.Marshal(data)
	if err != nil {
		logger.DbLog.Errorln("could not marshal data")
		return nil
	}
	return ret
}
