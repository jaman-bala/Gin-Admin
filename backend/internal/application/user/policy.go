package user

import (
	domainUser "gin_auth_service/internal/domain/user"
	"gin_auth_service/pkg/errors"

	"uuid"
)

// Actor is the authenticated caller of an admin operation. It comes from the
// verified access token, never from the request body.
type Actor struct {
	ID   uuid.UUID
	Role domainUser.Role
}

// authorizeAdminAction is the baseline for create/patch/delete of other
// users. The router already requires the admin role; this is the second
// line of defense for the use case itself.
func authorizeAdminAction(actor Actor) error {
	if actor.Role.Level() < domainUser.LevelAdmin {
		return errors.ErrForbiddenRole
	}
	return nil
}

// authorizeManage allows acting on an existing user only if the target's
// role is not higher than the actor's. Without this an admin could reset a
// superuser's password (or block/delete them) and take over the account.
// Peers are allowed: an admin may manage other admins, which grants nothing
// they do not already have.
func authorizeManage(actor Actor, target *domainUser.User) error {
	if err := authorizeAdminAction(actor); err != nil {
		return err
	}
	if target.Role.Level() > actor.Role.Level() {
		return errors.ErrForbiddenRole
	}
	return nil
}

// authorizeAssign allows granting a role only up to the actor's own level,
// so nobody can mint an account more privileged than themselves.
func authorizeAssign(actor Actor, role domainUser.Role) error {
	if err := authorizeAdminAction(actor); err != nil {
		return err
	}
	if role.Level() > actor.Role.Level() {
		return errors.ErrForbiddenRole
	}
	return nil
}

func isActiveSuperuser(u *domainUser.User) bool {
	return u.Role == domainUser.RoleSuperuser && u.IsActive
}
