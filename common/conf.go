package common

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"time"

	"github.com/go-yaml/yaml"
	"github.com/sirupsen/logrus"
)

type (
	config struct {
		Port         int    `yaml:"port"`
		Timeout      int    `yaml:"timeout"`
		IntervalTime int    `yaml:"intervalTime"`
		Ip           string `yaml:"ip"`
		LogPath      string `yaml:"logPath"`
	}
)

var g_conf config

func GetConf() *config {
	return &g_conf
}
func (this *config) GetIntervalTime() int {
	return this.IntervalTime
}
func (this *config) GetIp() string {
	return this.Ip
}
func (this *config) GetPort() int {
	return this.Port
}
func (this *config) GetLogPath() string {
	return this.LogPath
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

func (this *config) initConfig() bool {
	confPath, err := processPath(os.Args[1])
	if nil != err {
		fmt.Println("conf path err : ", err)
		return false
	}
	sliByte, err := ioutil.ReadFile(confPath)
	if nil != err {
		fmt.Println("load conf err : ", err)
		return false
	}
	err = yaml.Unmarshal(sliByte, this)
	if nil != err {
		fmt.Println("unmashal conf err : ", err)
		return false
	}

	if this.LogPath, err = filepath.Abs(this.LogPath); nil != err {
		fmt.Println("log dir err : ", err)
		return false
	}
	if _, err = os.Stat(this.LogPath); os.IsNotExist(err) {
		if err = os.MkdirAll(path.Dir(this.LogPath), 0755); nil != err {
			fmt.Println("mak logdir err : ", err)
			return false
		}
	}
	return true
}

var (
	Log      *logrus.Logger
	g_client http.Client
)

func (this *config) initTimeout() {
	g_client.Timeout = time.Duration(this.Timeout) * time.Millisecond
}

func (this *config) InitLog() bool {
	Log = logrus.New()
	Log.SetReportCaller(true)
	Log.SetFormatter(&logrus.TextFormatter{
		CallerPrettyfier: func(f *runtime.Frame) (string, string) {
			filename := path.Base(f.File)
			return "", fmt.Sprintf("%s:%d", filename, f.Line)
		},
		DisableColors: true,
		FullTimestamp: true,
	})

	logFile, err := os.OpenFile(this.LogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err == nil {
		Log.SetOutput(logFile)
		return true
	} else {
		Log.Infof("Failed to log to file, using default stderr.")
		return true
	}
}

func (this *config) Init() bool {
	if !this.initConfig() {
		return false
	}

	if !this.InitLog() {
		return false
	}
	this.initTimeout()
	return true
}