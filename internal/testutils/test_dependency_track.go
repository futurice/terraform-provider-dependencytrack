package testutils

import (
	"context"
	"fmt"
	"os"

	dtrack "github.com/futurice/dependency-track-client-go"
	"github.com/google/uuid"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

const ExternalDependencyTrackEndpointEnvVarName = "TF_ACC_EXTERNAL_DEPENDENCY_TRACK_ENDPOINT"
const ExternalDependencyTrackApikeyEnvVarName = "TF_ACC_EXTERNAL_DEPENDENCY_TRACK_APIKEY"

const defaultDependencyTrackUser = "admin"
const defaultDependencyTrackPassword = "admin"

// TestDependencyTrack represents the Dependency-Track instance against which the tests will
// be run, and an API client to access it.
type TestDependencyTrack struct {
	Endpoint string
	ApiKey   string
	Client   *dtrack.Client

	container testcontainers.Container
}

// InitTestDependencyTrack is a utility function intended to be used within TestMain to
// initialize a TestDependencyTrack. Returns a TestDependencyTrack and a function to
// clean-up after the test.
func InitTestDependencyTrack() (testDependencyTrack *TestDependencyTrack, cleanup func()) {
	var err error
	testDependencyTrack, err = NewTestDependencyTrack()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Failed to initialize test: %v\n", err)
		os.Exit(1)
	}

	cleanup = func() {
		if err := testDependencyTrack.Close(); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "Failed to clean-up test: %v\n", err)
			// not calling os.Exit(1) from here to let the test report its results
		}
	}

	return
}

// NewTestDependencyTrack returns a TestDependencyTrack. Close it with Close() after the test.
//
// Supported environment variables:
//
//   - TF_ACC_EXTERNAL_DEPENDENCY_TRACK_ENDPOINT - an HTTP(S) URL to the Dependency-Track API server you
//     wish to use for testing. WARNING: the tests will affect its state at least while they run. Only use
//     a dedicated test instance for this.
//
//   - TF_ACC_EXTERNAL_DEPENDENCY_TRACK_APIKEY - the API key to the Dependency-Track API server defined by
//     TF_ACC_EXTERNAL_DEPENDENCY_TRACK_ENDPOINT. The API key should have all permissions. This variable must
//     be defined if TF_ACC_EXTERNAL_DEPENDENCY_TRACK_ENDPOINT is defined, and must not be defined otherwise.
//
// If TF_ACC_EXTERNAL_DEPENDENCY_TRACK_ENDPOINT is empty/unset an internal Dockerized Dependency-Track will be
// started and configured for the test, to be discarded after the test.
func NewTestDependencyTrack() (*TestDependencyTrack, error) {
	config, err := getTestDependencyTrackConfig()
	if err != nil {
		return nil, fmt.Errorf("invalid test Dependency-Track configuration: %w", err)
	}

	if config.isExternalDependencyTrackConfigured() {
		return newTestDependencyTrackFromExternalEndpoint(config)
	} else {
		return newTestDependencyTrackFromInternalContainer()
	}
}

// AddProviderConfiguration prepends a provider configuration to a piece of Terraform code (HCL) defining resources.
func (tdt *TestDependencyTrack) AddProviderConfiguration(terraformCode string) string {
	return fmt.Sprintf(`
provider "dependencytrack" {
  host    = %[1]q
  api_key = %[2]q
}

%s
`, tdt.Endpoint, tdt.ApiKey, terraformCode)
}

// Close disposes of the test Dependency-Track API server. If an internal Docker container was started, it will be discarded.
// In case an external API server was used it will not be affected by Close.
func (tdt *TestDependencyTrack) Close() error {
	if tdt.container != nil {
		err := stopDependencyTrackContainer(context.Background(), tdt.container)
		if err != nil {
			return err
		}
	}

	fmt.Printf("Test Dependency-Track closed")

	return nil
}

type testDependencyTrackConfig struct {
	endpoint string
	apiKey   string
}

func (c *testDependencyTrackConfig) validate() error {
	if (c.endpoint == "") != (c.apiKey == "") {
		return fmt.Errorf(
			"both %s and %s environment variables must be defined if either of them is",
			ExternalDependencyTrackEndpointEnvVarName,
			ExternalDependencyTrackApikeyEnvVarName,
		)
	}

	return nil
}

func (c *testDependencyTrackConfig) isExternalDependencyTrackConfigured() bool {
	return (c.endpoint != "") && (c.apiKey != "")
}

func getTestDependencyTrackConfig() (*testDependencyTrackConfig, error) {
	config := &testDependencyTrackConfig{
		endpoint: os.Getenv(ExternalDependencyTrackEndpointEnvVarName),
		apiKey:   os.Getenv(ExternalDependencyTrackApikeyEnvVarName),
	}

	if err := config.validate(); err != nil {
		return nil, err
	}

	return config, nil
}

func newTestDependencyTrackFromExternalEndpoint(config *testDependencyTrackConfig) (*TestDependencyTrack, error) {
	client, err := dtrack.NewClient(
		config.endpoint,
		dtrack.WithAPIKey(config.apiKey),
	)
	if err != nil {
		return nil, fmt.Errorf("could not create Dependency-Track client: %w", err)
	}

	fmt.Printf("Using extrernal Dependency-Track with endpoint: %s\n", config.endpoint)

	return &TestDependencyTrack{
		Endpoint:  config.endpoint,
		ApiKey:    config.apiKey,
		Client:    client,
		container: nil,
	}, nil
}

