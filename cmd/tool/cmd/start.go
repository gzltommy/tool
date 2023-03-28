package cmd

import (
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	svc "github.com/judwhite/go-svc"
	"github.com/robfig/cron/v3"
	"github.com/spf13/cobra"
	"net/http"
	"os"
	"syscall"
	"time"

	"tool-attendance/config"
	"tool-attendance/log"
	"tool-attendance/model"
	"tool-attendance/router"
	"tool-attendance/utils/wrapper"
)

type Application struct {
	wrapper    wrapper.Wrapper
	ginEngine  *gin.Engine
	httpServer *http.Server
	cron       *cron.Cron
}

var cfgFile *string

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "start the api",
	Long: `usage example:
	server(.exe) start -c config.json
	start the api`,
	Run: func(cmd *cobra.Command, args []string) {
		app := &Application{}
		if err := svc.Run(app, syscall.SIGINT, syscall.SIGTERM); err != nil {
			fmt.Println(err)
		}
	},
}

func init() {
	rootCmd.AddCommand(startCmd)
	cfgFile = startCmd.Flags().StringP("config", "c", "", "api config file (required)")
	startCmd.MarkFlagRequired("config")
}

func (app *Application) Init(env svc.Environment) error {
	cfg, err := config.Init(cfgFile)
	if err != nil {
		return err
	}
	err = log.Init(&cfg.Logger)
	if err != nil {
		return err
	}
	// mysql
	if err = model.Init(&cfg.Mysql); err != nil {
		return err
	}

	// http
	app.ginEngine = router.InitAiRouter(&cfg)
	return nil
}

func (app *Application) Start() error {
	fmt.Println("start begin")
	app.wrapper.Wrap(func() {
		cfg := config.GetConfig().Server
		app.httpServer = &http.Server{
			Handler:        app.ginEngine,
			Addr:           cfg.ListenAddr,
			ReadTimeout:    cfg.ReadTimeout * time.Second,
			WriteTimeout:   cfg.WriteTimeout * time.Second,
			IdleTimeout:    cfg.IdleTimeout * time.Second,
			MaxHeaderBytes: cfg.MaxHeaderBytes,
		}
		log.Log.Info("Listen on->", cfg.ListenAddr)

		if err := app.httpServer.ListenAndServe(); err != nil {
			fmt.Println(err)
		}
	})
	fmt.Println("start end")

	// 服务内存和cpu使用监控
	go func() {
		ip := fmt.Sprintf("localhost:%d", 28001)
		if err := http.ListenAndServe(ip, nil); err != nil {
			fmt.Printf("start pprof failed on %s\n", ip)
			os.Exit(1)
		}
	}()

	return nil
}

func (app *Application) Stop() error {
	fmt.Println("done begin")
	if app.httpServer != nil {
		if err := app.httpServer.Shutdown(context.Background()); err != nil {
			fmt.Printf("http shutdown error:%v\n", err)
		}
		fmt.Println("http shutdown")
	}
	app.wrapper.Wait()
	fmt.Println("done end")
	return nil
}
