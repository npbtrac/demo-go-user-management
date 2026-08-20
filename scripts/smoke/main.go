package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/structpb"

	usermgmtv1 "github.com/npbtrac/demo-go-user-management/api/proto/usermgmt/v1"
)

func main() {
	only := flag.String("only", "all", "which surface to check: all, public, internal, or service")
	flag.Parse()
	_ = godotenv.Load()

	publicBase := fmt.Sprintf("http://127.0.0.1:%s", envOr("PUBLIC_HTTP_PORT", "10100"))
	internalBase := fmt.Sprintf("http://127.0.0.1:%s", envOr("INTERNAL_HTTP_PORT", "10101"))
	serviceAPIAddr := fmt.Sprintf("127.0.0.1:%s", envOr("SERVICE_API_PORT", "10102"))

	var err error
	switch strings.ToLower(*only) {
	case "public":
		err = checkPublic(publicBase, "")
	case "internal":
		_, err = checkInternal(internalBase)
	case "service":
		err = checkServiceAPI(serviceAPIAddr)
	case "all":
		err = checkAll(publicBase, internalBase, serviceAPIAddr)
	default:
		err = fmt.Errorf("unknown -only=%s (want all, public, internal, or service)", *only)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "FAIL: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("OK")
}

func checkAll(publicBase, internalBase, serviceAPIAddr string) error {
	id, err := checkInternal(internalBase)
	if err != nil {
		return err
	}
	if err := checkPublic(publicBase, id); err != nil {
		return err
	}
	return checkServiceAPI(serviceAPIAddr)
}

func checkPublic(base, knownID string) error {
	if err := waitHealth(base + "/healthz"); err != nil {
		return fmt.Errorf("public healthz: %w", err)
	}
	if knownID != "" {
		if _, err := httpJSON(http.MethodGet, base+"/public/v1/users/"+knownID, nil, http.StatusOK); err != nil {
			return fmt.Errorf("public get: %w", err)
		}
	} else {
		if _, err := httpJSON(http.MethodGet, base+"/public/v1/users/00000000-0000-0000-0000-000000000001", nil, http.StatusNotFound); err != nil {
			return fmt.Errorf("public get missing: %w", err)
		}
	}
	if _, err := httpJSON(http.MethodPost, base+"/public/v1/users", []byte(`{}`), http.StatusNotFound); err != nil {
		return fmt.Errorf("public create must be unavailable: %w", err)
	}
	fmt.Println("public REST: ok")
	return nil
}

func checkInternal(base string) (string, error) {
	if err := waitHealth(base + "/healthz"); err != nil {
		return "", fmt.Errorf("internal healthz: %w", err)
	}
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	body := fmt.Sprintf(`{"username":"smoke-%s","email":"smoke-%s@example.com","phone":"+100","params":{"via":"make"}}`, suffix, suffix)
	created, err := httpJSON(http.MethodPost, base+"/internal/v1/users", []byte(body), http.StatusCreated)
	if err != nil {
		return "", fmt.Errorf("internal create: %w", err)
	}
	id, _ := created["id"].(string)
	if id == "" {
		return "", fmt.Errorf("internal create: missing id in %v", created)
	}
	if _, err := httpJSON(http.MethodGet, base+"/internal/v1/users/"+id, nil, http.StatusOK); err != nil {
		return "", fmt.Errorf("internal get: %w", err)
	}
	if _, err := httpJSON(http.MethodGet, base+"/internal/v1/users", nil, http.StatusOK); err != nil {
		return "", fmt.Errorf("internal list: %w", err)
	}
	patch := []byte(`{"params":{"via":"make","ok":true}}`)
	if _, err := httpJSON(http.MethodPatch, base+"/internal/v1/users/"+id, patch, http.StatusOK); err != nil {
		return "", fmt.Errorf("internal patch: %w", err)
	}
	fmt.Println("internal REST: ok")
	return id, nil
}

func checkServiceAPI(addr string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("service API dial %s: %w", addr, err)
	}
	defer conn.Close()
	client := usermgmtv1.NewUserServiceClient(conn)
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	params, err := structpb.NewStruct(map[string]any{"via": "make"})
	if err != nil {
		return err
	}
	created, err := client.CreateUser(ctx, &usermgmtv1.CreateUserRequest{
		Username: "service-smoke-" + suffix,
		Email:    "service-smoke-" + suffix + "@example.com",
		Params:   params,
	})
	if err != nil {
		return fmt.Errorf("service API CreateUser: %w", err)
	}
	if _, err := client.GetUser(ctx, &usermgmtv1.GetUserRequest{Id: created.GetId()}); err != nil {
		return fmt.Errorf("service API GetUser: %w", err)
	}
	if _, err := client.ListUsers(ctx, &usermgmtv1.ListUsersRequest{PageSize: 5}); err != nil {
		return fmt.Errorf("service API ListUsers: %w", err)
	}
	phone := "+101"
	if _, err := client.UpdateUser(ctx, &usermgmtv1.UpdateUserRequest{Id: created.GetId(), Phone: &phone}); err != nil {
		return fmt.Errorf("service API UpdateUser: %w", err)
	}
	if _, err := client.DeleteUser(ctx, &usermgmtv1.DeleteUserRequest{Id: created.GetId()}); err != nil {
		return fmt.Errorf("service API DeleteUser: %w", err)
	}
	fmt.Println("Service API: ok")
	return nil
}

func waitHealth(url string) error {
	var last error
	for i := 0; i < 30; i++ {
		resp, err := http.Get(url)
		if err == nil {
			_, _ = io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
			last = fmt.Errorf("status %d", resp.StatusCode)
		} else {
			last = err
		}
		time.Sleep(time.Second)
	}
	return last
}

func httpJSON(method, url string, body []byte, want int) (map[string]any, error) {
	var rdr io.Reader
	if body != nil {
		rdr = bytes.NewReader(body)
	}
	req, err := http.NewRequest(method, url, rdr)
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != want {
		return nil, fmt.Errorf("%s %s: got %d want %d (%s)", method, url, resp.StatusCode, want, strings.TrimSpace(string(raw)))
	}
	if len(raw) == 0 {
		return map[string]any{}, nil
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		return map[string]any{}, nil
	}
	return out, nil
}

func envOr(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}
