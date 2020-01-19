package core

import (
	"context"
	"github.com/gin-gonic/gin"
	"github.com/jonnywang/redcon2"
	"github.com/tidwall/redcon"
	"net/http"
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

func ExitServer() {
	p, err := os.FindProcess(PID)
	if err == nil {
		p.Signal(syscall.SIGTERM)
	}
}

func RunRedisServer(ctx context.Context)  {
	rs := redcon2.NewRedconServeMux()
	rs.Handle("version", func (conn redcon.Conn, cmd redcon.Command) {
		conn.WriteBulkString(common.VERSION)
	})
	rs.Handle("command", func (conn redcon.Conn, cmd redcon.Command) {
		conn.WriteString("OK")
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

	rs.Handle("save", func(conn redcon.Conn, cmd redcon.Command) {
		go SaveAll()
		conn.WriteString("OK")
	})

	go func() {
		common.Logger.Printf("run redis protocol server at %+v with pid=%d", common.Config.RedisAddress, PID)
		err := rs.Run(common.Config.RedisAddress)
		if err != nil {
			common.Logger.Print(err)
			ExitServer()
		}
	}()

	select {
	case <-ctx.Done():
		common.Logger.Print("redis server catch exit signal")
		rs.Close()
		return
	}
}

func RunWebServer(ctx context.Context)  {
	gin.SetMode(gin.ReleaseMode)
	router := gin.Default()
	router.GET("/", func(c *gin.Context) {
		c.String(http.StatusOK, "version " + common.VERSION)
	})
	router.GET("/token/:name/*flag", func(c *gin.Context) {
		name := c.Param("name")
		flag := c.Param("flag")

		token, err := GetToken(name, flag != "1")
		if err != nil {
			common.Logger.Print(err)
			token = ""
		}

		c.String(http.StatusOK, token)
	})
	router.GET("/ticket/:name/*flag", func(c *gin.Context) {
		name := c.Param("name")
		flag := c.Param("flag")

		ticket, err := GetTicket(name, flag != "1")
		if err != nil {
			common.Logger.Print(err)
			ticket = ""
		}

		c.String(http.StatusOK, ticket)
	})

	server := &http.Server{
		Addr:    common.Config.WebAddress,
		Handler: router,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			common.Logger.Print(err)
			ExitServer()
		}
	}()

	select {
	case <-ctx.Done():
		common.Logger.Print("redis server catch exit signal")
		server.Shutdown(ctx)
		return
	}
}

func Run() error {
	quit := make(chan os.Signal)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	ctx, cancel := context.WithCancel(context.Background())

	go RunInit()
	go RunRedisServer(ctx)
	go RunWebServer(ctx)

	select {
	case <-quit:
		common.Logger.Print("Shutdown Server ...")
		cancel()
	}

	<-time.After(5 * time.Second)
	common.Logger.Print("Server exiting")

	return nil
}
