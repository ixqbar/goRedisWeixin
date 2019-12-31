package core

import (
	"context"
	"github.com/jonnywang/redcon2"
	"github.com/tidwall/redcon"
	"os"
	"os/signal"
	"syscall"
	"time"
	"weixin/common"
)

var PID int

func init()  {
	PID = syscall.Getpid()
}

func RunInit()  {
	wx.LoadData()
}

func RunRedisServer(ctx context.Context)  {
	rs := redcon2.NewRedconServeMux()
	rs.Handle("version", func (conn redcon.Conn, cmd redcon.Command) {
		conn.WriteBulkString(common.VERSION)
	})
	rs.Handle("token", func(conn redcon.Conn, cmd redcon.Command) {
		if len(cmd.Args) < 2 {
			conn.WriteError("ERR command args with token")
			return
		}

		cacheFirst := true
		if len(cmd.Args) >= 3 && string(cmd.Args[2]) == "1" {
			cacheFirst = false
		}

		token, err := GetToken(string(cmd.Args[1]), cacheFirst)
		if err != nil {
			conn.WriteBulkString("")
			return
		}
		conn.WriteBulkString(token)
	})
	rs.Handle("ticket", func(conn redcon.Conn, cmd redcon.Command) {
		if len(cmd.Args) < 2 {
			conn.WriteError("ERR command args with token")
			return
		}

		cacheFirst := true
		if len(cmd.Args) >= 3 && string(cmd.Args[2]) == "1" {
			cacheFirst = false
		}

		token, err := GetTicket(string(cmd.Args[1]), cacheFirst)
		if err != nil {
			conn.WriteBulkString("")
			return
		}
		conn.WriteBulkString(token)
	})

	go func() {
		common.Logger.Printf("run redis protocol server at %+v with pid=%d", common.Config.Address, PID)
		err := rs.Run(common.Config.Address)
		if err != nil {
			common.Logger.Print(err)
			//兼容win
			p, err := os.FindProcess(PID)
			if err == nil {
				p.Signal(syscall.SIGTERM)
			}
		}
	}()

	select {
	case <-ctx.Done():
		common.Logger.Print("redis server catch exit signal")
		rs.Close()
		return
	}
}

func Run() error {
	quit := make(chan os.Signal)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	ctx, cancel := context.WithCancel(context.Background())

	go RunInit()
	go RunRedisServer(ctx)

	select {
	case <-quit:
		common.Logger.Print("Shutdown Server ...")
		cancel()
	}

	<-time.After(5 * time.Second)
	common.Logger.Print("Server exiting")

	return nil
}
