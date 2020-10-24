package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
)

// Config 配置
type Config struct {
	Listen  uint16
	Forward []string
}

// DoServer 启动服务器
func DoServer(config []Config) {
	var handle = func(cfg Config) {
		// 负载均衡封装
		var getforward func() string
		var fid = -1
		if len(cfg.Forward) > 1 {
			getforward = func() string {
				fid++
				if fid >= len(cfg.Forward) {
					fid = 0
				}
				return cfg.Forward[fid]
			}
		} else {
			getforward = func() string {
				return cfg.Forward[0]
			}
		}
		var doconn = func(conn net.Conn) {
			// 处理进来的连接
			defer conn.Close()
			forw := getforward()
			log.Println(forw)
			fconn, err := net.Dial("tcp", forw)
			if err != nil {
				log.Println(err)
				return
			}
			defer fconn.Close()
			go io.Copy(conn, fconn)
			io.Copy(fconn, conn)
		}
		// 处理
		lis, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%v", cfg.Listen))
		if err != nil {
			panic(err)
		}
		defer lis.Close()
		log.Println("Listen on", cfg.Listen)
		for {
			conn, err := lis.Accept()
			if err != nil {
				continue
			}
			go doconn(conn)
		}
	}
	for _, cfg := range config {
		go handle(cfg)
	}
}

func main() {
	cfg := flag.String("f", "config.json", "Config file")
	flag.Parse()
	psignal := make(chan os.Signal, 1)
	// ctrl+c->SIGINT, kill -9 -> SIGKILL
	signal.Notify(psignal, syscall.SIGINT, syscall.SIGKILL)
	configBytes, err := ioutil.ReadFile(*cfg)
	if err != nil {
		panic(err)
	}
	var config []Config
	err = json.Unmarshal(configBytes, &config)
	if err != nil {
		panic(err)
	}
	go DoServer(config)
	<-psignal
	log.Println("Bye~")
}
