package invoker

import (
	"archive/zip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/rpc"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/lambda/messages"
	uuid "github.com/satori/go.uuid"
	"github.com/vrealzhou/lambda-local/internal/template"
)

var LambdaBinBase = "/lambdas"
var Functions = make(map[string]*FunctionMeta)
var ConnError = errors.New("Conn Error")

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
	zipFile := filepath.Join(LambdaBinBase, splits[len(splits)-1])
	target := filepath.Join(LambdaBinBase, name)
	log.Printf("lambda zip file: %s, lambda exec path: %s\n", zipFile, target)
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
	command := filepath.Join(LambdaBinBase, name, function.Properties.Handler)
	env := make([]string, 0)
	for key, val := range function.Properties.Environment.Variables {
		if os.Getenv(key) == "" {
			env = append(env, key+"="+val)
		}
	}
	env = append(env, os.Environ()...)
	env = append(env, "_LAMBDA_SERVER_PORT="+strconv.Itoa(meta.Port))
	log.Printf("Command: %s, env: %v\n", command, env)
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
			log.Printf("Error on starting function %s: %s", name, err.Error())
			return
		}
		meta.Pid = cmd.Process.Pid
		log.Printf("Function %s started, Pid is %d", name, meta.Pid)
		if err := cmd.Wait(); err != nil {
			log.Printf("Function %s returned error: %v", name, err)
		}
		log.Printf("Function %s finished", name)
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
				log.Fatal(err)
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
	log.Printf("Start Invoke Function %s at: %s\n", meta.Name, start.Format("2006/01/02 15:04:05"))
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
			Seconds: meta.TimeoutSec,
		},
		InvokedFunctionArn: meta.Arn,
	}
	response := &messages.InvokeResponse{}
	invokeStart := time.Now()
	err = client.Call("Function.Invoke", req, response)
	invokeEnd := time.Now()
	log.Printf("Invoke Function %s preparing took: %s; Invoke took: %s; Total time cost: %s; Payload: %s\n", meta.Name, invokeStart.Sub(start), invokeEnd.Sub(invokeStart), invokeEnd.Sub(start), string(response.Payload))
	if err != nil {
		return nil, err
	}
	if response.Error != nil {
		return json.Marshal(response.Error)
	}
	return response.Payload, nil
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
		fmt.Printf("Unzip %s to %s", f.Name, fpath)

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
