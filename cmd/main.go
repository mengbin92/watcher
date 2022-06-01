package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"

	"github.com/fsnotify/fsnotify"
	"github.com/mengbin92/watcher/logger"
	"go.uber.org/zap"
)

var (
	h   bool
	d   bool
	s   bool
	log *zap.SugaredLogger
)

func init() {
	log = logger.DefaultLogger().Sugar()
	flag.BoolVar(&h, "h", false, "usage")
	flag.BoolVar(&d, "d", false, "run watcher as a daemon with -d=true")
	flag.BoolVar(&s, "s", false, "shutdown watcher")
	flag.Usage = usage
	flag.Parse()

	if d {
		cmd := exec.Command(os.Args[0], flag.Args()...)
		if err := cmd.Start(); err != nil {
			fmt.Printf("start %s failed with error: %s", os.Args[0], err.Error())
			os.Exit(-1)
		}
		f, _ := os.OpenFile("watcher.lock", os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
		fmt.Fprintf(f, "%d", cmd.Process.Pid)
		log.Info(fmt.Sprintf("%s [PID] %d running...\n", os.Args[0], cmd.Process.Pid))
		f.Close()
		os.Exit(0)
	}
	if h {
		flag.Usage()
	}
	if s {
		data, err := ioutil.ReadFile("watcher.lock")
		if err != nil {
			log.Fatal(err.Error())
		}
		cmd := exec.Command("kill", "-9", string(data))
		if err := cmd.Start(); err != nil {
			log.Error(fmt.Sprintf("shutdown watcher error: %s", err.Error()))
			os.Exit(-1)
		}
		log.Info("watcher is down")
		os.Exit(0)
	}
}

func usage() {
	fmt.Fprintf(os.Stdout, `watcher usage:
Options:
`)
	flag.PrintDefaults()
	os.Exit(0)
}

func main() {

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal("NewWatcher failed: ", err)
	}
	defer watcher.Close()

	done := make(chan bool)
	go func() {
		defer close(done)

		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				log.Infof("%s %s\n", event.Name, event.Op)
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Infof("error: %s", err)
			}
		}
	}()

	err = watcher.Add("./")
	if err != nil {
		log.Fatal("Add failed:", err)
	}
	<-done
}
