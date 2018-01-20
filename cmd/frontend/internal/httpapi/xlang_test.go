package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/sourcegraph/jsonrpc2"
	"sourcegraph.com/sourcegraph/sourcegraph/cmd/frontend/internal/backend"
	sourcegraph "sourcegraph.com/sourcegraph/sourcegraph/pkg/api"
	"sourcegraph.com/sourcegraph/sourcegraph/pkg/api/legacyerr"
)

func TestXLang(t *testing.T) {
	c := newTest()

	calledValid := false
	calledUnauthed := false
	backend.Mocks.Repos.GetByURI = func(ctx context.Context, uri string) (*sourcegraph.Repo, error) {
		switch uri {
		case "my/repo":
			calledValid = true
			return &sourcegraph.Repo{ID: 1, URI: uri}, nil
		case "your/repo":
			calledUnauthed = true
			return nil, legacyerr.Errorf(legacyerr.Unauthenticated, "nope")
		default:
			t.Errorf("got unexpected repo %q", uri)
			return nil, legacyerr.Errorf(legacyerr.NotFound, "404")
		}
	}

	orig := xlangNewClient
	defer func() {
		xlangNewClient = orig
	}()
	var xc xlangTestClient
	xlangNewClient = func() (xlangClient, error) { return &xc, nil }

	postJSON := func(lspMethod string, h http.Header, reqBodyJSON string, respBody interface{}) error {
		req, err := http.NewRequest("POST", "/xlang/"+lspMethod, strings.NewReader(reqBodyJSON))
		if err != nil {
			return err
		}
		req.Header.Set("content-type", "application/json; charset=utf-8")
		for k, v := range h {
			req.Header[k] = v
		}
		resp, err := c.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("HTTP error status %d", resp.StatusCode)
		}
		if respBody != nil {
			if err := json.NewDecoder(resp.Body).Decode(&respBody); err != nil {
				return err
			}
		}
		return nil
	}

	// First try on a private repo we can't access
	if err := postJSON("someMethod", nil, `[{"id":0,"method":"initialize","params":{"rootUri":"git://your/repo?myrev"}},{"id":1,"method":"someMethod","params":{}},{"id":2,"method":"shutdown"},{"method":"exit"}]`, nil); err == nil {
		t.Error(err)
	}
	if calledValid {
		t.Error("calledValid")
	}
	if !calledUnauthed {
		t.Error("!calledUnauthed")
	}
	calledUnauthed = false

	if err := postJSON("someMethod", nil, `[{"id":0,"method":"initialize","params":{"rootUri":"git://my/repo?myrev"}},{"id":1,"method":"someMethod","params":{}},{"id":2,"method":"shutdown"},{"method":"exit"}]`, nil); err != nil {
		t.Fatal(err)
	}
	if want := []string{"initialize", "someMethod", "shutdown", "exit"}; !reflect.DeepEqual(xc.methodsCalled, want) {
		t.Errorf("got methods called == %v, want %v", xc.methodsCalled, want)
	}

	if !calledValid {
		t.Error("!calledValid")
	}
	if calledUnauthed {
		t.Error("calledUnauthed")
	}
}

type xlangTestClient struct{ methodsCalled []string }

func (c *xlangTestClient) Call(ctx context.Context, method string, params, result interface{}, opt ...jsonrpc2.CallOption) error {
	c.methodsCalled = append(c.methodsCalled, method)
	return nil
}

func (c *xlangTestClient) Notify(ctx context.Context, method string, params interface{}, opt ...jsonrpc2.CallOption) error {
	c.methodsCalled = append(c.methodsCalled, method)
	return nil
}

func (c *xlangTestClient) Close() error { return nil }
