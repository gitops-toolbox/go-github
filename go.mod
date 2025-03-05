module github.com/gitops-toolbox/go-github

go 1.23.0

require (
	github.com/bradleyfalzon/ghinstallation/v2 v2.12.0
	github.com/google/go-github/v68 v68.0.0
	golang.org/x/crypto v0.31.0
	golang.org/x/oauth2 v0.24.0
	gopkg.in/yaml.v2 v2.4.0
)

require (
	github.com/golang-jwt/jwt/v4 v4.5.1 // indirect
	github.com/google/go-github/v66 v66.0.0 // indirect
	github.com/google/go-querystring v1.1.0 // indirect
)

replace github.com/bradleyfarzon/ghinstallation/v2 => github.com/gitops-toolbox/ghinstallation/v2 v2.12.0
