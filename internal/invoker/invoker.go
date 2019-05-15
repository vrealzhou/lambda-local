package invoker

import (
	"archive/zip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/rpc"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/lambda/messages"
	uuid "github.com/satori/go.uuid"
	log "github.com/sirupsen/logrus"

	"github.com/vrealzhou/goformation/cloudformation/resources"
	config "github.com/vrealzhou/lambda-local/config/docker"
)

// Functions holds all functions' meta data
var Functions = make(map[string]*FunctionMeta)

// ErrConn defines connection error to lambda
var ErrConn = errors.New("Conn Error")

// ErrExec defines error on execute lambda
var ErrExec = errors.New("Exec Error")

var envSettings = make(map[string]map[string]string)

// FunctionMeta defines struct of function meta data
type FunctionMeta struct {
	Name       string
	Arn        string
	Port       int
	TimeoutSec int64
	Pid        int
}

// PrepareFunction preload function and make it ready to invoke
func PrepareFunction(name string, function *resources.AWSServerlessFunction) error {
	if _, ok := Functions[name]; ok {
		return nil
	}
	meta := &FunctionMeta{
		Name:       name,
		Arn:        name,
		TimeoutSec: int64(function.Timeout),
	}
	// locate exec file
	splits := strings.Split(*function.CodeUri.String, "/")
	zipFile := filepath.Join(config.LambdaBase(), splits[len(splits)-1])
	target := filepath.Join(config.LambdaBase(), name)
	log.Debugf("lambda zip file: %s, lambda exec path: %s\n", zipFile, target)
	if _, err := os.Stat(target); os.IsNotExist(err) {
		err := os.MkdirAll(target, os.ModePerm)
		if err != nil {
			return err
		}
	}
	err := unzip(zipFile, target)
	if err != nil {
		return err
	}
	// find available port between 2000-3000
	meta.Port = pickPort()
	command := filepath.Join(config.LambdaBase(), name, function.Handler)
	envs := generateEnvs(name, function, meta)
	log.Debugf("Command: %s, envs: %v\n", command, envs)
	cmd := exec.Command(command)
	cmd.Env = envs
	stdoutIn, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	go func() {
		copyAndCapture(os.Stdout, stdoutIn)
	}()
	stderrIn, _ := cmd.StderrPipe()
	if err != nil {
		return err
	}
	go func() {
		copyAndCapture(os.Stderr, stderrIn)
	}()
	Functions[name] = meta
	go func() {
		defer delete(Functions, name)
		if err := cmd.Start(); err != nil {
			log.Errorf("Error on starting function %s: %s", name, err.Error())
			return
		}
		meta.Pid = cmd.Process.Pid
		log.Debugf("Function %s started, Pid is %d", name, meta.Pid)
		if err := cmd.Wait(); err != nil {
			log.Errorf("Function %s returned error: %v", name, err)
		}
		log.Debugf("Function %s finished", name)
	}()
	WaitFuncReady(meta)
	return nil
}

func generateEnvs(name string, function *resources.AWSServerlessFunction, meta *FunctionMeta) []string {
	envMap := make(map[string]string)
	envs := make([]string, 0)
	for key, val := range function.Environment.Variables {
		envMap[key] = val
	}
	for _, env := range os.Environ() {
		index := strings.Index(env, "=")
		if index <= 0 || index == len(env)-1 {
			continue
		}
		key := env[:index]
		val := env[index+1:]
		if strings.HasPrefix(key, "AWS_") {
			envMap[key] = val
		}
		if _, ok := envMap[key]; ok {
			envMap[key] = val
		}
	}
	if extraEnvs, ok := envSettings[name]; ok {
		for key, val := range extraEnvs {
			if strings.HasPrefix(key, "AWS_") {
				envMap[key] = val
			}
			if _, ok := envMap[key]; ok {
				envMap[key] = val
			}
		}
	}
	for k, v := range envMap {
		envs = append(envs, k+"="+v)
	}
	envs = append(envs, "_LAMBDA_SERVER_PORT="+strconv.Itoa(meta.Port))
	return envs
}

func copyAndCapture(w io.Writer, r io.Reader) {
	buf := make([]byte, 1024, 1024)
	for {
		n, err := r.Read(buf[:])
		if n > 0 {
			_, err := w.Write(buf[:n])
			if err != nil {
				panic(err)
			}
		}
		if err != nil {
			// Read returns io.EOF at the end of file, which is not an error for us
			if err == io.EOF {
				err = nil
			}
			return
		}
	}
}

// WaitFuncReady waits for specified Lambda function service ready to invoke.
func WaitFuncReady(meta *FunctionMeta) {
	for {
		err := pingFunc(meta)
		if err != nil {
			if err == ErrConn {
				time.Sleep(50 * time.Millisecond)
			} else {
				log.Fatalf("Lambda %s is crashed: %s", meta.Name, err)
				return
			}
		} else {
			return
		}
	}
}

