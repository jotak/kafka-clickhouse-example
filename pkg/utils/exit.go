package utils

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/sirupsen/logrus"
)

var (
	exitChannel chan struct{}
)

func ExitChannel() <-chan struct{} {
	return exitChannel
}

func IsStopped() bool {
	select {
	case <-exitChannel:
		return true
	default:
		return false
	}
}

func SetupElegantExit() {
	exitChannel = make(chan struct{})
	exitSigChan := make(chan os.Signal, 1)
	signal.Notify(exitSigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-exitSigChan
		logrus.Debugf("received exit signal = %v", sig)
		close(exitChannel)
	}()
}
