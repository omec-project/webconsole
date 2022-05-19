module github.com/omec-project/webconsole

go 1.14

replace github.com/omec-project/webconsole => ../webconsole

replace github.com/omec-project/webconsole/configmodels => ../webconsole/configmodels

replace github.com/omec-project/webconsole/configapi => ../webconsole/configapi

require (
	github.com/antonfisher/nested-logrus-formatter v1.3.1
	github.com/gin-contrib/cors v1.3.1
	github.com/mitchellh/mapstructure v1.4.1
	github.com/omec-project/MongoDBLibrary v1.0.100-dev
	github.com/omec-project/config5g v1.0.100-dev
	github.com/omec-project/logger_conf v1.0.100-dev
	github.com/omec-project/logger_util v1.0.100-dev
	github.com/omec-project/openapi v1.0.100-dev
	github.com/omec-project/path_util v1.0.100-dev
	github.com/omec-project/webconsole/configapi v0.0.0-00010101000000-000000000000
	github.com/omec-project/webconsole/configmodels v0.0.0-00010101000000-000000000000
	github.com/sirupsen/logrus v1.8.1
	github.com/urfave/cli v1.22.5
	go.mongodb.org/mongo-driver v1.7.3
	google.golang.org/grpc v1.39.0
	google.golang.org/protobuf v1.27.1
	gopkg.in/yaml.v2 v2.4.0
)
