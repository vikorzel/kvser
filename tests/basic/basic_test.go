package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/cucumber/godog"
	"gopkg.in/yaml.v2"
)

var lastResp *http.Response
var lastJSON string
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

type POSTPacket struct {
	Key  string `json:"key"`
	Body string `json:"body"`
}

///INIT

func initFeature() {
	InitDirs(tmpPath)
}

///GIVEN

func localServerOnPortInDebugMode(port int) error {
	if inited {
		return nil
	}
	serverPort = port
	cfgPath := MakeLuaConfig(tmpPath, port, 0)
	MakeDockerComposeConfig(tmpPath, cfgPath, port)
	StartDockerCompose()
	inited = true
	return nil
}

///WHEN

func weSendWithKeyAsAndValidJSONAsBody(sendType string, key string, validType string, missedElement string) error {
	const validJSON string = `{
		"glossary": {
			"title": "example glossary",
			"GlossDiv": {
				"title": "S",
				"GlossList": {
					"GlossEntry": {
						"ID": "SGML",
						"SortAs": "SGML",
						"GlossTerm": "Standard Generalized Markup Language",
						"Acronym": "SGML",
						"Abbrev": "ISO 8879:1986",
						"GlossDef": {
							"para": "A meta-markup language, used to create markup languages such as DocBook.",
							"GlossSeeAlso": ["GML", "XML"]
						},
						"GlossSee": "markup"
					}
				}
			}
		}
	}`

	const invalidJSON string = `{
		"glossary": {
			"title": "example glossary"
			"GlossDiv": {
				"title": "S",
				"GlossList": {
					"GlossEntry": {
						"ID": "SGML",
						"SortAs": "SGML",
						"GlossTerm": "Standard Generalized Markup Language",
						"Acronym": "SGML",
						"Abbrev": "ISO 8879:1986",
						"GlossDef": {
							"para": "A meta-markup language, used to create markup languages such as DocBook.",
							"GlossSeeAlso": ["GML", "XML"]
						},
						"GlossSee": "markup"
					}
				}
			}
		}
	}`

	var isValid bool = (validType == "valid")
	var isUniq bool = (validType == "uniq")
	var rawJSON string
	if isValid {
		rawJSON = validJSON
	} else if isUniq {
		rawJSON = generateNewJSONStr()
	} else {
		rawJSON = invalidJSON
	}

	var req *http.Request
	var err error
	if sendType == "POST" {
		if missedElement == "key" {
			req, err = http.NewRequest("POST", fmt.Sprintf("http://127.0.0.1:%v/kv", serverPort), strings.NewReader(fmt.Sprintf(`{"value":%s}`, rawJSON)))
		} else if missedElement == "value" {
			req, err = http.NewRequest("POST", fmt.Sprintf("http://127.0.0.1:%v/kv", serverPort), strings.NewReader(fmt.Sprintf(`{"key":"%s"}`, key)))
		} else {
			req, err = http.NewRequest("POST", fmt.Sprintf("http://127.0.0.1:%v/kv", serverPort), strings.NewReader(fmt.Sprintf(`{"key":"%s", "value":%s}`, key, rawJSON)))
		}
	} else {
		req, err = http.NewRequest("PUT", fmt.Sprintf("http://127.0.0.1:%v/kv/%s", serverPort, key), strings.NewReader(rawJSON))
	}

	cli := &http.Client{}
	lastResp, err = cli.Do(req)
	if err != nil {
		fmt.Println(err)
	}
	return nil
}

func weSendGETDEL(reqtype string, key string) error {
	url := fmt.Sprintf("http://127.0.0.1:%v/kv/%s", serverPort, key)
	if reqtype == "GET" {
		lastResp, _ = http.Get(url)
	}
	if reqtype == "DELETE" {
		req, _ := http.NewRequest("DELETE", url, nil)
		cli := &http.Client{}
		lastResp, _ = cli.Do(req)
	}
	return nil
}

///THEN

func theResponseBodyWillBeContian(arg1 string) error {
	respBody := getRespBody()
	if strings.Contains(respBody, arg1) {
		return nil
	}
	return fmt.Errorf("We are failed this request. Expect BODY contains %s but actualy is %s ", arg1, respBody)
}

func theResponseCodeWillBe(code int) error {
	if lastResp.StatusCode == code {
		return nil
	}
	respBody := getRespBody()
	return fmt.Errorf("We are failed this request. Expect CODE %v but actualy is %v (%s)", code, lastResp.StatusCode, respBody)

}

func theResponseBodyWillContainsTheSameJSON() error {
	var sent map[string]string
	var received map[string]string

	recBody := getRespBody()

	json.Unmarshal([]byte(lastJSON), &sent)
	json.Unmarshal([]byte(recBody), &received)

	for k := range sent {
		if sent[k] != received[k] {
			return fmt.Errorf("We are failed this request. Expect BODY %s but actualy is %s", lastJSON, recBody)
		}
	}
	return nil
}

///UTILS

func generateNewJSONStr() string {
	lastJSON = fmt.Sprintf(`{"firs":"%v","meat":"%v","kernes":"%v"}`, rand.Int(), rand.Int(), rand.Int())
	return lastJSON
}

func getRespBody() string {
	d, _ := ioutil.ReadAll(lastResp.Body)
	return string(d)
}

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
func InitDirs(tmpPath string) {
	os.Remove(tmpPath)
	os.MkdirAll(tmpPath, 0777)
	os.Chdir(tmpPath)
}

func FeatureContext(s *godog.Suite) {
	s.Step(`^local server on port (\d+) in debug mode$`, localServerOnPortInDebugMode)
	s.Step(`^the response body will be contain "(.*)"$`, theResponseBodyWillBeContian)
	s.Step(`^the response code will be (\d+)$`, theResponseCodeWillBe)
	s.Step(`^we send (POST|PUT) with key as "([^"]*)" and (invalid|valid|uniq) JSON as body(?: but without the (key|value))?$`, weSendWithKeyAsAndValidJSONAsBody)
	s.Step(`^we send (GET|DELETE) with key as "([^"]*)"$`, weSendGETDEL)
	s.Step(`^the response body will contains the same JSON$`, theResponseBodyWillContainsTheSameJSON)

	s.BeforeSuite(initFeature)
	s.AfterSuite(StopDocker)
}
