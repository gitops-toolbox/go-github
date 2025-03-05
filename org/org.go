package org

import (
	"context"
	"encoding/base64"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strconv"

	"github.com/bradleyfalzon/ghinstallation/v2"
	"github.com/google/go-github/v68/github"
	"golang.org/x/oauth2"
)

const (
	APP_ID                 = "APP_ID"
	APP_INSTALLATION_ID    = "APP_INSTALLATION_ID"
	APP_PRIVATE_KEY_BASE64 = "APP_PRIVATE_KEY_BASE64"
	GITHUB_TOKEN           = "GITHUB_TOKEN"
)

type Org struct {
	client       *github.Client
	ctx          context.Context
	Organization string
	privateKey   *string
}

type RemoteTypes interface {
	github.User | github.Team | github.Repository
}

func new(ctx context.Context, client *github.Client, org string) Org {
	return Org{
		client:       client,
		ctx:          ctx,
		Organization: org,
	}
}

// GetClientFromGithubApp retrieves a GitHub client for a specified organization using GitHub App authentication.
// It requires the following environment variables to be set:
// - APP_ID: The GitHub App ID.
// - APP_INSTALLATION_ID: The GitHub App installation ID.
// - APP_PRIVATE_KEY_BASE64: The GitHub App private key in base64 encoding.
//
// Parameters:
// - ctx: The context for the request.
// - organization: The GitHub organization name.
//
// Returns:
// - Org: The organization object.
// - error: An error if the client could not be created or if any required environment variable is missing or invalid.
func GetClientFromGithubApp(ctx context.Context, organization string) (Org, error) {
	var org Org
	slog.Debug("Getting a github client from a github application")

	app_id, app_id_ok := os.LookupEnv(APP_ID)
	installation_id, installation_id_ok := os.LookupEnv(APP_INSTALLATION_ID)
	base64_private_key, private_key_ok := os.LookupEnv(APP_PRIVATE_KEY_BASE64)

	if !app_id_ok {
		slog.Error(fmt.Sprintf("no app id specified, please set the value on %s environment variable", APP_ID))
	}

	if !installation_id_ok {
		slog.Error(fmt.Sprintf("no app installation id specified, please set the value on %s environment variable", APP_INSTALLATION_ID))
	}

	if !private_key_ok {
		slog.Error(fmt.Sprintf("no app private key specified, please set the value on %s environment variable", APP_PRIVATE_KEY_BASE64))
	}

	if !app_id_ok || !installation_id_ok || !private_key_ok {
		return org, fmt.Errorf("missing values to authenticate using github application")
	}

	// Convert environment variable to appropriate format
	id, err := strconv.ParseInt(app_id, 10, 64)

	if err != nil {
		return org, fmt.Errorf("%s is not a valid int64: %v", APP_ID, err)
	}

	inst_id, err := strconv.ParseInt(installation_id, 10, 64)

	if err != nil {
		return org, fmt.Errorf("%s is not a valid int64: %v", APP_INSTALLATION_ID, err)
	}

	private_key, err := base64.StdEncoding.DecodeString(base64_private_key)

	if err != nil {
		return org, fmt.Errorf("%s cannot be deconded from base64: %v", APP_PRIVATE_KEY_BASE64, err)
	}

	itr, err := ghinstallation.New(http.DefaultTransport, id, inst_id, []byte(private_key))
	if err != nil {
		return org, fmt.Errorf("cannot initialize transport from app: %v", err)
	}

	return new(ctx, github.NewClient(&http.Client{Transport: itr}), organization), nil
}

// GetClientFromPAT retrieves a GitHub client for the specified organization using a Personal Access Token (PAT).
// The PAT is expected to be set in the environment variable specified by GITHUB_TOKEN.
//
// Parameters:
//   - ctx: The context for the request, used for cancellation and timeouts.
//   - organization: The GitHub organization for which the client is being created.
//
// Returns:
//   - Org: An instance of the Org struct representing the GitHub organization.
//   - error: An error if the PAT is not found in the environment variables or if there is an issue creating the client.
func GetClientFromPAT(ctx context.Context, organization string) (Org, error) {
	var org Org
	token, token_ok := os.LookupEnv(GITHUB_TOKEN)
	if !token_ok {
		slog.Error(fmt.Sprintf("no github token specified, please set the value on %s environment variable", GITHUB_TOKEN))
		return org, fmt.Errorf("missing values to authenticate using github application")
	}

	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)

	return new(ctx, github.NewClient(tc), organization), nil
}

func GetClient(organization string) (Org, error) {
	_, app_id_ok := os.LookupEnv(APP_INSTALLATION_ID)
	if app_id_ok {
		return GetClientFromGithubApp(context.Background(), organization)
	}
	return GetClientFromPAT(context.Background(), organization)
}

func (org Org) GetRepositories() ([]github.Repository, error) {
	opts := &github.RepositoryListByOrgOptions{
		ListOptions: github.ListOptions{
			PerPage: 10,
		},
	}

	return paginate[github.Repository](
		func(page int) ([]*github.Repository, *github.Response, error) {
			if page != -1 {
				opts.ListOptions.Page = page
			}
			return org.client.Repositories.ListByOrg(org.ctx, org.Organization, opts)
		},
	)
}

func (org Org) GetFileContent(repoName string, filepath string) (string, error) {
	content, _, _, err := org.client.Repositories.GetContents(org.ctx, org.Organization, repoName, filepath, &github.RepositoryContentGetOptions{})
	if err != nil {
		return "", err
	}
	return content.GetContent()
}

func (org Org) GetConfig() ([]github.Team, error) {
	return paginate[github.Team](
		func(page int) ([]*github.Team, *github.Response, error) {
			return org.client.Teams.ListTeams(org.ctx, org.Organization, &github.ListOptions{Page: page})
		},
	)
}

func (org Org) GetClient() *github.Client {
	return org.client
}

func paginate[remoteType RemoteTypes](getPage func(nextPage int) ([]*remoteType, *github.Response, error)) ([]remoteType, error) {
	result := []remoteType{}
	nextPage := -1

	for nextPage != 0 {
		entities, resp, err := getPage(nextPage)
		if err != nil {
			return result, err
		}
		for _, e := range entities {
			result = append(result, *e)
		}

		nextPage = resp.NextPage
	}

	return result, nil
}
