package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"path"

	pb "github.com/pion/ion-avp/cmd/signal/grpc/proto"
	"github.com/pion/ion-avp/cmd/signal/grpc/server"
	avp "github.com/pion/ion-avp/pkg"
	"github.com/pion/ion-avp/pkg/elements"
	log "github.com/pion/ion-log"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
)

type webmsaver struct {
	Path string `mapstructure:"path"`
}

// Config for server
type Config struct {
	Webmsaver webmsaver  `mapstructure:"webmsaver"`
	Avp       avp.Config `mapstructure:"avp"`
}

var (
	conf = Config{}
	file string
	addr string
)

func createWebmSaver(sid, pid, tid string, config []byte) avp.Element {
	filewriter := elements.NewFileWriter(
		path.Join(conf.Webmsaver.Path, fmt.Sprintf("%s-%s.webm", sid, pid)),
		4096,
	)
	webm := elements.NewWebmSaver()
	webm.Attach(filewriter)
	return webm
}

func showHelp() {
	fmt.Printf("Usage:%s {params}\n", os.Args[0])
	fmt.Println("      -c {config file}")
	fmt.Println("      -a {listen addr}")
	fmt.Println("      -h (show help info)")
}

func load() bool {
	_, err := os.Stat(file)
	if err != nil {
		return false
	}

	viper.SetConfigFile(file)
	viper.SetConfigType("toml")

	err = viper.ReadInConfig()
	if err != nil {
		fmt.Printf("config file %s read failed. %v\n", file, err)
		return false
	}
	err = viper.GetViper().Unmarshal(&conf)
	if err != nil {
		fmt.Printf("config file %s loaded failed. %v\n", file, err)
		return false
	}

	fmt.Printf("config %s load ok!\n", file)
	return true
}

func parse() bool {
	flag.StringVar(&file, "c", "config.toml", "config file")
	flag.StringVar(&addr, "a", ":50052", "address to use")
	help := flag.Bool("h", false, "help info")
	flag.Parse()
	if !load() {
		return false
	}

	if *help {
		showHelp()
		return false
	}
	return true
}

func main() {
	if !parse() {
		showHelp()
		os.Exit(-1)
	}

	log.Init(conf.Avp.Log.Level)

	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Panicf("failed to listen: %v", err)
	}
	log.Infof("--- AVP Node Listening at %s ---", addr)

	s := grpc.NewServer()
	srv := server.NewAVPServer(conf.Avp, map[string]avp.ElementFun{
		"webmsaver": createWebmSaver,
	})
	pb.RegisterAVPServer(s, srv)

	if err := s.Serve(lis); err != nil {
		log.Panicf("failed to serve: %v", err)
	}

	log.Infof("server finished")
}
