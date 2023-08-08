package cache

import (
	"fmt"
	"os"

	"git.code.oa.com/trpc-go/trpc-go/log"
	"git.code.oa.com/video_pay_root/pay-go-comm/ratta"
)

var metricAtta ratta.ReliableAtta

// metricData 统计数据信息
type metricData struct {
	ServiceName string // 服务名
	Engine      string // cache引擎名
	Env         string // 环境名
}

func init() {
	var err error
	if metricAtta, err = ratta.NewReliableAtta("08b00074041", "2345548394"); err != nil {
		log.Errorf("Atta init err: %+v", err)
	}
}

func report(engine string) {
	v := &metricData{
		ServiceName: getServiceName(),
		Engine:      engine,
		Env:         getEnv(),
	}
	if metricAtta == nil {
		log.Errorf("Atta report nil client")
		return
	}
	if err := metricAtta.SendByStruct(v); err != nil {
		log.Errorf("Atta report err: %+v, v: %+v", err, v)
	}
}

func getServiceName() string {
	return fmt.Sprintf("%v.%v", os.Getenv("SUMERU_APP"), os.Getenv("SUMERU_SERVER"))
}

func getEnv() string {
	return os.Getenv("DOCKER_ENV")
}
