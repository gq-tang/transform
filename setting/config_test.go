package setting

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func Test_LoadConfig(t *testing.T) {
	Convey("test load config", t, func() {
		LoadConfig("../config/app.toml")
		So(Cfg.General.Port, ShouldEqual, 9600)
		So(Cfg.General.LogLevel, ShouldEqual, 4)
		So(Cfg.TcpServer.KeepAlive, ShouldEqual, true)
		So(Cfg.TcpServer.KeepAlivePeriod, ShouldEqual, 30)
		So(Cfg.TcpServer.CloseClientPeriod, ShouldEqual, 300)
	})
}
