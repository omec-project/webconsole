module github.com/free5gc/webconsole

go 1.14

require (
	github.com/antonfisher/nested-logrus-formatter v1.3.0
	github.com/free5gc/MongoDBLibrary v1.0.0
	github.com/free5gc/logger_conf v1.0.0
	github.com/free5gc/logger_util v1.0.0
	github.com/free5gc/openapi v1.0.0
	github.com/free5gc/path_util v1.0.0
	github.com/free5gc/version v1.0.0
	github.com/gin-contrib/cors v1.3.1
	github.com/gin-gonic/gin v1.6.3
	github.com/mitchellh/mapstructure v1.4.0
	github.com/omec-project/webconsole v0.0.0-00010101000000-000000000000
	github.com/sirupsen/logrus v1.7.0
	github.com/urfave/cli v1.22.5
	go.mongodb.org/mongo-driver v1.4.4
	google.golang.org/grpc v1.31.0
	google.golang.org/protobuf v1.25.0
	gopkg.in/yaml.v2 v2.4.0
)

replace github.com/omec-project/webconsole => ../webconsole
