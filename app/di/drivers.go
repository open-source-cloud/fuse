package di

import "github.com/open-source-cloud/fuse/pkg/utils"

const (
	// mongoDriver is the name of the MongoDB driver
	mongoDriver = "mongodb"
)

// IsDriverEnabled checks if the driver is enabled
func IsDriverEnabled(cfgDriver string, targetDriver string) bool {
	return utils.SerializeString(cfgDriver) == utils.SerializeString(targetDriver)
}
