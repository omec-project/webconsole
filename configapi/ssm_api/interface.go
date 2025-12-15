package ssmapi

import "github.com/omec-project/webconsole/configmodels"

type SSMAPI interface {
	StoreKey(k4Data *configmodels.K4) error
	UpdateKey(k4Data *configmodels.K4) error
	DeleteKey(k4Data *configmodels.K4) error
}