func pingFunc(meta *FunctionMeta) error {
	client, err := rpc.Dial("tcp", "localhost:"+strconv.Itoa(meta.Port))
	if err != nil {
		return ErrConn
	}
	defer client.Close()
	req := &messages.PingRequest{}
	response := &messages.PingResponse{}
	err = client.Call("Function.Ping", req, response)
	if err != nil {
		return err
	}
	return nil
}

// InvokeFunc do the invoke operation to specified Lambda function
func InvokeFunc(meta *FunctionMeta, payload []byte) (json.RawMessage, error) {
	start := time.Now()
	log.Debugf("Invoke Function %s with Payload: %s", meta.Name, string(payload))
	log.Infof("Start Invoke Function %s at: %s\n", meta.Name, start.Format("2006/01/02 15:04:05"))
	client, err := rpc.Dial("tcp", "localhost:"+strconv.Itoa(meta.Port))
	if err != nil {
		return nil, err
	}
	defer client.Close()
	u1 := uuid.NewV4()
	req := &messages.InvokeRequest{
		Payload:   payload,
		RequestId: u1.String(),
		Deadline: messages.InvokeRequest_Timestamp{
			Seconds: time.Now().Unix() + meta.TimeoutSec,
		},
		InvokedFunctionArn: meta.Arn,
	}
	response := &messages.InvokeResponse{}
	invokeStart := time.Now()
	err = client.Call("Function.Invoke", req, response)
	invokeEnd := time.Now()
	log.Infof("Invoke Function %s preparing took: %s; Invoke took: %s; Total time cost: %s\n", meta.Name, invokeStart.Sub(start), invokeEnd.Sub(invokeStart), invokeEnd.Sub(start))
	if err != nil {
		return nil, err
	}
	if response.Error != nil {
		wrap := fromInvokErr(response.Error)
		wrappedErr, _ := json.Marshal(wrap)
		log.Debugf("Function %s with result: %s", meta.Name, string(wrappedErr))
		return wrappedErr, ErrExec
	}
	log.Debugf("Function %s with result: %s", meta.Name, string(response.Payload))
	return response.Payload, nil
}

// fromInvokErr wraps the raw error from lambda to standard lambda error struct.
func fromInvokErr(e *messages.InvokeResponse_Error) errWrapper {
	wrap := errWrapper{
		ErrorMessage: e.Message,
		ErrorType:    e.Type,
	}
	if e.StackTrace != nil {
		stackTrace := make([]errStackTrace, 0)
		for _, trace := range e.StackTrace {
			t := errStackTrace{
				Path:  trace.Path,
				Line:  trace.Line,
				Label: trace.Label,
			}
			stackTrace = append(stackTrace, t)
		}
		wrap.StackTrace = stackTrace
	}
	return wrap
}

// ErrWrapper used for wrap unhandled error message from lambda
type errWrapper struct {
	ErrorMessage string          `json:"errorMessage,omitempty"`
	ErrorType    string          `json:"errorType,omitempty"`
	StackTrace   []errStackTrace `json:"stackTrace,omitempty"`
}

type errStackTrace struct {
	Path  string `json:"path"`
	Line  int32  `json:"line"`
	Label string `json:"label"`
}

func unzip(src string, target string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		rc, err := f.Open()
		if err != nil {
			return err
		}
		defer rc.Close()

		// Store filename/path for returning and using later on
		fpath := filepath.Join(target, f.Name)
		log.Debugf("Unzip %s to %s\n", f.Name, fpath)

		// Check for ZipSlip. More Info: http://bit.ly/2MsjAWE
		if !strings.HasPrefix(fpath, filepath.Clean(target)+string(os.PathSeparator)) {
			return fmt.Errorf("%s: invalid file path", fpath)
		}

		if f.FileInfo().IsDir() {
			os.MkdirAll(fpath, os.ModePerm)
		} else {
			// Make File
			if err = os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
				return err
			}

			outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
			if err != nil {
				return err
			}
			_, err = io.Copy(outFile, rc)
			outFile.Close()
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func pickPort() int {
	// find available port between 2000-3000
	for port := 2000; port < 3000; port++ {
		found := false
		for _, m := range Functions {
			if m.Port == port {
				found = true
				break
			}
		}
		if !found {
			return port
		}
	}
	return 0
}

// LoadEnvFile loads extra env json file
func LoadEnvFile(file string) error {
	if _, err := os.Stat(file); os.IsNotExist(err) {
		return nil
	}
	f, err := os.Open(file)
	if err != nil {
		return fmt.Errorf("Error on opening env file %s: %s", file, err.Error())
	}
	err = json.NewDecoder(f).Decode(&envSettings)
	if err != nil {
		return fmt.Errorf("Error on parsing env file %s: %s", file, err.Error())
	}
	return nil
}
