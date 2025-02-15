package integrations

import "cuore/common"

type Integration interface {
	HandleControl(msg common.ControlMessage) error
	HandleSetup(msg common.SetupMessage) error
}