func newTestDependencyTrackFromInternalContainer() (*TestDependencyTrack, error) {
	ctx := context.Background()

	container, err := startDependencyTrackContainer(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not start Dependency-Track container: %w", err)
	}

	fmt.Printf("Configuring test Dependency-Track\n")

	testDependencyTrack, err := configureDependencyTrackContainer(ctx, container)
	if err != nil {
		stopErr := stopDependencyTrackContainer(ctx, container)
		if stopErr != nil {
			err = fmt.Errorf("%w (also failed to stop the container with error %v)", err, stopErr)
		}

		return nil, fmt.Errorf("could not configure Dependency-Track container: %w", err)
	}

	fmt.Printf("Inernal test Dependency-Track is ready\n")

	return testDependencyTrack, nil
}

func startDependencyTrackContainer(ctx context.Context) (testcontainers.Container, error) {
	containerRequest := testcontainers.ContainerRequest{
		Image:        "dependencytrack/apiserver:4.11.3",
		ExposedPorts: []string{"8080/tcp"},
		WaitingFor:   wait.ForLog("Dependency-Track is ready"),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: containerRequest,
		Started:          true,
	})

	return container, err
}

func stopDependencyTrackContainer(ctx context.Context, container testcontainers.Container) error {
	if err := container.Terminate(ctx); err != nil {
		return fmt.Errorf("could not stop Dependency-Track container: %w", err)
	}

	return nil
}

func configureDependencyTrackContainer(ctx context.Context, container testcontainers.Container) (*TestDependencyTrack, error) {
	containerEndpoint, err := container.Endpoint(ctx, "http")
	if err != nil {
		return nil, fmt.Errorf("could not get Dependency-Track container endpoint: %w", err)
	}

	fmt.Printf("Test Dependency-Track container endpoint is: %s\n", containerEndpoint)

	token, err := loginAsDefaultUser(ctx, containerEndpoint)
	if err != nil {
		return nil, fmt.Errorf("could not log in to Dependency-Track as default user: %w", err)
	}

	adminClient, err := dtrack.NewClient(
		containerEndpoint,
		dtrack.WithBearerToken(token),
	)
	if err != nil {
		return nil, fmt.Errorf("could not create Dependency-Track admin client: %w", err)
	}

	apiKey, err := createAllPowerfulApiKey(ctx, adminClient)
	if err != nil {
		return nil, fmt.Errorf("could not create Dependency-Track API key: %w", err)
	}

	return &TestDependencyTrack{
		Endpoint: containerEndpoint,
		ApiKey:   apiKey,
		Client:   adminClient,

		container: container,
	}, nil
}

func loginAsDefaultUser(ctx context.Context, endpoint string) (token string, err error) {
	bootstrapClient, err := dtrack.NewClient(endpoint)
	if err != nil {
		return "", fmt.Errorf("could not create Dependency-Track bootstrap client: %w", err)
	}

	// Using a random UUID as the password to avoid random string generation via other means
	//   This password does not need to have production-grade strength
	adminPassword := uuid.New().String()

	err = bootstrapClient.User.ForceChangePassword(ctx, defaultDependencyTrackUser, defaultDependencyTrackPassword, adminPassword)
	if err != nil {
		return "", fmt.Errorf("could not change Dependency-Track %s user password: %w", defaultDependencyTrackUser, err)
	}

	fmt.Printf("Reset password of user %s\n", defaultDependencyTrackUser)

	token, err = bootstrapClient.User.Login(ctx, defaultDependencyTrackUser, adminPassword)
	if err != nil {
		return "", fmt.Errorf("could not log in to Dependency-Track: %w", err)
	}

	fmt.Printf("Logged in as user %s\n", defaultDependencyTrackUser)

	return
}

func createAllPowerfulApiKey(ctx context.Context, client *dtrack.Client) (apiKey string, err error) {
	teamName := "test"

	team, err := client.Team.Create(ctx, dtrack.Team{
		Name: teamName,
	})
	if err != nil {
		return "", fmt.Errorf("could not create Dependency-Track test team: %w", err)
	}

	fmt.Printf("Created team %s\n", teamName)

	// all signs point to paging options being ignored by this endpoint
	allPermissions, err := client.Permission.GetAll(ctx, dtrack.PageOptions{})
	if err != nil {
		return "", fmt.Errorf("could not get Dependency-Track permissions: %w", err)
	}

	for _, permission := range allPermissions.Items {
		_, err = client.Permission.AddPermissionToTeam(ctx, permission, team.UUID)
		if err != nil {
			return "", fmt.Errorf("could not grant permission %s to Dependency-Track test team: %w", permission.Name, err)
		}
	}

	fmt.Printf("Granted all %d permissions to team %s\n", len(allPermissions.Items), teamName)

	apiKey, err = client.Team.GenerateAPIKey(ctx, team.UUID)
	if err != nil {
		return "", fmt.Errorf("could not create Dependency-Track API key for test team: %w", err)
	}

	fmt.Printf("Created an API key for team %s\n", teamName)

	return
}
