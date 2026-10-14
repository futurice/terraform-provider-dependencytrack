// Copyright (c) 2024 Futurice Oy
// SPDX-License-Identifier: MPL-2.0

package testutils

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	dtrack "github.com/futurice/dependency-track-client-go"
	"github.com/futurice/terraform-provider-dependencytrack/internal/utils"
	"github.com/google/uuid"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/network"
	"github.com/testcontainers/testcontainers-go/wait"
)

const ExternalDependencyTrackEndpointEnvVarName = "TF_ACC_EXTERNAL_DEPENDENCY_TRACK_ENDPOINT"
const ExternalDependencyTrackApikeyEnvVarName = "TF_ACC_EXTERNAL_DEPENDENCY_TRACK_APIKEY" //nolint:gosec // This is just an environment variable name, not actual credentials
const ShowDependencyTrackContainerLogsEnvVarName = "TF_ACC_SHOW_DEPENDENCY_TRACK_CONTAINER_LOGS"
const DependencyTrackVersionEnvVarName = "TF_ACC_DEPENDENCY_TRACK_VERSION"

const defaultDependencyTrackVersion = "4.14.5"

const defaultDependencyTrackUser = "admin"
const defaultDependencyTrackPassword = "admin"

// TestDependencyTrack represents the Dependency-Track instance against which the tests will
// be run, and an API client to access it.
type TestDependencyTrack struct {
	Endpoint string
	APIKey   string
	Client   *dtrack.Client

	v5        bool
	config    *testDependencyTrackConfig
	container *dependencyTrackContainer
}

type testDependencyTrackConfig struct {
	endpoint          string
	apiKey            string
	showContainerLogs bool
	version           string
}

// dependencyTrackContainer is the API server container, plus the PostgreSQL container and network that v5 needs.
type dependencyTrackContainer struct {
	apiServer testcontainers.Container
	postgres  testcontainers.Container
	network   *testcontainers.DockerNetwork
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
//   - TF_ACC_DEPENDENCY_TRACK_VERSION - the apiserver image tag to start, 4.14.5 by default; 5+ also starts PostgreSQL.
//
//   - TF_ACC_SHOW_DEPENDENCY_TRACK_CONTAINER_LOGS - if set to a non-empty value will cause the
//     Dependency-Track API server container logs to be printed after each test suite run. Ignored if an extenal
//     endpoint is configured.
//
// If TF_ACC_EXTERNAL_DEPENDENCY_TRACK_ENDPOINT is empty/unset an internal Dockerized Dependency-Track will be
// started and configured for the test, to be discarded after the test.
func NewTestDependencyTrack() (*TestDependencyTrack, error) {
	config, err := getTestDependencyTrackConfig()
	if err != nil {
		return nil, fmt.Errorf("invalid test Dependency-Track configuration: %w", err)
	}

	var tdt *TestDependencyTrack
	if config.isExternalDependencyTrackConfigured() {
		tdt, err = newTestDependencyTrackFromExternalEndpoint(config)
	} else {
		tdt, err = newTestDependencyTrackFromInternalContainer(config)
	}
	if err != nil {
		return nil, err
	}

	version, err := tdt.Client.ServerVersion(context.Background())
	if err != nil {
		closeErr := tdt.Close()
		return nil, errors.Join(fmt.Errorf("could not read Dependency-Track version: %w", err), closeErr)
	}
	tdt.v5, err = utils.IsV5Version(version)
	if err != nil {
		closeErr := tdt.Close()
		return nil, errors.Join(err, closeErr)
	}

	fmt.Printf("Test Dependency-Track version is %s\n", version)

	return tdt, nil
}

// IsV5 reports whether the test Dependency-Track runs version 5 or later.
func (tdt *TestDependencyTrack) IsV5() bool {
	return tdt.v5
}

// PublisherKindAttribute returns the HCL line that picks a built-in publisher such as "slack", per server version.
func (tdt *TestDependencyTrack) PublisherKindAttribute(extension string) string {
	if tdt.v5 {
		return fmt.Sprintf("extension_name = %q", extension)
	}
	return fmt.Sprintf("publisher_class = %q", PublisherClass(extension))
}

// WithPublisherKind sets PublisherClass on v4 or ExtensionName on v5, to build an expected publisher.
func (tdt *TestDependencyTrack) WithPublisherKind(publisher dtrack.NotificationPublisher, extension string) dtrack.NotificationPublisher {
	if tdt.v5 {
		publisher.ExtensionName = extension
	} else {
		publisher.PublisherClass = PublisherClass(extension)
	}
	return publisher
}

// PublisherClass maps a v5 extension name such as "slack" to its v4 class.
func PublisherClass(extension string) string {
	return "org.dependencytrack.notification.publisher." + strings.ToUpper(extension[:1]) + extension[1:] + "Publisher"
}

// AddProviderConfiguration prepends a provider configuration to a piece of Terraform code (HCL) defining resources.
func (tdt *TestDependencyTrack) AddProviderConfiguration(terraformCode string) string {
	return fmt.Sprintf(`
provider "dependencytrack" {
  host    = %[1]q
  api_key = %[2]q
}

%s
`, tdt.Endpoint, tdt.APIKey, terraformCode)
}

// Close disposes of the test Dependency-Track API server. If an internal Docker container was started, it will be discarded.
// In case an external API server was used it will not be affected by Close.
func (tdt *TestDependencyTrack) Close() error {
	if tdt.container != nil {
		err := stopDependencyTrackContainer(context.Background(), tdt.config, tdt.container)
		if err != nil {
			return err
		}
	}

	fmt.Printf("Test Dependency-Track closed")

	return nil
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
		endpoint:          os.Getenv(ExternalDependencyTrackEndpointEnvVarName),
		apiKey:            os.Getenv(ExternalDependencyTrackApikeyEnvVarName),
		showContainerLogs: os.Getenv(ShowDependencyTrackContainerLogsEnvVarName) != "",
		version:           os.Getenv(DependencyTrackVersionEnvVarName),
	}

	if config.version == "" {
		config.version = defaultDependencyTrackVersion
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
		Endpoint: config.endpoint,
		APIKey:   config.apiKey,
		Client:   client,

		config:    config,
		container: nil,
	}, nil
}

