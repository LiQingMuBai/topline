package main

import (
	"ushield_bot/internal/app"

	"github.com/sirupsen/logrus"
)

func main() {
	application, err := app.New()
	if err != nil {
		logrus.WithError(err).Fatal("应用初始化失败")
	}
	defer func() {
		if closeErr := application.Close(); closeErr != nil {
			logrus.WithError(closeErr).Warn("应用关闭时释放资源失败")
		}
	}()

	if err := application.Run(); err != nil {
		logrus.WithError(err).Fatal("应用运行失败")
	}
}
