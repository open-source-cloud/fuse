// FUSE Workflow Engine application server
//
// @title FUSE Workflow Engine API
// @version 1.0
// @description Actor-based workflow engine for end-to-end automations and task pipelines
// @contact.name FUSE Team
// @license.name GitHub Repository
// @license.url https://github.com/open-source-cloud/fuse
// @host localhost:9090
// @BasePath /
package main

import "github.com/open-source-cloud/fuse/internal/app/cli"

// FUSE Workflow Engine application cli entrypoint
func main() {
	cli.Run()
}
