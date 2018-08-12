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

func resetConfigFile(env string) {
	var config string
	config = path.Join(AppDir, fmt.Sprintf("config.%v.json", env))
	data := []byte(fmt.Sprintf("{\n\"%v_FOO\": \"bar\"\n}", Prefix))
	err := ioutil.WriteFile(config, data, 0644)
	if err != nil {
		logutil.Debugf("Loading config from: %v", config)
		log.Panic(err)
	}
}

func TestPrintEnvCommands(t *testing.T) {
	goPath := os.Getenv("GOPATH")
	AppDir = path.Join(goPath, "src", "github.com", "mozey", "config")
	Prefix = randString(15)
	os.Setenv(fmt.Sprintf("%v_BAZ", Prefix), "123")
	e := "dev"; Env = &e
	u := false; Update = &u
	resetConfigFile("dev")
	s := capturer.CaptureStdout(Cmd)
	//fmt.Println(s)
	require.Contains(t, s, fmt.Sprintf("unset %v_BAZ", Prefix))
	require.Contains(t, s, fmt.Sprintf("export %v_FOO=bar", Prefix))
}

func TestPrintConfig(t *testing.T) {

}

func TestUpdateConfig(t *testing.T) {

}


