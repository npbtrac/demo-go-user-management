package grpcsvc_test

import (
	"context"
	"net"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"

	usermgmtv1 "github.com/npbtrac/demo-go-user-management/api/proto/usermgmt/v1"
	grpcsvc "github.com/npbtrac/demo-go-user-management/internal/grpc"
	"github.com/npbtrac/demo-go-user-management/internal/user"
)

func startGRPC(t *testing.T, svc *user.Service) usermgmtv1.UserServiceClient {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	gs := grpc.NewServer()
	usermgmtv1.RegisterUserServiceServer(gs, grpcsvc.NewServer(svc))
	go gs.Serve(ln)
	t.Cleanup(gs.Stop)
	conn, err := grpc.NewClient(ln.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	return usermgmtv1.NewUserServiceClient(conn)
}

func TestGRPCCRUDAndErrors(t *testing.T) {
	svc := user.NewService(user.NewFakeRepository())
	client := startGRPC(t, svc)
	ctx := context.Background()

	params, err := structpb.NewStruct(map[string]any{"tier": "gold"})
	if err != nil {
		t.Fatal(err)
	}
	created, err := client.CreateUser(ctx, &usermgmtv1.CreateUserRequest{
		Username: "alice",
		Email:    "alice@example.com",
		Phone:    "+1",
		Params:   params,
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if created.GetId() == "" || created.GetCreatedAt() == nil {
		t.Fatalf("created: %+v", created)
	}
	if created.GetParams().GetFields()["tier"].GetStringValue() != "gold" {
		t.Fatalf("params: %v", created.GetParams())
	}

	got, err := client.GetUser(ctx, &usermgmtv1.GetUserRequest{Id: created.GetId()})
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.GetEmail() != "alice@example.com" {
		t.Fatalf("email %s", got.GetEmail())
	}

	listed, err := client.ListUsers(ctx, &usermgmtv1.ListUsersRequest{PageSize: 10})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(listed.GetUsers()) != 1 {
		t.Fatalf("list size %d", len(listed.GetUsers()))
	}

	username := "alice2"
	updated, err := client.UpdateUser(ctx, &usermgmtv1.UpdateUserRequest{Id: created.GetId(), Username: &username})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if updated.GetUsername() != "alice2" {
		t.Fatalf("username %s", updated.GetUsername())
	}

	if _, err := client.CreateUser(ctx, &usermgmtv1.CreateUserRequest{Username: "alice2", Email: "x@example.com"}); status.Code(err) != codes.AlreadyExists {
		t.Fatalf("conflict: %v", err)
	}
	if _, err := client.CreateUser(ctx, &usermgmtv1.CreateUserRequest{Username: "", Email: "x@example.com"}); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("invalid: %v", err)
	}
	if _, err := client.GetUser(ctx, &usermgmtv1.GetUserRequest{Id: "00000000-0000-0000-0000-000000000001"}); status.Code(err) != codes.NotFound {
		t.Fatalf("not found: %v", err)
	}

	if _, err := client.DeleteUser(ctx, &usermgmtv1.DeleteUserRequest{Id: created.GetId()}); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := client.GetUser(ctx, &usermgmtv1.GetUserRequest{Id: created.GetId()}); status.Code(err) != codes.NotFound {
		t.Fatalf("deleted get: %v", err)
	}
}
