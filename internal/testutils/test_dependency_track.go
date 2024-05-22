package testutils

import (
	"context"
	"fmt"

	dtrack "github.com/futurice/dependency-track-client-go"
	"github.com/google/uuid"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

const defaultDependencyTrackUser = "admin"
const defaultDependencyTrackPassword = "admin"

type TestDependencyTrack struct {
	Endpoint string
	ApiKey   string
	Client   *dtrack.Client

	container testcontainers.Container
}

func NewTestDependencyTrack() (*TestDependencyTrack, error) {
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

	fmt.Printf("Test Dependency-Track is ready\n")

	return testDependencyTrack, nil
}

func (tdt *TestDependencyTrack) AddProviderConfiguration(terraformCode string) string {
	return fmt.Sprintf(`
provider "dependencytrack" {
  host    = %[1]q
  api_key = %[2]q
}

%s
`, tdt.Endpoint, tdt.ApiKey, terraformCode)
}

func (tdt *TestDependencyTrack) Close() error {
	err := stopDependencyTrackContainer(context.Background(), tdt.container)
	if err != nil {
		return err
	}

	fmt.Printf("Test Dependency-Track closed")

	return nil
}

func startDependencyTrackContainer(ctx context.Context) (testcontainers.Container, error) {
	containerRequest := testcontainers.ContainerRequest{
		Image:        "dependencytrack/apiserver:4.11.1",
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

	fmt.Printf("Test Dependency-Track endpoint is: %s\n", containerEndpoint)

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

	// FIXME try to improve this magic number
	allPermissions, err := client.Permission.GetAll(ctx, dtrack.PageOptions{PageSize: 1000})
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