func newTestDependencyTrackFromInternalContainer(config *testDependencyTrackConfig) (*TestDependencyTrack, error) {
	ctx := context.Background()

	container, err := startDependencyTrackContainer(ctx, config.version)
	if err != nil {
		return nil, fmt.Errorf("could not start Dependency-Track container: %w", err)
	}

	fmt.Printf("Configuring test Dependency-Track\n")

	testDependencyTrack, err := configureDependencyTrackContainer(ctx, config, container)
	if err != nil {
		stopErr := stopDependencyTrackContainer(ctx, config, container)
		if stopErr != nil {
			err = fmt.Errorf("%w (also failed to stop the container with error %w)", err, stopErr)
		}

		return nil, fmt.Errorf("could not configure Dependency-Track container: %w", err)
	}

	fmt.Printf("Inernal test Dependency-Track is ready\n")

	return testDependencyTrack, nil
}

func startDependencyTrackContainer(ctx context.Context, version string) (*dependencyTrackContainer, error) {
	v5, err := utils.IsV5Version(version)
	if err != nil {
		return nil, fmt.Errorf("%s must be a release version such as %s: %w", DependencyTrackVersionEnvVarName, defaultDependencyTrackVersion, err)
	}
	if v5 {
		return startDependencyTrackV5Container(ctx, version)
	}

	containerRequest := testcontainers.ContainerRequest{
		Image:        "dependencytrack/apiserver:" + version,
		ExposedPorts: []string{"8080/tcp"},
		WaitingFor:   wait.ForLog("Dependency-Track is ready").WithStartupTimeout(2 * time.Minute),
	}

	apiServer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: containerRequest,
		Started:          true,
	})
	if err != nil {
		return nil, err
	}

	return &dependencyTrackContainer{apiServer: apiServer}, nil
}

// startDependencyTrackV5Container mirrors the Helm chart's env, since v5 has no embedded database and no "ready" log line.
func startDependencyTrackV5Container(ctx context.Context, version string) (container *dependencyTrackContainer, err error) {
	container = &dependencyTrackContainer{}
	defer func() {
		if err != nil {
			if termErr := container.terminate(ctx); termErr != nil {
				err = fmt.Errorf("%w (also failed to clean up with error %w)", err, termErr)
			}
			container = nil
		}
	}()

	container.network, err = network.New(ctx)
	if err != nil {
		return container, fmt.Errorf("could not create Docker network: %w", err)
	}

	container.postgres, err = testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:          "postgres:16-alpine",
			Networks:       []string{container.network.Name},
			NetworkAliases: map[string][]string{container.network.Name: {"postgres"}},
			Env: map[string]string{
				"POSTGRES_DB":       "dtrack",
				"POSTGRES_USER":     "dtrack",
				"POSTGRES_PASSWORD": "dtrack",
			},
			// The server logs this once during init and once when it's up for real.
			WaitingFor: wait.ForLog("database system is ready to accept connections").WithOccurrence(2).WithStartupTimeout(time.Minute),
		},
		Started: true,
	})
	if err != nil {
		return container, fmt.Errorf("could not start PostgreSQL container: %w", err)
	}

	container.apiServer, err = testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "dependencytrack/apiserver:" + version,
			ExposedPorts: []string{"8080/tcp", "9000/tcp"},
			Networks:     []string{container.network.Name},
			Env: map[string]string{ //nolint:gosec // Throwaway test container credentials, not real secrets
				"DT_DATASOURCE_URL":             "jdbc:postgresql://postgres:5432/dtrack?reWriteBatchedInserts=true",
				"DT_DATASOURCE_USERNAME":        "dtrack",
				"DT_DATASOURCE_PASSWORD":        "dtrack",
				"DT_SECRET_MANAGEMENT_PROVIDER": "database",
				// Fixed test fixture, base64 of the 32 bytes "dt-v5-acctest-fixture-not-a-real".
				"DT_SECRET_MANAGEMENT_DATABASE_KEK":       "ZHQtdjUtYWNjdGVzdC1maXh0dXJlLW5vdC1hLXJlYWw=",
				"DT_FILE_STORAGE_PROVIDER":                "local",
				"DT_FILE_STORAGE_LOCAL_DIRECTORY":         "/data/storage",
				"DT_MANAGEMENT_PORT":                      "9000",
				"DT_TELEMETRY_SUBMISSION_DEFAULT_ENABLED": "false",
			},
			// /health answers 200 even when the API failed to start, so wait on the API port too.
			WaitingFor: wait.ForAll(
				wait.ForHTTP("/health").WithPort("9000/tcp").WithStartupTimeout(3*time.Minute),
				wait.ForHTTP("/api/version").WithPort("8080/tcp").WithStartupTimeout(3*time.Minute),
			),
		},
		Started: true,
	})
	if err != nil {
		return container, fmt.Errorf("could not start Dependency-Track API server container: %w", err)
	}

	return container, nil
}

