package internal

import (
	"net/http"

	"connectrpc.com/connect"
	"github.com/base/base-microservice/gen/user/userconnect"
	"github.com/base/base-microservice/pkg/actor"
	"github.com/base/base-microservice/pkg/obs"
)

type BackendURLs struct {
	UserURL string `kong:"name='backend-user-url',default='http://localhost:8001'"`
}

type Backends struct {
	User userconnect.UserServiceClient
}

func NewBackends(httpClient *http.Client, urls BackendURLs) *Backends {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	opts := connect.WithInterceptors(obs.ConnectInterceptor(), actor.ForwardInterceptor())
	return &Backends{
		User: userconnect.NewUserServiceClient(httpClient, urls.UserURL, opts),
	}
}
