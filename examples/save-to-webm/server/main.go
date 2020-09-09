package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path"

	grpc "github.com/pion/ion-avp/cmd/server/grpc"
	avp "github.com/pion/ion-avp/pkg"
	"github.com/pion/ion-avp/pkg/elements"
	"github.com/spf13/viper"
)

var (
	conf = avp.Config{}
	file string
	addr string
)

func createWebmSaver(sid, pid, tid string, config []byte) avp.Element {
	filewriter := elements.NewFileWriter(elements.FileWriterConfig{
		ID:   pid,
		Path: path.Join(conf.Pipeline.WebmSaver.Path, fmt.Sprintf("%s-%s.webm", sid, pid)),
	})
	webm := elements.NewWebmSaver(elements.WebmSaverConfig{
		ID: pid,
	})
	err := webm.Attach(filewriter)
	if err != nil {
		log.Fatalf("error attaching filewriter to webm %s", err)
		return nil
	}
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

	registry := avp.NewRegistry()
	registry.AddElement("webmsaver", createWebmSaver)
	avp.Init(registry)

	grpc.NewServer(addr, conf)
	select {}
}
