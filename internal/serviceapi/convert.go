package serviceapi

import (
	"encoding/json"

	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	usermgmtv1 "github.com/npbtrac/demo-go-user-management/api/proto/usermgmt/v1"
	"github.com/npbtrac/demo-go-user-management/internal/user"
)

func toProto(u user.User) (*usermgmtv1.User, error) {
	params, err := paramsToStruct(u.Params)
	if err != nil {
		return nil, err
	}
	phone := ""
	if u.Phone != nil {
		phone = *u.Phone
	}
	return &usermgmtv1.User{
		Id:        u.ID,
		Username:  u.Username,
		Email:     u.Email,
		Phone:     phone,
		Params:    params,
		CreatedAt: timestamppb.New(u.CreatedAt.UTC()),
		UpdatedAt: timestamppb.New(u.UpdatedAt.UTC()),
	}, nil
}

func paramsToStruct(raw json.RawMessage) (*structpb.Struct, error) {
	if len(raw) == 0 {
		return &structpb.Struct{Fields: map[string]*structpb.Value{}}, nil
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, err
	}
	if m == nil {
		m = map[string]any{}
	}
	return structpb.NewStruct(m)
}

func structToParams(s *structpb.Struct) json.RawMessage {
	if s == nil {
		return nil
	}
	b, err := s.MarshalJSON()
	if err != nil {
		return json.RawMessage(`{}`)
	}
	return b
}

func phonePtr(phone string) *string {
	if phone == "" {
		return nil
	}
	p := phone
	return &p
}
