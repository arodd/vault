package alicloud

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/vault/api"
	"github.com/hashicorp/vault/command/agent/auth"
)

func TestNewAliCloudAuthMethod(t *testing.T) {

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(secret))
		w.WriteHeader(200)
	}))
	defer ts.Close()

	clientConfig := api.DefaultConfig()
	clientConfig.Address = ts.URL

	apiClient, err := api.NewClient(clientConfig)
	if err != nil {
		t.Fatal(err)
	}

	logger := hclog.Default()
	logger.SetLevel(hclog.Trace)
	authHandlerConfig := &auth.AuthHandlerConfig{
		Logger: logger,
		Client: apiClient,
	}
	authHandler := auth.NewAuthHandler(authHandlerConfig)

	config := map[string]interface{}{
		"role":          "web-workers",
		"region":        "us-west-1",
		"access_key":    "some-access-key",
		"access_secret": "some-access-secret",
	}
	authConfig := &auth.AuthConfig{
		Logger:    logger,
		MountPath: "alicloud",
		Config:    config,
	}
	authMethod, err := NewAliCloudAuthMethod(authConfig)
	if err != nil {
		t.Fatal(err)
	}

	// We need to Run in a different goroutine because if we don't,
	// it'll block until we read from the output channel.
	go func() {
		ctx, cancelFunc := context.WithTimeout(context.Background(), time.Second*10)
		authHandler.Run(ctx, authMethod)
		cancelFunc()
	}()
	receivedClientToken := false
	for clientToken := range authHandler.OutputCh {
		if clientToken != "client-token" {
			t.Fatalf("expected client-token but received %s", clientToken)
		}
		receivedClientToken = true
	}
	if !receivedClientToken {
		t.Fatal("should have received at least one client token")
	}
}

const secret = `
{
	"lease_id": "foo",
	"renewable": true,
	"lease_duration": 10,
	"data": {
		"key": "value"
	},
	"warnings": [
		"a warning!"
	],
	"wrap_info": {
		"token": "token",
		"accessor": "accessor",
		"ttl": 60,
		"creation_time": "2016-06-07T15:52:10-04:00",
		"wrapped_accessor": "abcd1234"
	},
	"auth": {
		"client_token": "client-token",
		"renewable": true
	}
}`