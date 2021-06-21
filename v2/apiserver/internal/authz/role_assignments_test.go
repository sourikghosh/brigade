package authz

import (
	"context"
	"errors"
	"testing"

	"github.com/brigadecore/brigade/v2/apiserver/internal/authn"
	libAuthz "github.com/brigadecore/brigade/v2/apiserver/internal/lib/authz"
	"github.com/brigadecore/brigade/v2/apiserver/internal/meta"
	metaTesting "github.com/brigadecore/brigade/v2/apiserver/internal/meta/testing" // nolint: lll
	"github.com/stretchr/testify/require"
)

func TestRoleAssignmentListMarshalJSON(t *testing.T) {
	metaTesting.RequireAPIVersionAndType(
		t,
		&RoleAssignmentList{},
		RoleAssignmentListKind,
	)
}

func TestNewRoleAssignmentsService(t *testing.T) {
	usersStore := &authn.MockUsersStore{}
	serviceAccountsStore := &authn.MockServiceAccountStore{}
	roleAssignmentsStore := &MockRoleAssignmentsStore{}
	svc := NewRoleAssignmentsService(
		libAuthz.AlwaysAuthorize,
		usersStore,
		serviceAccountsStore,
		roleAssignmentsStore,
	)
	require.NotNil(t, svc.(*roleAssignmentsService).authorize)
	require.Same(t, usersStore, svc.(*roleAssignmentsService).usersStore)
	require.Same(
		t,
		serviceAccountsStore,
		svc.(*roleAssignmentsService).serviceAccountsStore,
	)
	require.Same(
		t,
		roleAssignmentsStore,
		svc.(*roleAssignmentsService).roleAssignmentsStore,
	)
}

func TestRoleAssignmentsServiceGrant(t *testing.T) {
	testCases := []struct {
		name           string
		roleAssignment libAuthz.RoleAssignment
		service        RoleAssignmentsService
		assertions     func(error)
	}{
		{
			name: "unauthorized",
			service: &roleAssignmentsService{
				authorize: libAuthz.NeverAuthorize,
			},
			assertions: func(err error) {
				require.Error(t, err)
				require.IsType(t, &meta.ErrAuthorization{}, err)
			},
		},
		{
			name: "error retrieving user from store",
			roleAssignment: libAuthz.RoleAssignment{
				Principal: libAuthz.PrincipalReference{
					Type: PrincipalTypeUser,
					ID:   "foo",
				},
			},
			service: &roleAssignmentsService{
				authorize: libAuthz.AlwaysAuthorize,
				usersStore: &authn.MockUsersStore{
					GetFn: func(context.Context, string) (authn.User, error) {
						return authn.User{}, errors.New("something went wrong")
					},
				},
			},
			assertions: func(err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "something went wrong")
				require.Contains(t, err.Error(), "error retrieving user")
			},
		},
		{
			name: "error retrieving service account from store",
			roleAssignment: libAuthz.RoleAssignment{
				Principal: libAuthz.PrincipalReference{
					Type: PrincipalTypeServiceAccount,
					ID:   "foo",
				},
			},
			service: &roleAssignmentsService{
				authorize: libAuthz.AlwaysAuthorize,
				serviceAccountsStore: &authn.MockServiceAccountStore{
					GetFn: func(context.Context, string) (authn.ServiceAccount, error) {
						return authn.ServiceAccount{}, errors.New("something went wrong")
					},
				},
			},
			assertions: func(err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "something went wrong")
				require.Contains(t, err.Error(), "error retrieving service account")
			},
		},
		{
			name: "error granting the role",
			roleAssignment: libAuthz.RoleAssignment{
				Principal: libAuthz.PrincipalReference{
					Type: PrincipalTypeServiceAccount,
					ID:   "foo",
				},
			},
			service: &roleAssignmentsService{
				authorize: libAuthz.AlwaysAuthorize,
				serviceAccountsStore: &authn.MockServiceAccountStore{
					GetFn: func(context.Context, string) (authn.ServiceAccount, error) {
						return authn.ServiceAccount{}, nil
					},
				},
				roleAssignmentsStore: &MockRoleAssignmentsStore{
					GrantFn: func(context.Context, libAuthz.RoleAssignment) error {
						return errors.New("something went wrong")
					},
				},
			},
			assertions: func(err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "something went wrong")
				require.Contains(t, err.Error(), "error granting role")
			},
		},
		{
			name: "success",
			roleAssignment: libAuthz.RoleAssignment{
				Principal: libAuthz.PrincipalReference{
					Type: PrincipalTypeServiceAccount,
					ID:   "foo",
				},
			},
			service: &roleAssignmentsService{
				authorize: libAuthz.AlwaysAuthorize,
				serviceAccountsStore: &authn.MockServiceAccountStore{
					GetFn: func(context.Context, string) (authn.ServiceAccount, error) {
						return authn.ServiceAccount{}, nil
					},
				},
				roleAssignmentsStore: &MockRoleAssignmentsStore{
					GrantFn: func(context.Context, libAuthz.RoleAssignment) error {
						return nil
					},
				},
			},
			assertions: func(err error) {
				require.NoError(t, err)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			err := testCase.service.Grant(
				context.Background(),
				testCase.roleAssignment,
			)
			testCase.assertions(err)
		})
	}
}

