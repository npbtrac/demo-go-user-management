package httpserver_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	httpserver "github.com/npbtrac/demo-go-user-management/internal/http"
	"github.com/npbtrac/demo-go-user-management/internal/user"
)

func TestInternalCRUD(t *testing.T) {
	svc := user.NewService(user.NewFakeRepository())
	ts := httptest.NewServer(httpserver.InternalMux(svc))
	defer ts.Close()

	body := []byte(`{"username":"alice","email":"alice@example.com","phone":"+1","params":{"k":"v"}}`)
	resp, err := http.Post(ts.URL+"/internal/v1/users", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create status %d", resp.StatusCode)
	}
	var created map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		t.Fatal(err)
	}
	id, _ := created["id"].(string)
	if id == "" {
		t.Fatal("missing id")
	}

	resp, err = http.Get(ts.URL + "/internal/v1/users/" + id)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get %d", resp.StatusCode)
	}

	resp, err = http.Get(ts.URL + "/internal/v1/users")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("list %d", resp.StatusCode)
	}

	req, _ := http.NewRequest(http.MethodPatch, ts.URL+"/internal/v1/users/"+id, bytes.NewReader([]byte(`{"username":"alice2"}`)))
	req.Header.Set("Content-Type", "application/json")
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("patch %d", resp.StatusCode)
	}

	req, _ = http.NewRequest(http.MethodPut, ts.URL+"/internal/v1/users/"+id, bytes.NewReader([]byte(`{"username":"alice3","email":"alice3@example.com"}`)))
	req.Header.Set("Content-Type", "application/json")
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("put %d", resp.StatusCode)
	}

	req, _ = http.NewRequest(http.MethodDelete, ts.URL+"/internal/v1/users/"+id, nil)
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("delete %d", resp.StatusCode)
	}
}

func TestInternalErrors(t *testing.T) {
	svc := user.NewService(user.NewFakeRepository())
	ts := httptest.NewServer(httpserver.InternalMux(svc))
	defer ts.Close()

	resp, err := http.Post(ts.URL+"/internal/v1/users", "application/json", bytes.NewReader([]byte(`{"username":"a","email":"bad"}`)))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("invalid %d", resp.StatusCode)
	}

	ok := []byte(`{"username":"a","email":"a@example.com"}`)
	resp, err = http.Post(ts.URL+"/internal/v1/users", "application/json", bytes.NewReader(ok))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	resp, err = http.Post(ts.URL+"/internal/v1/users", "application/json", bytes.NewReader(ok))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("conflict %d", resp.StatusCode)
	}

	resp, err = http.Get(ts.URL + "/internal/v1/users/00000000-0000-0000-0000-000000000001")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("not found %d", resp.StatusCode)
	}
}

func TestPublicGetOnly(t *testing.T) {
	svc := user.NewService(user.NewFakeRepository())
	created, err := svc.Create(context.Background(), user.CreateInput{Username: "p", Email: "p@example.com"})
	if err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(httpserver.PublicMux(svc))
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/public/v1/users/" + created.ID)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("public get %d", resp.StatusCode)
	}

	resp, err = http.Get(ts.URL + "/public/v1/users")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("public list %d", resp.StatusCode)
	}

	resp, err = http.Post(ts.URL+"/public/v1/users", "application/json", bytes.NewReader([]byte(`{}`)))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("public create %d", resp.StatusCode)
	}

	req, _ := http.NewRequest(http.MethodDelete, ts.URL+"/public/v1/users/"+created.ID, nil)
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("public delete %d", resp.StatusCode)
	}

	resp, err = http.Get(ts.URL + "/public/v1/users/00000000-0000-0000-0000-000000000001")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("public missing %d", resp.StatusCode)
	}
}
