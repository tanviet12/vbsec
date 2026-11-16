package authz

import (
	"example.com/tallyboard/internal/model"
	"example.com/tallyboard/internal/tenant"
)

const (
	RoleOwner      = "owner"
	RoleAdmin      = "admin"
	RoleAccountant = "accountant"
	RoleMember     = "member"
	RoleClient     = "client"

	PermFinance      = "finance"
	PermBillingAdmin = "billing_admin"
	PermExports      = "exports"
	PermTimesheets   = "timesheets"
)

var ownerGranted = map[string]bool{
	PermFinance:      true,
	PermBillingAdmin: true,
}

var knownPerms = map[string]bool{
	PermFinance:      true,
	PermBillingAdmin: true,
	PermExports:      true,
	PermTimesheets:   true,
}

func IsStaff(a *tenant.Actor) bool {
	return a.Role != RoleClient
}

func CanViewFinance(a *tenant.Actor) bool {
	return a.Role == RoleOwner || a.Role == RoleAccountant || a.Has(PermFinance)
}

func CanManageProject(a *tenant.Actor, p *model.Project) bool {
	if a.Role == RoleOwner || a.Role == RoleAdmin {
		return true
	}
	return IsStaff(a) && p.ManagerID == a.MemberID
}

func CanManageInvoices(a *tenant.Actor) bool {
	return a.Role == RoleOwner || a.Role == RoleAccountant
}

// CanBill: accountants and owners bill any project; a manager bills their own
// project once they hold the finance permission.
func CanBill(a *tenant.Actor, p *model.Project) bool {
	if CanManageInvoices(a) {
		return true
	}
	return p.ManagerID == a.MemberID && a.Has(PermFinance)
}

func CanManageMembers(a *tenant.Actor) bool {
	return a.Role == RoleOwner || a.Role == RoleAdmin
}

// CanGrant reports whether the actor may hand out a permission. Finance and
// billing administration are granted by the owner only.
func CanGrant(a *tenant.Actor, perm string) bool {
	if !knownPerms[perm] {
		return false
	}
	if ownerGranted[perm] {
		return a.Role == RoleOwner
	}
	return CanManageMembers(a)
}

// ProjectView shapes a project for the caller. Budget and rate are only
// included for people who may see financial figures.
func ProjectView(a *tenant.Actor, p *model.Project) map[string]any {
	v := map[string]any{
		"id":         p.ID,
		"code":       p.Code,
		"name":       p.Name,
		"manager_id": p.ManagerID,
		"archived":   p.ArchivedAt != nil,
	}
	if IsStaff(a) {
		v["notes"] = p.Notes
	}
	if CanViewFinance(a) {
		v["budget_cents"] = p.BudgetCents
		v["hourly_rate_cents"] = p.HourlyRateCents
	}
	return v
}
