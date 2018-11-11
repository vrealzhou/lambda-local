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
	config "github.com/vrealzhou/lambda-local/config/docker"
	"github.com/vrealzhou/lambda-local/internal/template"
)

var Functions = make(map[string]*FunctionMeta)
var ConnError = errors.New("Conn Error")
var ExecError = errors.New("Exec Error")

type FunctionMeta struct {
	Name       string
	Arn        string
	Port       int
	TimeoutSec int64
	Pid        int
}

func PrepareFunction(name string, function template.Function) error {
	if _, ok := Functions[name]; ok {
		return nil
	}
	meta := &FunctionMeta{
		Name:       name,
		Arn:        name,
		TimeoutSec: int64(function.Properties.Timeout),
	}
	// locate exec file
	splits := strings.Split(function.Properties.CodeURI, "/")
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
	for port := 2000; port < 3000; port++ {
		found := false
		for _, m := range Functions {
			if m.Port == port {
				found = true
				break
			}
		}
		if !found {
			meta.Port = port
			break
		}
	}
	command := filepath.Join(config.LambdaBase(), name, function.Properties.Handler)
	env := make([]string, 0)
	for key, val := range function.Properties.Environment.Variables {
		if os.Getenv(key) == "" {
			env = append(env, key+"="+val)
		}
	}
	env = append(env, os.Environ()...)
	env = append(env, "_LAMBDA_SERVER_PORT="+strconv.Itoa(meta.Port))
	log.Debugf("Command: %s, env: %v\n", command, env)
	cmd := exec.Command(command)
	cmd.Env = env
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

func WaitFuncReady(meta *FunctionMeta) {
	for {
		err := pingFunc(meta)
		if err != nil {
			if err == ConnError {
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
		return ConnError
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
		wrap := FromInvokErr(response.Error)
		wrappedErr, _ := json.Marshal(wrap)
		log.Debugf("Function %s with result: %s", meta.Name, string(wrappedErr))
		return wrappedErr, ExecError
	}
	log.Debugf("Function %s with result: %s", meta.Name, string(response.Payload))
	return response.Payload, nil
}

func FromInvokErr(e *messages.InvokeResponse_Error) errWrapper {
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
