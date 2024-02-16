package cmd

import (
	"context"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/metal-toolbox/iam-runtime-static/internal/server"

	"github.com/metal-toolbox/iam-runtime/pkg/iam/runtime/authentication"
	"github.com/metal-toolbox/iam-runtime/pkg/iam/runtime/authorization"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
)

// serveCmd starts the TODO service
var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "starts the iam-runtime-static service",
	RunE: func(cmd *cobra.Command, _ []string) error {
		return serve(cmd.Context(), viper.GetViper())
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)

	serveCmd.Flags().String("listen", "/var/"+appName+"/runtime.sock", "address to listen on")
	viperBindFlag("listen", serveCmd.Flags().Lookup("listen"))

	// App specific flags
	serveCmd.Flags().String("policy", "/etc/"+appName+"/policy.yaml", "runtime policy file")
	viperBindFlag("policy", serveCmd.Flags().Lookup("policy"))
}

func serve(_ context.Context, v *viper.Viper) error {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	policyPath := v.GetString("policy")
	socketPath := v.GetString("listen")

	if _, err := os.Stat(socketPath); err == nil {
		logger.Warnw("socket found, unlinking", "socket_path", socketPath)

		if err := syscall.Unlink(socketPath); err != nil {
			logger.Fatalw("error unlinking socket", "error", err)
		}
	}

	iamSrv, err := server.NewServer(policyPath, logger)
	if err != nil {
		logger.Fatalw("failed to create server", "error", err)
	}

	grpcSrv := grpc.NewServer()
	authorization.RegisterAuthorizationServer(grpcSrv, iamSrv)
	authentication.RegisterAuthenticationServer(grpcSrv, iamSrv)

	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		logger.Fatalw("failed to listen", "error", err)
	}

	logger.Infow("starting server",
		"address", socketPath,
	)

	go func() {
		if err := grpcSrv.Serve(listener); err != nil {
			logger.Fatalw("failed starting server", "error", err)
		}
	}()

	<-c

	logger.Info("signal received, stopping server")

	grpcSrv.GracefulStop()

	return nil
}
