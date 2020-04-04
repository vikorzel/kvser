package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/cucumber/godog"
	"gopkg.in/yaml.v2"
)

var qpsLimit = 0
var qpsChannel chan (int)
var inited = false
var _ = godog.Version
var ownPath, _ = filepath.Abs(filepath.Dir(os.Args[0]))
var tmpPath = filepath.Join(ownPath, "tmp_basic")
var serverPort = 0

type luaConfig struct {
	Address  string `json:"address"`
	Port     int    `json:"port"`
	QPSLimit int    `json:"qps_limit"`
}

type dockerConfig struct {
	Version  string `yaml:"version"`
	Services struct {
		Tarantool struct {
			Build   string
			Ports   []string
			Volumes []string
			Image   string
		}
	}
}

func limitOfOurRequestsAsPerSec(_qpsLimit int) error {
	qpsLimit = _qpsLimit
	return nil
}

func localServerOnPortInDebugModeWithQPSLimitSetAs(port, qps int) error {
	if inited {
		return nil
	}
	serverPort = port
	cfgPath := MakeLuaConfig(tmpPath, port, qps)
	MakeDockerComposeConfig(tmpPath, cfgPath, port)
	StartDockerCompose()
	inited = true
	return nil
}

func noLimitOfOurRequests() error {
	qpsLimit = 0
	return nil
}

func weReceiveCodeTimesWithTolerance(expectCode, repeatTimes int) error {
	cntr := 0
	chanLen := len(qpsChannel)
	for i := 0; i != chanLen; i++ {
		code, _ := <-qpsChannel
		if code == expectCode {
			cntr++
		}
	}

	if math.Abs(float64(repeatTimes-cntr)) <= 1 {
		return nil
	}

	return fmt.Errorf("We are waitin code %v repeated %v times, but actualy is %v", expectCode, repeatTimes, cntr)
}

func weSendGETRequests(arg1 int) error {
	qpsChannel = make(chan int, arg1)
	for connectionsNum := 0; connectionsNum != arg1; connectionsNum++ {
		if qpsLimit > 0 && connectionsNum%qpsLimit == 0 {
			time.Sleep(time.Second)
		}
		go func(ch chan int) {
			resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%v/kv/test", serverPort))
			if err != nil {
				fmt.Printf("GET Error: %s", err)
			}
			ch <- resp.StatusCode
			if resp.StatusCode == 200 {
				d, _ := ioutil.ReadAll(resp.Body)

				fmt.Println(string(d))
			}
		}(qpsChannel)
	}

	for len(qpsChannel) < arg1 {
		time.Sleep(time.Second)
	}
	return nil
}

//UTILS

//MakeLuaConfig generate serverconfig.json
func MakeLuaConfig(tmpPath string, port int, qps int) string {
	configPath := filepath.Join(tmpPath, "config")
	os.MkdirAll(configPath, 0777)
	config := luaConfig{"0.0.0.0", port, qps}
	rawJSON, _ := json.Marshal(config)
	ioutil.WriteFile(filepath.Join(configPath, "serverconfig.json"), rawJSON, 0644)
	return configPath
}

// MakeDockerComposeConfig generate docker-compose config
func MakeDockerComposeConfig(tmpPath string, configPath string, port int) {
	dockerLocalConfig := dockerConfig{}

	dockerLocalConfig.Version = "2"
	dockerLocalConfig.Services.Tarantool.Build = "."
	dockerLocalConfig.Services.Tarantool.Ports = make([]string, 1)
	dockerLocalConfig.Services.Tarantool.Ports[0] = fmt.Sprintf("%v:%v", port, port)
	dockerLocalConfig.Services.Tarantool.Volumes = make([]string, 4)

	dataPath := filepath.Join(tmpPath, "data")
	os.MkdirAll(dataPath, 0777)

	dockerLocalConfig.Services.Tarantool.Volumes[0] = fmt.Sprintf("%s:%s", filepath.Join(ownPath, "../../lua"), "/opt/tarantool/lua")
	dockerLocalConfig.Services.Tarantool.Volumes[1] = fmt.Sprintf("%s:%s", dataPath, "/var/lib/tarantool")
	dockerLocalConfig.Services.Tarantool.Volumes[2] = fmt.Sprintf("%s:%s", configPath, "/opt/tarantool/config")
	dockerLocalConfig.Services.Tarantool.Volumes[3] = fmt.Sprintf("%s:%s", tmpPath, "/opt/tarantool/log")

	dockerLocalConfig.Services.Tarantool.Image = "tarantool/tarantool:2.3.1"
	rawYaml, _ := yaml.Marshal(&dockerLocalConfig)
	ioutil.WriteFile("docker-compose.yml", rawYaml, 0644)
}

//StartDockerCompose just for starting docker
func StartDockerCompose() {
	exec.Command("docker-compose", "pull").Run()
	exec.Command("docker-compose", "create", "--force-recreate").Run()
	exec.Command("docker-compose", "up", "-d").Run()
	cmd := exec.Command("docker-compose", "exec", "tarantool", "tarantool", "/opt/tarantool/lua/app.lua")
	err := cmd.Start()
	if err != nil {
		fmt.Println("Start tarantool app error:", err)
	} else {
		cmd.Process.Release()
		time.Sleep(time.Second * 7)
	}
}

//StopDocker just for killing the docker
func StopDocker() {
	exec.Command("docker-compose", "stop", "tarantool").Run()
	exec.Command("docker-compose", "rm", "-f", "-v", "tarantool").Run()
}

//InitDirs is recreate tmp dirs
func InitDirs() {
	os.Remove(tmpPath)
	os.MkdirAll(tmpPath, 0777)
	os.Chdir(tmpPath)
}

func FeatureContext(s *godog.Suite) {

	s.Step(`^limit of our requests as (\d+) per sec$`, limitOfOurRequestsAsPerSec)
	s.Step(`^local server on port (\d+) in debug mode with QPS limit set as (\d+)$`, localServerOnPortInDebugModeWithQPSLimitSetAs)
	s.Step(`^no limit of our requests$`, noLimitOfOurRequests)
	s.Step(`^we receive (\d+) code (\d+) times$`, weReceiveCodeTimesWithTolerance)
	s.Step(`^we send (\d+) GET requests$`, weSendGETRequests)

	s.BeforeSuite(InitDirs)
	s.AfterSuite(StopDocker)

}
