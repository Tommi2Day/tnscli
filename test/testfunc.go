// Package test defines path settings while testing
package test

// https://intellij-support.jetbrains.com/hc/en-us/community/posts/360009685279-Go-test-working-directory-keeps-changing-to-dir-of-the-test-file-instead-of-value-in-template
import (
	"bytes"
	"os"
	"path"
	"runtime"
	"testing"

	tnscmd "github.com/tommi2day/tnscli/cmd"

	"github.com/sirupsen/logrus"
)

// TestDir working dir for tests
var TestDir string

// TestData directory for working files
var TestData string

func init() {
	_, filename, _, _ := runtime.Caller(0)
	dir := path.Dir(filename)
	err := os.Chdir(dir)
	if err != nil {
		panic(err)
	}
	TestDir = dir
	TestData = path.Join(TestDir, "testdata")
	// create data directory and ignore errors
	err = os.Mkdir(TestData, 0750)
	if err != nil && !os.IsExist(err) {
		panic(err)
	}
	println("Work in " + dir)
}

// Testinit alternative init Test directories
func Testinit(t *testing.T) {
	_, filename, _, _ := runtime.Caller(0)
	dir := path.Dir(filename)
	err := os.Chdir(dir)
	if err == nil {
		TestDir = dir
		TestData = path.Join(TestDir, "testdata")
		// create data directory and ignore errors
		err = os.Mkdir(TestData, 0750)
		if err != nil && !os.IsExist(err) {
			t.Fatalf("Init error:%s", err)
		}
		t.Logf("Test in %s", dir)
	} else {
		t.Fatalf("Init error:%s", err)
	}
}

func cmdTest(args []string) (out string, err error) {
	cmd := tnscmd.RootCmd
	b := bytes.NewBufferString("")
	logrus.SetOutput(b)
	cmd.SetOut(b)
	cmd.SetErr(b)
	cmd.SetArgs(args)
	err = cmd.Execute()
	out = b.String()
	return
}
