package remotecommandhttpserver

import (
	"fmt"
	"net"
	"net/http"
)

type Server struct {
	listener     net.Listener
	config       *Config
	processCount int64
	count        uint64
	httpServer   *http.Server
}

var ErrCommandExited = fmt.Errorf("command exited")
var ErrCommandExitedWithError = fmt.Errorf("command exited with error")

func NewServer(addr string, configPath string) (*Server, error) {

	config, err := LoadConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}
	if config.Validate() != nil {
		return nil, fmt.Errorf("failed to validate config: %w", err)
	}

	s := &Server{
		config: config,
	}

	mux := http.NewServeMux()
	for i := range config.Cmds {
		cmd := &config.Cmds[i]
		mux.HandleFunc(
			cmd.Path,
			func(res http.ResponseWriter, req *http.Request) {
				s.ServeCall(res, req, cmd)
			},
		)
	}
	mux.Handle("/", http.HandlerFunc(s.List))

	s.httpServer = &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	return s, nil
}

func (s *Server) Start() error {
	listener, err := net.Listen("tcp", s.httpServer.Addr)
	if err != nil {
		return err
	}

	s.listener = listener

	go func() {
		err := s.httpServer.Serve(listener)
		if err == http.ErrServerClosed {
			panic(err)
		}
	}()

	return nil
}

func (s *Server) Shoutdown() error {
	return s.listener.Close()
}
