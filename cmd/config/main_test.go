package main

import (
	"testing"
	"log"
	"os"
	"path"
	"fmt"
	"io/ioutil"
	"github.com/mozey/logutil"
	"math/rand"
	"time"
	"github.com/mozey/go-capturer"
	"github.com/stretchr/testify/require"
	"encoding/json"
)

// https://stackoverflow.com/a/22892986/639133
var letters = []rune("ABCDEFGHIJKLMNOPQRSTUVWXYZ")
func randString(n int) string {
	rand.Seed(time.Now().UnixNano())
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

var key1 string
var value1 string
var configJson string
func resetConfigFile(env string) {
	var config string
	config = path.Join(AppDir, fmt.Sprintf("config.%v.json", env))
	key1 = fmt.Sprintf("%v_FOO", Prefix)
	value1 = "bar"
	configJson = fmt.Sprintf("{\"%v\":\"%v\"}", key1, value1)
	data := []byte(configJson)
	err := ioutil.WriteFile(config, data, 0644)
	if err != nil {
		logutil.Debugf("Loading config from: %v", config)
		log.Panic(err)
	}
}

func setFlags() {
	goPath := os.Getenv("GOPATH")
	AppDir = path.Join(goPath, "src", "github.com", "mozey", "config")
	Prefix = randString(15)
	e := "dev"; Env = &e
	u := false; Update = &u
}

func TestPrintEnvCommands(t *testing.T) {
	setFlags()
	resetConfigFile("dev")
	os.Setenv(fmt.Sprintf("%v_BAZ", Prefix), "123")
	b := capturer.CaptureStdout(Cmd)
	s := string(b)
	logutil.Debug("\n", s)
	require.Contains(t, s, fmt.Sprintf("export %v_FOO=bar", Prefix))
	require.Contains(t, s, fmt.Sprintf("unset %v_BAZ", Prefix))
}

func TestUpdateConfig(t *testing.T) {
	setFlags()
	resetConfigFile("dev")
	key2 := fmt.Sprintf("%v_BAZ", Prefix)
	Keys.Set(key2)
	value2 := "456"
	Values.Set(value2)

	// Prints new config if update is not set
	b := capturer.CaptureStdout(Cmd)
	c := ConfigMap{}
	err := json.Unmarshal(b, &c)
	if err != nil {
		logutil.Debugf("Unmarshal captured output: %v", string(b))
		log.Panic(err)
	}
	s := string(b)
	logutil.Debug("\n", s)
	require.Equal(t, value1, c[key1])
	require.Equal(t, value2, c[key2])

	// Check config file is updated
	u := true; Update = &u
	Cmd()
	configPath := GetConfigPath()
	b, err = ioutil.ReadFile(configPath)
	if err != nil {
		logutil.Debugf("Loading config from: %v", configPath)
		log.Panic(err)
	}
	s = string(b)
	logutil.Debug("\n", s)
	require.Equal(t, value1, c[key1])
	require.Equal(t, value2, c[key2])
}

// TODO Test LoadFile
