package invoker

import (
	"encoding/json"
	"net/rpc"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/aws/aws-lambda-go/lambda/messages"
	"github.com/awslabs/goformation/cloudformation/resources"
	uuid "github.com/satori/go.uuid"
	log "github.com/sirupsen/logrus"
	config "github.com/vrealzhou/lambda-local/config/docker"
)

type Go1xHandler struct {
	name       string
	arn        string
	handler    string
	port       int
	timeoutSec int64
	pid        int
	mutex      *sync.Mutex
}

func (h *Go1xHandler) Runtime() string {
	return "go1.x"
}

func (h *Go1xHandler) Init(name string, function *resources.AWSServerlessFunction) {
	h.name = name
	h.arn = name
	h.timeoutSec = int64(function.Timeout)
	h.mutex = &sync.Mutex{}
	h.handler = function.Handler
}

func (h *Go1xHandler) Name() string {
	return h.name
}

func (h *Go1xHandler) Arn() string {
	return h.arn
}

func (h *Go1xHandler) Start(envs []string) error {
	h.port = h.pickPort()
	envs = append(envs, "_LAMBDA_SERVER_PORT="+strconv.Itoa(h.port))
	command := filepath.Join(config.LambdaBase(), h.name, h.handler)
	log.Debugf("Command: %s, envs: %v\n", command, envs)
	cmd := exec.Command(command)
	cmd.Dir = filepath.Join(config.LambdaBase(), h.name)
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
	registerFunc(h.name, h)
	go func() {
		defer deregisterFunc(h.name)
		if err := cmd.Start(); err != nil {
			log.Errorf("Error on starting function %s: %s", h.name, err.Error())
			return
		}
		h.pid = cmd.Process.Pid
		log.Debugf("Function %s started, Pid is %d", h.name, h.pid)
		if err := cmd.Wait(); err != nil {
			log.Errorf("Function %s returned error: %v", h.name, err)
		}
		log.Debugf("Function %s finished", h.name)
	}()
	err = h.waitFuncReady()
	if err != nil {
		log.Fatalf("Lambda %s is crashed: %s", h.name, err)
	}
	return nil
}
func (h *Go1xHandler) Stop() error {
	defer deregisterFunc(h.name)
	proc, err := os.FindProcess(h.pid)
	if err != nil {
		return err
	}
	return proc.Kill()
}

func (h *Go1xHandler) Invoke(payload []byte) ([]byte, error) {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	start := time.Now()
	log.Debugf("Invoke Function %s with Payload: %s", h.name, string(payload))
	log.Infof("Start Invoke Function %s at: %s\n", h.name, start.Format("2006/01/02 15:04:05"))
	client, err := rpc.Dial("tcp", "localhost:"+strconv.Itoa(h.port))
	if err != nil {
		return nil, err
	}
	defer client.Close()
	u1 := uuid.NewV4()
	req := &messages.InvokeRequest{
		Payload:   payload,
		RequestId: u1.String(),
		Deadline: messages.InvokeRequest_Timestamp{
			Seconds: time.Now().Unix() + h.timeoutSec,
		},
		InvokedFunctionArn: h.arn,
	}
	response := &messages.InvokeResponse{}
	invokeStart := time.Now()
	err = client.Call("Function.Invoke", req, response)
	invokeEnd := time.Now()
	log.Infof("Invoke Function %s preparing took: %s; Invoke took: %s; Total time cost: %s\n", h.name, invokeStart.Sub(start), invokeEnd.Sub(invokeStart), invokeEnd.Sub(start))
	if err != nil {
		return nil, err
	}
	if response.Error != nil {
		wrap := fromInvokErr(response.Error)
		wrappedErr, _ := json.Marshal(wrap)
		log.Debugf("Function %s with result: %s", h.name, string(wrappedErr))
		return wrappedErr, ErrExec
	}
	log.Debugf("Function %s with result: %s", h.name, string(response.Payload))
	return response.Payload, nil
}

func (h *Go1xHandler) Property(key string) interface{} {
	switch key {
	case InvokerPropertyPort:
		return h.port
	}
	return nil
}

// waitFuncReady waits for specified Lambda function service ready to invoke.
func (h *Go1xHandler) waitFuncReady() error {
	for {
		err := h.pingFunc()
		if err != nil {
			if err == ErrConn {
				time.Sleep(50 * time.Millisecond)
			} else {
				return err
			}
		} else {
			return nil
		}
	}
}

func (h *Go1xHandler) pingFunc() error {
	client, err := rpc.Dial("tcp", "localhost:"+strconv.Itoa(h.port))
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

func (h *Go1xHandler) pickPort() int {
	// find available port between 2000-3000
	for port := 2000; port < 3000; port++ {
		found := false
		for _, m := range Functions {
			if m.Property(InvokerPropertyPort) == port {
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
