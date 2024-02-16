package server

import (
	"context"
	"fmt"
	"os"

	"github.com/metal-toolbox/iam-runtime/pkg/iam/runtime/authentication"
	"github.com/metal-toolbox/iam-runtime/pkg/iam/runtime/authorization"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func checkAccess(sub policySubject, action, resourceID string) bool {
	var (
		resource policyResource
		found    bool
	)

	for _, candidate := range sub.Resources {
		if candidate.ID == resourceID {
			resource = candidate
			found = true
		}
	}

	if !found {
		return false
	}

	for _, candidate := range resource.Actions {
		if candidate == action {
			return true
		}
	}

	return false
}

// Server represents an IAM runtime server.
type Server interface {
	authentication.AuthenticationServer
	authorization.AuthorizationServer
}

type server struct {
	// Map from tokens to subjects
	tokens map[string]policySubject

	logger *zap.SugaredLogger

	authentication.UnimplementedAuthenticationServer
	authorization.UnimplementedAuthorizationServer
}

// NewServer creates a new static runtime server.
func NewServer(policyPath string, logger *zap.SugaredLogger) (Server, error) {
	f, err := os.Open(policyPath)
	if err != nil {
		return nil, err
	}

	defer f.Close()

	policy, err := readPolicy(f)
	if err != nil {
		return nil, err
	}

	return newFromPolicy(policy, logger)
}

func newFromPolicy(c policy, logger *zap.SugaredLogger) (*server, error) {
	tokens := make(map[string]policySubject)

	for _, sub := range c.Subjects {
		for _, tok := range sub.Tokens {
			tokValue := os.Getenv(tok.EnvVar)
			if tokValue == "" {
				err := fmt.Errorf("%s: %s: %w", sub.ID, tok.EnvVar, ErrMissingValue)
				return nil, err
			}

			if _, ok := tokens[tokValue]; ok {
				err := fmt.Errorf("%s: %s: %w", sub.ID, tok.EnvVar, ErrDuplicateValue)
				return nil, err
			}

			tokens[tokValue] = sub
		}
	}

	out := &server{
		tokens: tokens,
		logger: logger,
	}

	return out, nil
}

func (s *server) ValidateCredential(_ context.Context, req *authentication.ValidateCredentialRequest) (*authentication.ValidateCredentialResponse, error) {
	s.logger.Info("received ValidateCredential request")

	sub, ok := s.tokens[req.Credential]
	if !ok {
		return nil, status.Errorf(codes.Unauthenticated, "invalid credential")
	}

	resp := &authentication.ValidateCredentialResponse{
		SubjectId: sub.ID,
	}

	return resp, nil
}

func (s *server) CheckAccess(_ context.Context, req *authorization.CheckAccessRequest) (*authorization.CheckAccessResponse, error) {
	s.logger.Info("received CheckAccess request")

	sub, ok := s.tokens[req.Credential]
	if !ok {
		return nil, status.Errorf(codes.Unauthenticated, "invalid credential")
	}

	for _, action := range req.Actions {
		if ok := checkAccess(sub, action.Action, action.ResourceId); !ok {
			return nil, status.Errorf(codes.PermissionDenied, "subject does not have permission to perform '%s' on resource '%s'", action.Action, action.ResourceId)
		}
	}

	return &authorization.CheckAccessResponse{}, nil
}
