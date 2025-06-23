package rate

import (
	"time"

	"github.com/sirupsen/logrus"
)

var Record = newIPBucket()
var Limit = newLimiter()

func init() {
	go func() {
		logrus.Info("[Rate] start Record.Clean routine")
		for {
			time.Sleep(time.Second * 10)
			Record.Clean()
		}
	}()
}
