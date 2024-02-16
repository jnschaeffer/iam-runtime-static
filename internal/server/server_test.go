package server

import (
	"context"
	"os"
	"testing"

	"github.com/metal-toolbox/iam-runtime/pkg/iam/runtime/authentication"
	"github.com/metal-toolbox/iam-runtime/pkg/iam/runtime/authorization"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestServer(t *testing.T) {
	// Run everything in parallel
	t.Parallel()

	// Set up a server with a simple policy.
	subjectAlice := "alice"
	subjectBob := "bob"

	envVarAlice := "IAM_ALICE_TOKEN"
	envVarBob := "IAM_BOB_TOKEN"

	tokenAlice := "alic3!"
	tokenBob := "b0b"
	tokenDNE := "doesnotexist"

	actionGreet := "greet"

	resourceWorld := "resourc-world"

	// In this policy, Alice has the ability to greet the world, but Bob does not.
	authPolicy := policy{
		Subjects: []policySubject{
			{
				ID: subjectAlice,
				Tokens: []policyToken{
					{
						EnvVar: envVarAlice,
					},
				},
				Resources: []policyResource{
					{
						ID: resourceWorld,
						Actions: []string{
							actionGreet,
						},
					},
				},
			},
			{
				ID: subjectBob,
				Tokens: []policyToken{
					{
						EnvVar: envVarBob,
					},
				},
				Resources: []policyResource{},
			},
		},
	}

	// Set the environment variables for the test
	os.Setenv(envVarAlice, tokenAlice)
	os.Setenv(envVarBob, tokenBob)

	logger := zap.NewNop().Sugar()

	srv, err := newFromPolicy(authPolicy, logger)

	require.NoError(t, err)

	t.Run("ValidateCredentialSuccess", func(t *testing.T) {
		t.Parallel()

		req := &authentication.ValidateCredentialRequest{
			Credential: tokenAlice,
		}
		resp, err := srv.ValidateCredential(context.Background(), req)

		require.NoError(t, err)
		require.Equal(t, subjectAlice, resp.SubjectId)
	})

	t.Run("ValidateCredentialFail", func(t *testing.T) {
		t.Parallel()

		req := &authentication.ValidateCredentialRequest{
			Credential: tokenDNE,
		}
		_, err := srv.ValidateCredential(context.Background(), req)

		errStatus, ok := status.FromError(err)

		require.Equal(t, true, ok)
		require.Equal(t, codes.Unauthenticated, errStatus.Code())
	})

	t.Run("CheckAccessSuccess", func(t *testing.T) {
		t.Parallel()

		req := &authorization.CheckAccessRequest{
			Credential: tokenAlice,
			Actions: []*authorization.AccessRequestAction{
				{
					ResourceId: resourceWorld,
					Action:     actionGreet,
				},
			},
		}

		_, err := srv.CheckAccess(context.Background(), req)
		require.NoError(t, err)
	})

	t.Run("CheckAccessUnauthenticated", func(t *testing.T) {
		t.Parallel()

		req := &authorization.CheckAccessRequest{
			Credential: tokenDNE,
			Actions: []*authorization.AccessRequestAction{
				{
					ResourceId: resourceWorld,
					Action:     actionGreet,
				},
			},
		}

		_, err := srv.CheckAccess(context.Background(), req)

		errStatus, ok := status.FromError(err)

		require.Equal(t, true, ok)
		require.Equal(t, codes.Unauthenticated, errStatus.Code())
	})

	t.Run("CheckAccessUnauthorized", func(t *testing.T) {
		t.Parallel()

		req := &authorization.CheckAccessRequest{
			Credential: tokenBob,
			Actions: []*authorization.AccessRequestAction{
				{
					ResourceId: resourceWorld,
					Action:     actionGreet,
				},
			},
		}

		_, err := srv.CheckAccess(context.Background(), req)

		errStatus, ok := status.FromError(err)

		require.Equal(t, true, ok)
		require.Equal(t, codes.PermissionDenied, errStatus.Code())
	})
}
