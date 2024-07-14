package remotecommandhttpserver

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"sync/atomic"
)

func buildEnv(cmd *CmdConfig) ([]string, error) {
	envMap := make(map[string]string, len(os.Environ()))

	for _, env := range os.Environ() {
		pair := strings.SplitN(env, "=", 2)
		envMap[pair[0]] = pair[1]
	}

	for key, value := range cmd.Envs {
		envMap[key] = value

	}

	if cmd.EnvFile != "" {

		f, err := os.Open(cmd.EnvFile)
		if err != nil {
			return nil, fmt.Errorf("cannot open env file: %w", err)
		}
		defer f.Close()

		reader := bufio.NewReader(f)
		for {
			l, _, err := reader.ReadLine()
			if err != nil {
				break
			}

			pair := strings.SplitN(string(l), "=", 2)
			if len(pair) != 2 {
				continue
			}

			envMap[pair[0]] = pair[1]
		}
	}

	env := make([]string, 0, len(envMap))
	for key, value := range envMap {
		env = append(env, fmt.Sprintf("%s=%s", key, value))
	}

	return env, nil
}

func buildCwd(cmd *CmdConfig) (string, error) {
	if len(cmd.Cwd) == 0 {
		cwd, err := os.Getwd()
		if err != nil {
			return "", err
		}
		return cwd, nil
	}

	stat, err := os.Stat(cmd.Cwd)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("cwd not found: %s", cmd.Cwd)
		}
	}
	if !stat.IsDir() {
		return "", fmt.Errorf("cwd is not directory: %s", cmd.Cwd)
	}

	return cmd.Cwd, nil
}

func (s *Server) ServeCall(res http.ResponseWriter, req *http.Request, cmd *CmdConfig) {
	no := atomic.AddUint64(&s.count, 1)
	log.Printf("[%8d]Request: %s", no, req.URL.Path)

	currentCount := atomic.AddInt64(&s.processCount, 1)
	defer func() {
		atomic.AddInt64(&s.processCount, -1)
	}()

	if currentCount > int64(s.config.MaxConcurrency) {
		log.Printf("[%8d]too many processes: %d\n", no, currentCount)
		res.WriteHeader(http.StatusTooManyRequests)
		return
	}

	ctx := req.Context()
	ctx, cancel := context.WithCancelCause(ctx)

	env, err := buildEnv(cmd)
	if err != nil {
		log.Printf("[%8d]failed to load env: %v\n", no, err)
		res.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(res, "error occured with start command\r\n")
		return
	}

	cwd, err := buildCwd(cmd)
	if err != nil {
		log.Printf("[%8d]failed to load cwd: %v\n", no, err)
		res.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(res, "error occured with start command: %v\r\n", err)
		return
	}

	rawCmd := exec.Command(cmd.Cmd[0], cmd.Cmd[1:]...)
	rawCmd.Env = env
	rawCmd.Dir = cwd
	stdout, _ := rawCmd.StdoutPipe()
	stderr, _ := rawCmd.StderrPipe()

	if err := rawCmd.Start(); err != nil {
		log.Printf("[%8d]failed to start command: %v\n", no, err)
		res.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(res, "error occured with start command\r\n")
		return
	}
	defer func() {
		// 終了していなければkillしておく
		if !rawCmd.ProcessState.Exited() {
			_ = rawCmd.Process.Kill()
		}
	}()

	res.WriteHeader(http.StatusOK)
	resFlusher := res.(http.Flusher)

	resWriteLock := &sync.Mutex{}
	wait := sync.WaitGroup{}
	readCmdOutput := func(out io.ReadCloser) {
		defer func() {
			wait.Done()
		}()
		reader := bufio.NewReader(out)
		for {
			line, _, err := reader.ReadLine()
			if err != nil {
				return
			}
			resWriteLock.Lock()
			log.Printf("[%8d]output: %s\n", no, line)
			_, _ = res.Write(line)
			_, _ = res.Write([]byte("\n"))
			resFlusher.Flush()
			resWriteLock.Unlock()
		}
	}

	wait.Add(2)

	go readCmdOutput(stdout)
	go readCmdOutput(stderr)

	go func() {
		err := rawCmd.Wait()
		if err != nil {
			log.Printf("[%8d]process exited: %v\n", no, err)
		}
		cancel(ErrCommandExited)
	}()

	<-ctx.Done()

	// すべて書き出すのを待つ
	wait.Wait()

	exitedErr := context.Cause(ctx)
	if exitedErr == ErrCommandExitedWithError {
		exitCode := rawCmd.ProcessState.ExitCode()
		log.Printf("[%8d]process exited status:%d\n", no, exitCode)
		fmt.Fprintf(res, "Exit with error: %d\n", exitCode)
	}
	if exitedErr == ErrCommandExited {
		log.Printf("[%8d]request completed\n", no)
	}
	if exitedErr == context.Canceled {
		log.Printf("[%8d]connection canceled\n", no)
	}
}

func (s *Server) List(res http.ResponseWriter, req *http.Request) {
	res.WriteHeader(http.StatusOK)
	for _, cmd := range s.config.Cmds {
		fmt.Fprintf(res, "%s\n", cmd.Path)
	}
}
