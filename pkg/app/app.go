package app

import (
	"os"

	"github.com/BurntSushi/toml"
	"github.com/bicosteve/booking-system/pkg/entities"
)

func LoadConfigs(file string) (entities.Config, error) {
	var config entities.Config

	data, err := os.ReadFile(file)
	if err != nil {
		entities.MessageLogs.ErrorLog.Fatalf("could not read toml file due to %v ", err)

	}

	_, err = toml.Decode(string(data), &config)
	if err != nil {
		entities.MessageLogs.ErrorLog.Fatalf("could not load configs due to %v ", err)

	}

	return config, nil
}

// func StartApp(entities.Config) error {
// 	return nil
// }

// import (
// 	"context"
// 	"fmt"
// 	"log/slog"
// 	"os"
// 	"os/signal"
// 	"syscall"
// 	"time"

// 	"github.com/bicosteve/booking-system/pkg/config"
// )

// type done struct {
// 	AppId  string
// 	Cancel context.CancelFunc
// }

// // a callback function provided to user to destroyable components created
// // on start of the application if any.
// type HookFunc func()

// // a callback function provided to users to create or initialize
// // components on start of the program. The callback function
// // exposes a global cancellable context and configuration and returns
// // an error if  there's failure during execution of the callback function
// type InitFunc func(ctx context.Context, config config.Config) error

// var doneChan = make(chan done, 1)

// // allows users to create objects to be initialized in the main thread
// // provides global context and configuration for use and reads users
// // configurations provided

// func Run(cfg config.Config, runFunc InitFunc) error {
// 	err := newLogger(cfg.Logger)
// 	if err != nil {
// 		return err
// 	}

// 	if !cfg.App.Enable {
// 		slog.Error(fmt.Errorf("%v disabled", cfg.App.Id).Error())
// 	}

// 	slog.Info(fmt.Sprintf("starting %v ...", cfg.App.Developer))
// 	for _, name := range cfg.App.Developer {
// 		slog.Info(fmt.Sprintf("author: %v", name))
// 	}

// 	// crate a new cancel context to be used as the global context
// 	ctx, cancel := context.WithCancel(context.Background())
// 	doneChan <- done{AppId: cfg.App.Id, Cancel: cancel}

// 	err = runFunc(ctx, cfg)
// 	if err != nil {
// 		slog.Error(fmt.Errorf("starting %v failed: %v", cfg.App.Id).Error())
// 		return fmt.Errorf("starting %v failed because of %v", cfg.App.Id, err)

// 	}
// 	return nil
// }

// // allows users to create objects to be initialized in the main thread
// // and provides global context and configuration for use and reads user configurations
// // file given in the filename.
// func Start(configFile string, startFunc InitFunc) error {
// 	cfg, err := config.ParseConfig(configFile)
// 	if err != nil {
// 		return err
// 	}

// 	return Run(cfg, startFunc)
// }

// // holds the main thread untill an interrupt signal is received
// func Join() error {
// 	return Wait(nil)
// }

// func Wait(hookFunc HookFunc) error {
// 	interrupt := make(chan os.Signal, 1)
// 	signal.Notify(interrupt, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGTERM)
// 	done := <-doneChan
// 	slog.Info(fmt.Sprintf("%v started", done.AppId))
// 	<-interrupt
// 	slog.Info(fmt.Sprintf("%v shutting down %v ...", done.AppId))
// 	done.Cancel()
// 	if hookFunc != nil {
// 		if err := util.NewTimedTask(time.Second, func(t util.Task) {
// 			defer t.Done()
// 			hookFunc()
// 		}); err != nil {
// 			return err
// 		}
// 	}

// 	slog.Info(fmt.Sprintf("%v shutdown", done.AppId))
// }
