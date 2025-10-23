package di

import "github.com/open-source-cloud/fuse/pkg/strutil"

const (
	// mongoDriver is the name of the MongoDB driver
	mongoDriver = "mongodb"
)

// IsDriverEnabled checks if the driver is enabled
func IsDriverEnabled(cfgDriver string, targetDriver string) bool {
	return strutil.SerializeString(cfgDriver) == strutil.SerializeString(targetDriver)
}
