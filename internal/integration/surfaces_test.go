package integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/structpb"

	usermgmtv1 "github.com/npbtrac/demo-go-user-management/api/proto/usermgmt/v1"
	"github.com/npbtrac/demo-go-user-management/internal/db"
	grpcsvc "github.com/npbtrac/demo-go-user-management/internal/grpc"
	httpserver "github.com/npbtrac/demo-go-user-management/internal/http"
	"github.com/npbtrac/demo-go-user-management/internal/user"
)

type stack struct {
	publicURL   string
	internalURL string
	grpc        usermgmtv1.UserServiceClient
}

func startStack(t *testing.T) stack {
	t.Helper()
	ctx := context.Background()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		req := testcontainers.ContainerRequest{
			Image:        "postgres:16-alpine",
			ExposedPorts: []string{"5432/tcp"},
			Env: map[string]string{
				"POSTGRES_USER":     "usermgmt",
				"POSTGRES_PASSWORD": "usermgmt",
				"POSTGRES_DB":       "usermgmt",
			},
			WaitingFor: wait.ForListeningPort("5432/tcp").WithStartupTimeout(60 * time.Second),
		}
		pg, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
			ContainerRequest: req,
			Started:          true,
		})
		if err != nil {
			t.Skipf("postgres container unavailable: %v", err)
		}
		t.Cleanup(func() { _ = pg.Terminate(ctx) })
		host, err := pg.Host(ctx)
		if err != nil {
			t.Fatal(err)
		}
		port, err := pg.MappedPort(ctx, "5432")
		if err != nil {
			t.Fatal(err)
		}
		dsn = fmt.Sprintf("postgres://usermgmt:usermgmt@%s:%s/usermgmt?sslmode=disable", host, port.Port())
	}
	if err := db.Migrate(dsn); err != nil {
		t.Fatal(err)
	}
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	svc := user.NewService(user.NewPostgresRepository(pool))

	publicLn, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	internalLn, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	grpcLn, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	publicSrv := &http.Server{Handler: httpserver.PublicMux(svc)}
	internalSrv := &http.Server{Handler: httpserver.InternalMux(svc)}
	gs := grpc.NewServer()
	usermgmtv1.RegisterUserServiceServer(gs, grpcsvc.NewServer(svc))
	go publicSrv.Serve(publicLn)
	go internalSrv.Serve(internalLn)
	go gs.Serve(grpcLn)
	t.Cleanup(func() {
		_ = publicSrv.Close()
		_ = internalSrv.Close()
		gs.Stop()
	})
	conn, err := grpc.NewClient(grpcLn.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	return stack{
		publicURL:   "http://" + publicLn.Addr().String(),
		internalURL: "http://" + internalLn.Addr().String(),
		grpc:        usermgmtv1.NewUserServiceClient(conn),
	}
}

func TestIntegrationSurfaces(t *testing.T) {
	s := startStack(t)
	ctx := context.Background()
	params, err := structpb.NewStruct(map[string]any{"source": "it"})
	if err != nil {
		t.Fatal(err)
	}
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	created, err := s.grpc.CreateUser(ctx, &usermgmtv1.CreateUserRequest{
		Username: "grpc-user-" + suffix,
		Email:    "grpc-" + suffix + "@example.com",
		Phone:    "+1000",
		Params:   params,
	})
	if err != nil {
		t.Fatalf("grpc create: %v", err)
	}
	got, err := s.grpc.GetUser(ctx, &usermgmtv1.GetUserRequest{Id: created.GetId()})
	if err != nil {
		t.Fatalf("grpc get: %v", err)
	}
	if got.GetParams().GetFields()["source"].GetStringValue() != "it" {
		t.Fatalf("grpc params %+v", got.GetParams())
	}
	if _, err := s.grpc.ListUsers(ctx, &usermgmtv1.ListUsersRequest{PageSize: 10}); err != nil {
		t.Fatalf("grpc list: %v", err)
	}
	phone := "+2000"
	if _, err := s.grpc.UpdateUser(ctx, &usermgmtv1.UpdateUserRequest{Id: created.GetId(), Phone: &phone}); err != nil {
		t.Fatalf("grpc update: %v", err)
	}

	body := []byte(fmt.Sprintf(`{"username":"rest-user-%s","email":"rest-%s@example.com","params":{"a":1}}`, suffix, suffix))
	resp, err := http.Post(s.internalURL+"/internal/v1/users", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("rest create %d", resp.StatusCode)
	}
	var createdREST map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&createdREST); err != nil {
		t.Fatal(err)
	}
	id := createdREST["id"].(string)

	resp, err = http.Get(s.internalURL + "/internal/v1/users/" + id)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("rest get %d", resp.StatusCode)
	}
	resp, err = http.Get(s.internalURL + "/internal/v1/users")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("rest list %d", resp.StatusCode)
	}
	req, _ := http.NewRequest(http.MethodPatch, s.internalURL+"/internal/v1/users/"+id, bytes.NewReader([]byte(`{"params":{"a":2}}`)))
	req.Header.Set("Content-Type", "application/json")
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("rest patch %d", resp.StatusCode)
	}
	req, _ = http.NewRequest(http.MethodPut, s.internalURL+"/internal/v1/users/"+id, bytes.NewReader([]byte(fmt.Sprintf(`{"username":"rest-user-%s","email":"rest2-%s@example.com"}`, suffix, suffix))))
	req.Header.Set("Content-Type", "application/json")
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("rest put %d", resp.StatusCode)
	}

	resp, err = http.Get(s.publicURL + "/public/v1/users/" + id)
	if err != nil {
		t.Fatal(err)
	}
	var publicUser map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&publicUser); err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("public get %d", resp.StatusCode)
	}
	if publicUser["email"] != "rest2-"+suffix+"@example.com" {
		t.Fatalf("public email %+v", publicUser)
	}

	resp, err = http.Post(s.publicURL+"/public/v1/users", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("public create %d", resp.StatusCode)
	}

	req, _ = http.NewRequest(http.MethodDelete, s.internalURL+"/internal/v1/users/"+id, nil)
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("rest delete %d", resp.StatusCode)
	}
	if _, err := s.grpc.DeleteUser(ctx, &usermgmtv1.DeleteUserRequest{Id: created.GetId()}); err != nil {
		t.Fatalf("grpc delete: %v", err)
	}
}
