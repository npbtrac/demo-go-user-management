package grpcsvc

import (
	"context"
	"errors"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	usermgmtv1 "github.com/npbtrac/demo-go-user-management/api/proto/usermgmt/v1"
	"github.com/npbtrac/demo-go-user-management/internal/user"
)

// Server implements usermgmt.v1.UserService.
type Server struct {
	usermgmtv1.UnimplementedUserServiceServer
	svc *user.Service
}

func NewServer(svc *user.Service) *Server {
	return &Server{svc: svc}
}

func (s *Server) CreateUser(ctx context.Context, req *usermgmtv1.CreateUserRequest) (*usermgmtv1.User, error) {
	u, err := s.svc.Create(ctx, user.CreateInput{
		Username: req.GetUsername(),
		Email:    req.GetEmail(),
		Phone:    phonePtr(req.GetPhone()),
		Params:   structToParams(req.GetParams()),
	})
	if err != nil {
		return nil, toStatus(err)
	}
	out, err := toProto(u)
	if err != nil {
		return nil, toStatus(err)
	}
	return out, nil
}

func (s *Server) GetUser(ctx context.Context, req *usermgmtv1.GetUserRequest) (*usermgmtv1.User, error) {
	u, err := s.svc.Get(ctx, req.GetId())
	if err != nil {
		return nil, toStatus(err)
	}
	out, err := toProto(u)
	if err != nil {
		return nil, toStatus(err)
	}
	return out, nil
}

func (s *Server) ListUsers(ctx context.Context, req *usermgmtv1.ListUsersRequest) (*usermgmtv1.ListUsersResponse, error) {
	page, err := s.svc.List(ctx, req.GetPageSize(), req.GetPageToken())
	if err != nil {
		return nil, toStatus(err)
	}
	out := &usermgmtv1.ListUsersResponse{
		Users:         make([]*usermgmtv1.User, 0, len(page.Users)),
		NextPageToken: page.NextPageToken,
	}
	for _, u := range page.Users {
		p, err := toProto(u)
		if err != nil {
			return nil, toStatus(err)
		}
		out.Users = append(out.Users, p)
	}
	return out, nil
}

func (s *Server) UpdateUser(ctx context.Context, req *usermgmtv1.UpdateUserRequest) (*usermgmtv1.User, error) {
	in := user.UpdateInput{}
	if req.Username != nil {
		in.Username = req.Username
	}
	if req.Email != nil {
		in.Email = req.Email
	}
	if req.Phone != nil {
		in.Phone = req.Phone
	}
	if req.Params != nil {
		params := structToParams(req.Params)
		in.Params = &params
	}
	u, err := s.svc.Update(ctx, req.GetId(), in)
	if err != nil {
		return nil, toStatus(err)
	}
	out, err := toProto(u)
	if err != nil {
		return nil, toStatus(err)
	}
	return out, nil
}

func (s *Server) DeleteUser(ctx context.Context, req *usermgmtv1.DeleteUserRequest) (*emptypb.Empty, error) {
	if err := s.svc.Delete(ctx, req.GetId()); err != nil {
		return nil, toStatus(err)
	}
	return &emptypb.Empty{}, nil
}

func toStatus(err error) error {
	if err == nil {
		return nil
	}
	st, ok := status.FromError(err)
	if ok {
		return st.Err()
	}
	switch {
	case errors.Is(err, user.ErrInvalid):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, user.ErrNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, user.ErrConflict):
		return status.Error(codes.AlreadyExists, err.Error())
	default:
		return status.Error(codes.Internal, "internal error")
	}
}

// ListenAndServe starts a gRPC server on addr until ctx is cancelled.
func ListenAndServe(ctx context.Context, addr string, svc *user.Service) error {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	gs := grpc.NewServer()
	usermgmtv1.RegisterUserServiceServer(gs, NewServer(svc))
	reflection.Register(gs)
	errCh := make(chan error, 1)
	go func() {
		errCh <- gs.Serve(ln)
	}()
	select {
	case <-ctx.Done():
		gs.GracefulStop()
		return <-errCh
	case err := <-errCh:
		return err
	}
}
