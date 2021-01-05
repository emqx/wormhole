package common

import (
	"fmt"
	"github.com/go-yaml/yaml"
	filename "github.com/keepeye/logrus-filename"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
)

type (
	BasicConfig struct {
		QuicBindAddr string `yaml:"quicBindAddr"`
		QuicBindPort int    `yaml:"quicBindPort"`
		Debug        bool   `yaml:"debug"`
		ConsoleLog   bool   `yaml:"consoleLog"`
		LogPath      string `yaml:"logPath"`
	}

	ServerConfig struct {
		Basic BasicConfig
		RestBindAddr string `yaml:"restBindAddr"`
		RestBindPort int    `yaml:"restBindPort"`
	}

	ClientConfig struct {
		Basic BasicConfig
		HttpTimeout int `yaml:"httpTimeout"`
	}
)

//var WormholeBaseKey = "WormholeBaseKey"
var Log *logrus.Logger
var serverConf *ServerConfig
var clientConf *ClientConfig

func GetSrvConf() (*ServerConfig, bool) {
	if serverConf == nil {
		serverConf = &ServerConfig{}
		return serverConf, serverConf.initSrvConfig()
	}
	return serverConf, true
}

func GetClientConf() (*ClientConfig, bool) {
	if clientConf == nil {
		clientConf = &ClientConfig{}
		return clientConf, clientConf.initClientConfig()
	}
	return clientConf, clientConf.initClientConfig()
}

func processPath(path string) (string, error) {
	if abs, err := filepath.Abs(path); err != nil {
		return "", nil
	} else {
		if _, err := os.Stat(abs); os.IsNotExist(err) {
			return "", err
		}
		return abs, nil
	}
}

func loadConf(fname string) ([]byte, bool) {
	var confPath, err = processPath("etc/" + fname)
	content, err := ioutil.ReadFile(confPath)
	if nil != err {
		fmt.Println("load conf err : ", err)
		return nil, false
	}
	return content, true
}

func (conf BasicConfig) validateLogSettings() bool {
	var err error
	if conf.LogPath, err = filepath.Abs(conf.LogPath); nil != err {
		fmt.Println("log dir err : ", err)
		return false
	}
	if _, err = os.Stat(conf.LogPath); os.IsNotExist(err) {
		if err = os.MkdirAll(path.Dir(conf.LogPath), 0755); nil != err {
			fmt.Println("make logdir err : ", err)
			return false
		}
	}

	Log = logrus.New()
	filenameHook := filename.NewHook()
	filenameHook.Field = "file"
	Log.AddHook(filenameHook)

	Log.SetFormatter(&logrus.TextFormatter{
		TimestampFormat: "2006-01-02 15:04:05",
		DisableColors:   true,
		FullTimestamp:   true,
	})

	logFile, err := os.OpenFile(conf.LogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err == nil {
		Log.SetOutput(logFile)
		return true
	} else {
		Log.Infof("Failed to log to file, using default stderr.")
		return true
	}
}

func (conf *ServerConfig) initSrvConfig() bool {
	d, ok := loadConf("server.yaml")
	if !ok {
		return false
	}

	err := yaml.Unmarshal(d, conf)
	if nil != err {
		fmt.Println("unmashal conf err : ", err)
		return false
	}

	if !conf.Basic.validateLogSettings() {
		return false
	}
	return true
}

func (conf *ClientConfig) initClientConfig() bool {
	d, ok := loadConf("client.yaml")
	if !ok {
		return false
	}

	err := yaml.Unmarshal(d, conf)
	if nil != err {
		fmt.Println("unmashal conf err : ", err)
		return false
	}
	if !conf.Basic.validateLogSettings() {
		return false
	}
	return true
}