func TestRoleAssignmentsServiceList(t *testing.T) {
	testCases := []struct {
		name       string
		service    RoleAssignmentsService
		assertions func(error)
	}{
		{
			name: "unauthorized",
			service: &roleAssignmentsService{
				authorize: libAuthz.NeverAuthorize,
			},
			assertions: func(err error) {
				require.Error(t, err)
				require.IsType(t, &meta.ErrAuthorization{}, err)
			},
		},
		{
			name: "error getting role assignments from store",
			service: &roleAssignmentsService{
				authorize: libAuthz.AlwaysAuthorize,
				roleAssignmentsStore: &MockRoleAssignmentsStore{
					ListFn: func(
						context.Context,
						RoleAssignmentsSelector,
						meta.ListOptions,
					) (RoleAssignmentList, error) {
						return RoleAssignmentList{}, errors.New("something went wrong")
					},
				},
			},
			assertions: func(err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "something went wrong")
				require.Contains(
					t,
					err.Error(),
					"error retrieving role assignments from store",
				)
			},
		},
		{
			name: "success",
			service: &roleAssignmentsService{
				authorize: libAuthz.AlwaysAuthorize,
				roleAssignmentsStore: &MockRoleAssignmentsStore{
					ListFn: func(
						context.Context,
						RoleAssignmentsSelector,
						meta.ListOptions,
					) (RoleAssignmentList, error) {
						return RoleAssignmentList{}, nil
					},
				},
			},
			assertions: func(err error) {
				require.NoError(t, err)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
		})
	}
}

func TestRoleAssignmentsServiceRevoke(t *testing.T) {
	testCases := []struct {
		name           string
		roleAssignment libAuthz.RoleAssignment
		service        RoleAssignmentsService
		assertions     func(error)
	}{
		{
			name: "unauthorized",
			service: &roleAssignmentsService{
				authorize: libAuthz.NeverAuthorize,
			},
			assertions: func(err error) {
				require.Error(t, err)
				require.IsType(t, &meta.ErrAuthorization{}, err)
			},
		},
		{
			name: "error retrieving user from store",
			roleAssignment: libAuthz.RoleAssignment{
				Principal: libAuthz.PrincipalReference{
					Type: PrincipalTypeUser,
					ID:   "foo",
				},
			},
			service: &roleAssignmentsService{
				authorize: libAuthz.AlwaysAuthorize,
				usersStore: &authn.MockUsersStore{
					GetFn: func(context.Context, string) (authn.User, error) {
						return authn.User{}, errors.New("something went wrong")
					},
				},
			},
			assertions: func(err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "something went wrong")
				require.Contains(t, err.Error(), "error retrieving user")
			},
		},
		{
			name: "error retrieving service account from store",
			roleAssignment: libAuthz.RoleAssignment{
				Principal: libAuthz.PrincipalReference{
					Type: PrincipalTypeServiceAccount,
					ID:   "foo",
				},
			},
			service: &roleAssignmentsService{
				authorize: libAuthz.AlwaysAuthorize,
				serviceAccountsStore: &authn.MockServiceAccountStore{
					GetFn: func(context.Context, string) (authn.ServiceAccount, error) {
						return authn.ServiceAccount{}, errors.New("something went wrong")
					},
				},
			},
			assertions: func(err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "something went wrong")
				require.Contains(t, err.Error(), "error retrieving service account")
			},
		},
		{
			name: "error revoking the role",
			roleAssignment: libAuthz.RoleAssignment{
				Principal: libAuthz.PrincipalReference{
					Type: PrincipalTypeServiceAccount,
					ID:   "foo",
				},
			},
			service: &roleAssignmentsService{
				authorize: libAuthz.AlwaysAuthorize,
				serviceAccountsStore: &authn.MockServiceAccountStore{
					GetFn: func(context.Context, string) (authn.ServiceAccount, error) {
						return authn.ServiceAccount{}, nil
					},
				},
				roleAssignmentsStore: &MockRoleAssignmentsStore{
					RevokeFn: func(context.Context, libAuthz.RoleAssignment) error {
						return errors.New("something went wrong")
					},
				},
			},
			assertions: func(err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "something went wrong")
				require.Contains(t, err.Error(), "error revoking role")
			},
		},
		{
			name: "success",
			roleAssignment: libAuthz.RoleAssignment{
				Principal: libAuthz.PrincipalReference{
					Type: PrincipalTypeServiceAccount,
					ID:   "foo",
				},
			},
			service: &roleAssignmentsService{
				authorize: libAuthz.AlwaysAuthorize,
				serviceAccountsStore: &authn.MockServiceAccountStore{
					GetFn: func(context.Context, string) (authn.ServiceAccount, error) {
						return authn.ServiceAccount{}, nil
					},
				},
				roleAssignmentsStore: &MockRoleAssignmentsStore{
					RevokeFn: func(context.Context, libAuthz.RoleAssignment) error {
						return nil
					},
				},
			},
			assertions: func(err error) {
				require.NoError(t, err)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			err := testCase.service.Revoke(
				context.Background(),
				testCase.roleAssignment,
			)
			testCase.assertions(err)
		})
	}
}