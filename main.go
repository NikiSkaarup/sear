package main

import (
	"os"

	"github.com/nikiskaarup/sear/cmd"
	"github.com/sirupsen/logrus"
)

func main() {
	if err := cmd.Execute(); err != nil {
		logrus.Errorf("Fatal error: %v", err)
		os.Exit(1)
	}
}
