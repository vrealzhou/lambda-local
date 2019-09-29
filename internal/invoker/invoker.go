package invoker

import (
	"archive/zip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/aws/aws-lambda-go/lambda/messages"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/awslabs/goformation/cloudformation/resources"
	log "github.com/sirupsen/logrus"

	config "github.com/vrealzhou/lambda-local/config/docker"
)

const (
	InvokerPropertyPort string = "port"
)

// Functions holds all functions' meta data
var Functions = make(map[string]Handler)

// ErrConn defines connection error to lambda
var ErrConn = errors.New("Conn Error")

// ErrExec defines error on execute lambda
var ErrExec = errors.New("Exec Error")

var envSettings = make(map[string]map[string]string)

type Handler interface {
	Runtime() string
	Arn() string
	Name() string
	Init(name string, function *resources.AWSServerlessFunction)
	Start(envs []string) error
	Stop() error
	Invoke(payload []byte) ([]byte, error)
	Property(key string) interface{}
}

func registerFunc(name string, handler Handler) error {
	if _, ok := Functions[name]; ok {
		return fmt.Errorf("Function %s already exists.", name)
	}
	Functions[name] = handler
	return nil
}

func deregisterFunc(name string) {
	if _, ok := Functions[name]; !ok {
		return
	}
	delete(Functions, name)
}

func getHandler(function *resources.AWSServerlessFunction) Handler {
	switch function.Runtime {
	case "go1.x":
		return &Go1xHandler{}
	default:
		return nil
	}
}

// PrepareFunction preload function and make it ready to invoke
func PrepareFunction(name string, function *resources.AWSServerlessFunction) error {
	if _, ok := Functions[name]; ok {
		return nil
	}
	h := getHandler(function)
	h.Init(name, function)
	// unzip lambda package
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
	envs := generateEnvs(name, function)
	return h.Start(envs)
}

func generateEnvs(name string, function *resources.AWSServerlessFunction) []string {
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
	// set AWS credentials from file if not set
	if !hasCredentials(envMap) {
		provider := &credentials.SharedCredentialsProvider{
			Profile: os.Getenv("AWS_DEFAULT_PROFILE"),
		}
		value, err := provider.Retrieve()
		if err != nil {
			log.Println("Error on retrive credentials: %s", err.Error())
		}
		envMap["AWS_ACCESS_KEY_ID"] = value.AccessKeyID
		envMap["AWS_SECRET_ACCESS_KEY"] = value.SecretAccessKey
		if value.SessionToken != "" {
			envMap["AWS_SESSION_TOKEN"] = value.SessionToken
		}
	}
	for k, v := range envMap {
		envs = append(envs, k+"="+v)
	}
	return envs
}

func hasCredentials(envMap map[string]string) bool {
	if _, ok := envMap["AWS_AWS_ACCESS_KEY_ID"]; ok {
		return true
	}
	return false
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
