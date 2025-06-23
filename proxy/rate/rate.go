package rate

import (
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

var Record = newIPBucket()
var Limit = newLimiter()

var (
	initHeartbeat = sync.Once{}
)

func init() {
	initHeartbeat.Do(func() {
		go func() {
			logrus.Info("[Rate] start Record.Clean routine")
			for {
				time.Sleep(time.Second * 10)
				Record.Clean()
			}
		}()
	})
}