func (c *dependencyTrackContainer) terminate(ctx context.Context) error {
	var errs []error
	for _, tc := range []testcontainers.Container{c.apiServer, c.postgres} {
		if tc != nil {
			if err := tc.Terminate(ctx); err != nil {
				errs = append(errs, err)
			}
		}
	}
	if c.network != nil {
		if err := c.network.Remove(ctx); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func stopDependencyTrackContainer(ctx context.Context, config *testDependencyTrackConfig, container *dependencyTrackContainer) error {
	if config.showContainerLogs {
		if err := showDependencyTrackContainerLogs(ctx, container.apiServer); err != nil {
			return fmt.Errorf("could not show Dependency-Track container logs: %w", err)
		}
	}

	if err := container.terminate(ctx); err != nil {
		return fmt.Errorf("could not stop Dependency-Track container: %w", err)
	}

	return nil
}

func showDependencyTrackContainerLogs(ctx context.Context, container testcontainers.Container) error {
	logReader, err := container.Logs(ctx)
	if err != nil {
		return fmt.Errorf("could not get Dependency-Track container logs from Docker: %w", err)
	}
	defer logReader.Close()

	fmt.Printf("********* Dependency-Track container logs\n")

	_, err = io.Copy(os.Stdout, logReader)
	if err != nil {
		return fmt.Errorf("could not copy Dependency-Track container logs to stdout: %w", err)
	}

	fmt.Printf("********* End of Dependency-Track container logs\n")

	return nil
}

func configureDependencyTrackContainer(ctx context.Context, config *testDependencyTrackConfig, container *dependencyTrackContainer) (*TestDependencyTrack, error) {
	containerEndpoint, err := container.apiServer.PortEndpoint(ctx, "8080/tcp", "http")
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

	apiKey, err := createAllPowerfulAPIKey(ctx, adminClient)
	if err != nil {
		return nil, fmt.Errorf("could not create Dependency-Track API key: %w", err)
	}

	return &TestDependencyTrack{
		Endpoint: containerEndpoint,
		APIKey:   apiKey.Key,
		Client:   adminClient,

		config:    config,
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

func createAllPowerfulAPIKey(ctx context.Context, client *dtrack.Client) (apiKey dtrack.APIKey, err error) {
	teamName := "test"

	team, err := client.Team.Create(ctx, dtrack.Team{
		Name: teamName,
	})
	if err != nil {
		return dtrack.APIKey{}, fmt.Errorf("could not create Dependency-Track test team: %w", err)
	}

	fmt.Printf("Created team %s\n", teamName)

	// all signs point to paging options being ignored by this endpoint
	allPermissions, err := client.Permission.GetAll(ctx, dtrack.PageOptions{})
	if err != nil {
		return dtrack.APIKey{}, fmt.Errorf("could not get Dependency-Track permissions: %w", err)
	}

	for _, permission := range allPermissions.Items {
		_, err = client.Permission.AddPermissionToTeam(ctx, permission, team.UUID)
		if err != nil {
			return dtrack.APIKey{}, fmt.Errorf("could not grant permission %s to Dependency-Track test team: %w", permission.Name, err)
		}
	}

	fmt.Printf("Granted all %d permissions to team %s\n", len(allPermissions.Items), teamName)

	apiKey, err = client.Team.GenerateAPIKey(ctx, team.UUID)
	if err != nil {
		return dtrack.APIKey{}, fmt.Errorf("could not create Dependency-Track API key for test team: %w", err)
	}

	fmt.Printf("Created an API key for team %s\n", teamName)

	return apiKey, err
}

// SkipUnlessV5 skips a test for a resource that only exists on Dependency-Track v5.
func (tdt *TestDependencyTrack) SkipUnlessV5(t *testing.T) {
	t.Helper()
	if !tdt.v5 {
		t.Skip("requires Dependency-Track v5")
	}
}
