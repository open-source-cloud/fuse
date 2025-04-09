package cli

import (
	"os"
	"os/signal"
	"syscall"
)

// waitForEsc blocks the calling process until the ESC key is pressed.
func quitOnCtrlC() {
	// Create a channel to receive signals
	sigs := make(chan os.Signal, 1)
	// Register for SIGINT (Ctrl+C)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	// Block until a signal is received
	<-sigs
}
