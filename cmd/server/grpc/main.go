package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net"
	_ "net/http/pprof"
	"os"
	"path"

	pb "github.com/pion/ion-avp/cmd/server/grpc/proto"
	avp "github.com/pion/ion-avp/pkg"
	"github.com/pion/ion-avp/pkg/elements"
	"github.com/pion/ion-avp/pkg/log"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
)

var (
	conf = avp.Config{}
	file string
)

type server struct {
	pb.UnimplementedAVPServer
	avp *avp.AVP
}

func getDefaultElements(id string) map[string]avp.Element {
	de := make(map[string]avp.Element)
	if conf.Pipeline.WebmSaver.Enabled && conf.Pipeline.WebmSaver.DefaultOn {
		filewriter := elements.NewFileWriter(elements.FileWriterConfig{
			ID:   id,
			Path: path.Join(conf.Pipeline.WebmSaver.Path, fmt.Sprintf("%s.webm", id)),
		})
		webm := elements.NewWebmSaver(elements.WebmSaverConfig{
			ID: id,
		})
		err := webm.Attach(filewriter)
		if err != nil {
			log.Errorf("error attaching filewriter to webm %s", err)
		} else {
			de[elements.TypeWebmSaver] = webm
		}
	}
	return de
}

func getTogglableElement(e *pb.Element) (avp.Element, error) {
	switch e.Type {
	case elements.TypeWebmSaver:
		filewriter := elements.NewFileWriter(elements.FileWriterConfig{
			ID:   e.Mid,
			Path: path.Join(conf.Pipeline.WebmSaver.Path, fmt.Sprintf("%s.webm", e.Mid)),
		})
		webm := elements.NewWebmSaver(elements.WebmSaverConfig{
			ID: e.Mid,
		})
		err := webm.Attach(filewriter)
		if err != nil {
			log.Errorf("error attaching filewriter to webm %s", err)
			return nil, err
		}
		return webm, nil
	}

	return nil, errors.New("element not found")
}

func showHelp() {
	fmt.Printf("Usage:%s {params}\n", os.Args[0])
	fmt.Println("      -c {config file}")
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

	lis, err := net.Listen("tcp", conf.GRPC.Port)
	if err != nil {
		log.Panicf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	pb.RegisterAVPServer(s, &server{
		avp: avp.NewAVP(conf, getDefaultElements, getTogglableElement),
	})

	log.Infof("--- AVP Node Listening at %s ---", conf.GRPC.Port)

	if err := s.Serve(lis); err != nil {
		log.Panicf("failed to serve: %v", err)
	}
	select {}
}

func (s *server) StartProcess(ctx context.Context, in *pb.StartProcessRequest) (*pb.StartProcessReply, error) {
	log.Infof("process einfo=%v", in.Element)
	pipeline := s.avp.GetPipeline(in.Element.Mid)
	if pipeline == nil {
		return nil, errors.New("process: pipeline not found")
	}
	pipeline.AddElement(in.Element)
	return &pb.StartProcessReply{}, nil
}

func (s *server) StopProcess(ctx context.Context, in *pb.StopProcessRequest) (*pb.StopProcessReply, error) {
	log.Infof("publish unprocess=%v", in)
	return &pb.StopProcessReply{}, nil
}
