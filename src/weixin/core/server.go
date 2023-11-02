package core

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/jonnywang/redcon2"
	"github.com/tidwall/redcon"
	"net/http"
	"os"
	"syscall"
	"time"
	"weixin/common"
)

var PID int
var runAtTime time.Time

func init() {
	PID = syscall.Getpid()
	runAtTime = time.Now()
}

func RunInit() {
	wx.LoadData()
}

func ExitServer() {
	p, err := os.FindProcess(PID)
	if err == nil {
		p.Signal(syscall.SIGTERM)
	}
}

func RunRedisServer(ctx *common.ServerContext) {
	defer ctx.Done()
	ctx.Add()

	rs := redcon2.NewRedconServeMux()
	rs.Handle("version", func(conn redcon.Conn, cmd redcon.Command) {
		conn.WriteBulkString(common.VERSION)
	})
	rs.Handle("command", func(conn redcon.Conn, cmd redcon.Command) {
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

		wxValue, err := GetToken(string(cmd.Args[1]), cacheFirst)
		if err != nil {
			conn.WriteBulkString("")
			return
		}
		conn.WriteBulkString(wxValue.value)
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

		wxValue, err := GetTicket(string(cmd.Args[1]), cacheFirst)
		if err != nil {
			conn.WriteBulkString("")
			return
		}
		conn.WriteBulkString(wxValue.value)
	})
	rs.Handle("ztoken", func(conn redcon.Conn, cmd redcon.Command) {
		if len(cmd.Args) < 2 {
			conn.WriteError("ERR command args with ztoken")
			return
		}

		cacheFirst := true
		if len(cmd.Args) >= 3 && string(cmd.Args[2]) == "1" {
			cacheFirst = false
		}

		conn.WriteArray(2)

		wxValue, err := GetToken(string(cmd.Args[1]), cacheFirst)
		if err == nil {
			conn.WriteBulkString(wxValue.value)
			conn.WriteBulkString(fmt.Sprintf("%d", wxValue.expireAt.Unix()))
		} else {
			common.Logger.Print(err)
			conn.WriteBulkString("")
			conn.WriteBulkString("0")
		}
	})
	rs.Handle("zticket", func(conn redcon.Conn, cmd redcon.Command) {
		if len(cmd.Args) < 2 {
			conn.WriteError("ERR command args with zticket")
			return
		}

		cacheFirst := true
		if len(cmd.Args) >= 3 && string(cmd.Args[2]) == "1" {
			cacheFirst = false
		}

		conn.WriteArray(2)

		wxValue, err := GetTicket(string(cmd.Args[1]), cacheFirst)
		if err == nil {
			conn.WriteBulkString(wxValue.value)
			conn.WriteBulkString(fmt.Sprintf("%d", wxValue.expireAt.Unix()))
		} else {
			common.Logger.Print(err)
			conn.WriteBulkString("")
			conn.WriteBulkString("0")
		}
	})
	rs.Handle("zall", func(conn redcon.Conn, cmd redcon.Command) {
		if len(cmd.Args) < 2 {
			conn.WriteError("ERR command args with zall")
			return
		}

		conn.WriteArray(4)

		wxValue, err := GetToken(string(cmd.Args[1]), false)
		if err == nil {
			conn.WriteBulkString(wxValue.value)
			conn.WriteBulkString(fmt.Sprintf("%d", wxValue.expireAt.Unix()))
		} else {
			common.Logger.Print(err)
			conn.WriteBulkString("")
			conn.WriteBulkString("0")
		}

		wxValue, err = GetTicket(string(cmd.Args[1]), false)
		if err == nil {
			conn.WriteBulkString(wxValue.value)
			conn.WriteBulkString(fmt.Sprintf("%d", wxValue.expireAt.Unix()))
		} else {
			common.Logger.Print(err)
			conn.WriteBulkString("")
			conn.WriteBulkString("0")
		}
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
			rs = nil
			ExitServer()
		}
	}()

	select {
	case <-ctx.Quit():
		common.Logger.Print("redis server catch exit signal")
		if rs != nil {
			rs.Close()
		}
	}
}

func RunWebServer(ctx *common.ServerContext) {
	defer ctx.Done()
	ctx.Add()

	gin.SetMode(gin.ReleaseMode)
	router := gin.Default()
	router.GET("/", func(c *gin.Context) {
		c.String(http.StatusOK, "version "+common.VERSION)
	})
	router.GET("/token/:name/*flag", func(c *gin.Context) {
		name := c.Param("name")
		flag := c.Param("flag")

		wxValue, err := GetToken(name, flag != "/1")
		if err == nil {
			c.String(http.StatusOK, wxValue.value)
		} else {
			common.Logger.Print(err)
			c.String(http.StatusOK, "")
		}
	})
	router.GET("/ticket/:name/*flag", func(c *gin.Context) {
		name := c.Param("name")
		flag := c.Param("flag")

		wxValue, err := GetTicket(name, flag != "/1")
		if err == nil {
			c.String(http.StatusOK, wxValue.value)
		} else {
			common.Logger.Print(err)
			c.String(http.StatusOK, "")
		}
	})
	router.GET("/ztoken/:name/*flag", func(c *gin.Context) {
		name := c.Param("name")
		flag := c.Param("flag")

		wxValue, err := GetToken(name, flag != "/1")
		if err == nil {
			c.JSON(http.StatusOK, gin.H{"value": wxValue.value, "expireAt": wxValue.expireAt.Unix()})
		} else {
			common.Logger.Print(err)
			c.JSON(http.StatusOK, gin.H{"value": "", "expireAt": 0})
		}
	})
	router.GET("/zticket/:name/*flag", func(c *gin.Context) {
		name := c.Param("name")
		flag := c.Param("flag")

		wxValue, err := GetTicket(name, flag != "/1")
		if err == nil {
			c.JSON(http.StatusOK, gin.H{"value": wxValue.value, "expireAt": wxValue.expireAt.Unix()})
		} else {
			common.Logger.Print(err)
			c.JSON(http.StatusOK, gin.H{"value": "", "expireAt": 0})
		}
	})
	router.GET("/zall/:name", func(c *gin.Context) {
		name := c.Param("name")

		var result = gin.H{
			"token":  gin.H{"value": "", "expireAt": 0},
			"ticket": gin.H{"value": "", "expireAt": 0},
		}

		wxValue, err := GetToken(name, false)
		if err == nil {
			result["token"] = gin.H{"value": wxValue.value, "expireAt": wxValue.expireAt.Unix()}
		} else {
			common.Logger.Print(err)
		}

		wxValue, err = GetTicket(name, false)
		if err == nil {
			result["ticket"] = gin.H{"value": wxValue.value, "expireAt": wxValue.expireAt.Unix()}
		} else {
			common.Logger.Print(err)
		}

		c.JSON(http.StatusOK, result)
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
	case <-ctx.Quit():
		common.Logger.Print("redis server catch exit signal")
		server.Shutdown(ctx.Context())
	}
}

func Run() error {
	ctx := common.NewServerContext()

	ctx.Set("startTime", runAtTime)

	go RunInit()
	go RunRedisServer(ctx)
	go RunWebServer(ctx)

	select {
	case <-ctx.Interrupt():
		common.Logger.Print("server interrupt")
		ctx.Cancel()
	}

	ctx.Wait()
	common.Logger.Printf("server uptime %v %v", runAtTime.Format("2006-01-02 15:04:05"), time.Now().Sub(runAtTime))
	common.Logger.Print("server exit")

	return nil
}
