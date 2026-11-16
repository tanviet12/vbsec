package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"time"

	"example.com/tallyboard/internal/authz"
	"example.com/tallyboard/internal/model"
	"example.com/tallyboard/internal/repo"
	"example.com/tallyboard/internal/tenant"
)

type Members struct {
	Repo *repo.Members
}

var assignableRoles = map[string]bool{
	authz.RoleAdmin: true, authz.RoleAccountant: true, authz.RoleMember: true, authz.RoleClient: true,
}

func (s *Members) List(ctx context.Context, a *tenant.Actor) ([]*model.Member, error) {
	if !authz.IsStaff(a) {
		return nil, ErrForbidden
	}
	return s.Repo.List(ctx, a.CompanyID)
}

func (s *Members) SetRole(ctx context.Context, a *tenant.Actor, memberID int64, role string) error {
	if !authz.CanManageMembers(a) || !assignableRoles[role] {
		return ErrForbidden
	}
	target, err := s.Repo.Get(ctx, a.CompanyID, memberID)
	if err != nil {
		return err
	}
	if target.Role == authz.RoleOwner || target.ID == a.MemberID {
		return ErrForbidden
	}
	if (role == authz.RoleAdmin || role == authz.RoleAccountant) && a.Role != authz.RoleOwner {
		return ErrForbidden
	}
	return s.Repo.SetRole(ctx, a.CompanyID, memberID, role)
}

// SetPermissions replaces the extra permissions of a member. A permission the
// member does not hold yet can only be added by someone allowed to grant it.
func (s *Members) SetPermissions(ctx context.Context, a *tenant.Actor, memberID int64, perms []string) error {
	if !authz.CanManageMembers(a) {
		return ErrForbidden
	}
	target, err := s.Repo.Get(ctx, a.CompanyID, memberID)
	if err != nil {
		return err
	}
	if target.Role == authz.RoleClient {
		return ErrInvalid
	}
	current, err := s.Repo.Permissions(ctx, a.CompanyID, memberID)
	if err != nil {
		return err
	}
	held := map[string]bool{}
	for _, p := range current {
		held[p] = true
	}
	for _, p := range perms {
		if !held[p] && !authz.CanGrant(a, p) {
			return ErrForbidden
		}
	}
	return s.Repo.ReplacePermissions(ctx, a.CompanyID, memberID, a.MemberID, perms)
}

// CopyPermissions sets up a new colleague with the same extras as an
// existing one, e.g. when someone takes over a role.
func (s *Members) CopyPermissions(ctx context.Context, a *tenant.Actor, fromID, toID int64) ([]string, error) {
	if !authz.CanManageMembers(a) {
		return nil, ErrForbidden
	}
	if fromID == toID {
		return nil, ErrInvalid
	}
	if _, err := s.Repo.Get(ctx, a.CompanyID, fromID); err != nil {
		return nil, err
	}
	to, err := s.Repo.Get(ctx, a.CompanyID, toID)
	if err != nil {
		return nil, err
	}
	if to.Role == authz.RoleClient || to.Status == "left" {
		return nil, ErrInvalid
	}
	if err := s.Repo.CopyPermissions(ctx, a.CompanyID, fromID, toID, a.MemberID); err != nil {
		return nil, err
	}
	return s.Repo.Permissions(ctx, a.CompanyID, toID)
}

func (s *Members) AcceptInvite(ctx context.Context, inviteToken string) (string, error) {
	if len(inviteToken) != 48 {
		return "", ErrInvalid
	}
	session, err := newToken()
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256([]byte(session))
	expires := time.Now().UTC().Add(14 * 24 * time.Hour)
	if err := s.Repo.AcceptInvite(ctx, inviteToken, hex.EncodeToString(sum[:]), expires); err != nil {
		return "", err
	}
	return session, nil
}
