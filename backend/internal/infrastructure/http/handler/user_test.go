package handler

import (
	"context"
	"net/http"
	"testing"

	"gin_auth_service/internal/application/file"
	"gin_auth_service/internal/application/user"
	domainUser "gin_auth_service/internal/domain/user"
	appErrors "gin_auth_service/pkg/errors"

	"github.com/gin-gonic/gin"
	"uuid"
)

// stubUsers implements user.UseCase for the three admin mutations and
// records the actor each one received.
type stubUsers struct {
	user.UseCase
	err    error
	actor  user.Actor
	called bool
}

func (s *stubUsers) Create(_ context.Context, a user.Actor, _ user.UserRequestDTO, _ *file.FileUpload) (*user.UserResponseDTO, error) {
	s.called, s.actor = true, a
	return &user.UserResponseDTO{}, s.err
}
func (s *stubUsers) Patch(_ context.Context, a user.Actor, _ uuid.UUID, _ user.UserUpdateDTO, _ *file.FileUpload) (*user.UserResponseDTO, error) {
	s.called, s.actor = true, a
	return &user.UserResponseDTO{}, s.err
}
func (s *stubUsers) Delete(_ context.Context, a user.Actor, _ uuid.UUID) error {
	s.called, s.actor = true, a
	return s.err
}

// usersRouter mounts the admin mutations. caller, when non-nil, plays the
// part of AuthMiddleware by putting the authenticated user into the context.
func usersRouter(h *UserHandler, caller *domainUser.User) *gin.Engine {
	r := gin.New()
	if caller != nil {
		r.Use(func(c *gin.Context) { c.Set("user", caller); c.Next() })
	}
	r.POST("/users", h.Create)
	r.PATCH("/users/:id", h.Patch)
	r.DELETE("/users/:id", h.Delete)
	return r
}

const (
	createBody = `{"first_name":"A","last_name":"B","password":"Password123","phone":"+996500111222","role":"user"}`
	patchBody  = `{"first_name":"New"}`
)

func adminCaller() *domainUser.User {
	return &domainUser.User{ID: uuid.NewV7(), Role: domainUser.RoleAdmin, IsActive: true}
}

func TestUserMutations_FailClosedWithoutAuthenticatedActor(t *testing.T) {
	id := uuid.NewV7().String()
	requests := map[string]func(*gin.Engine) int{
		"create": func(r *gin.Engine) int { return post(r, "/users", createBody, nil).Code },
		"patch": func(r *gin.Engine) int {
			return doJSON(r, http.MethodPatch, "/users/"+id, patchBody).Code
		},
		"delete": func(r *gin.Engine) int { return doJSON(r, http.MethodDelete, "/users/"+id, "").Code },
	}
	for name, send := range requests {
		t.Run(name, func(t *testing.T) {
			stub := &stubUsers{}
			code := send(usersRouter(NewUserHandler(stub), nil)) // AuthMiddleware missing

			if code != http.StatusUnauthorized {
				t.Fatalf("status = %d, want 401", code)
			}
			if stub.called {
				t.Error("the use case must not run without an authenticated actor")
			}
		})
	}
}

func TestUserMutations_PassTheAuthenticatedActor(t *testing.T) {
	caller := adminCaller()
	id := uuid.NewV7().String()

	t.Run("create", func(t *testing.T) {
		stub := &stubUsers{}
		if code := post(usersRouter(NewUserHandler(stub), caller), "/users", createBody, nil).Code; code != http.StatusCreated {
			t.Fatalf("status = %d", code)
		}
		if stub.actor.ID != caller.ID || stub.actor.Role != domainUser.RoleAdmin {
			t.Errorf("actor = %+v, want the authenticated caller", stub.actor)
		}
	})
	t.Run("patch", func(t *testing.T) {
		stub := &stubUsers{}
		if code := doJSON(usersRouter(NewUserHandler(stub), caller), http.MethodPatch, "/users/"+id, patchBody).Code; code != http.StatusOK {
			t.Fatalf("status = %d", code)
		}
		if stub.actor.ID != caller.ID || stub.actor.Role != domainUser.RoleAdmin {
			t.Errorf("actor = %+v, want the authenticated caller", stub.actor)
		}
	})
	t.Run("delete", func(t *testing.T) {
		stub := &stubUsers{}
		if code := doJSON(usersRouter(NewUserHandler(stub), caller), http.MethodDelete, "/users/"+id, "").Code; code != http.StatusOK {
			t.Fatalf("status = %d", code)
		}
		if stub.actor.ID != caller.ID || stub.actor.Role != domainUser.RoleAdmin {
			t.Errorf("actor = %+v, want the authenticated caller", stub.actor)
		}
	})
}

func TestUserMutations_MapAuthorizationErrors(t *testing.T) {
	id := uuid.NewV7().String()
	for _, tc := range []struct {
		err  error
		want int
	}{
		{appErrors.ErrForbiddenRole, http.StatusForbidden},
		{appErrors.ErrLastSuperuser, http.StatusConflict},
	} {
		t.Run(tc.err.Error(), func(t *testing.T) {
			r := usersRouter(NewUserHandler(&stubUsers{err: tc.err}), adminCaller())
			for name, code := range map[string]int{
				"create": post(r, "/users", createBody, nil).Code,
				"patch":  doJSON(r, http.MethodPatch, "/users/"+id, patchBody).Code,
				"delete": doJSON(r, http.MethodDelete, "/users/"+id, "").Code,
			} {
				if code != tc.want {
					t.Errorf("%s: status = %d, want %d", name, code, tc.want)
				}
			}
		})
	}
}
