package ssm

type SsmSyncMessage struct {
	Action string
	Info   string
}

type SSM interface {
	SyncKeyListen(chan *SsmSyncMessage)
	KeyRotationListen(chan *SsmSyncMessage)
	Login() (string, error)
	HealthCheck()
	InitDefault(ssmSyncMsg chan *SsmSyncMessage) error
}
