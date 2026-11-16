package httpapi

import (
	"database/sql"
	"log/slog"
	"net/http"

	"example.com/crewdesk/internal/agentapi"
	"example.com/crewdesk/internal/audit"
	"example.com/crewdesk/internal/platform"
	"example.com/crewdesk/internal/repo"
	"example.com/crewdesk/internal/service"
)

type Server struct {
	cfg        platform.Config
	db         *sql.DB
	log        *slog.Logger
	mail       *platform.Mailer
	accounts   *repo.Accounts
	members    *service.Members
	projects   *service.Projects
	files      *service.Files
	requests   *service.Requests
	settings   *service.Settings
	categories *service.Categories
	imports    *service.Imports
	activity   *service.Activity
	ops        *service.Ops
	agent      *agentapi.Dispatcher
}

func New(db *sql.DB, cfg platform.Config, logger *slog.Logger) *Server {
	trail := audit.New(db)
	mailer := platform.NewMailer(cfg.SMTPAddr, logger)
	settings := service.NewSettings(db, trail)
	return &Server{
		cfg: cfg, db: db, log: logger, mail: mailer,
		accounts:   repo.NewAccounts(db),
		members:    service.NewMembers(db, trail, mailer, cfg.BaseURL),
		projects:   service.NewProjects(db, trail),
		files:      service.NewFiles(db),
		requests:   service.NewRequests(db, trail, settings),
		settings:   settings,
		categories: service.NewCategories(db, trail),
		imports:    service.NewImports(db),
		activity:   service.NewActivity(db),
		ops:        service.NewOps(db, trail),
		agent:      agentapi.NewDispatcher(db),
	}
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /login", s.login)
	mux.HandleFunc("POST /password/forgot", s.forgotPassword)
	mux.HandleFunc("POST /password/reset", s.resetPassword)
	mux.HandleFunc("POST /invites/accept", s.acceptInvite)
	mux.Handle("POST /logout", s.signedIn(s.logout))
	mux.Handle("POST /me/password", s.signedIn(s.changePassword))
	mux.Handle("PATCH /me/profile", s.signedIn(s.updateProfile))
	mux.Handle("GET /me/security-events", s.signedIn(s.securityEvents))

	mux.Handle("POST /members/invite", s.signedIn(s.inviteMember))
	mux.Handle("DELETE /members/{id}", s.signedIn(s.removeMember))
	mux.Handle("POST /members/{id}/reinstate", s.signedIn(s.reinstateMember))
	mux.Handle("PUT /members/{id}/permissions", s.signedIn(s.setPermissions))
	mux.Handle("POST /members/{id}/two-factor/reset", s.signedIn(s.resetTwoFactor))

	mux.Handle("GET /projects/{id}", s.signedIn(s.getProject))
	mux.Handle("PATCH /projects/{id}", s.signedIn(s.updateProject))
	mux.Handle("POST /projects/{id}/notes", s.signedIn(s.addNote))
	mux.Handle("POST /projects/{id}/close", s.signedIn(s.closeProject))

	mux.Handle("POST /files", s.signedIn(s.uploadFile))
	mux.Handle("GET /files/{id}", s.signedIn(s.downloadFile))
	mux.Handle("POST /requests", s.signedIn(s.createRequest))
	mux.Handle("GET /requests/approvers", s.signedIn(s.listApprovers))
	mux.Handle("POST /requests/{id}/receipt", s.signedIn(s.attachReceipt))
	mux.Handle("POST /requests/{id}/submit", s.signedIn(s.submitRequest))
	mux.Handle("POST /requests/{id}/decision", s.signedIn(s.decideRequest))

	mux.Handle("PUT /settings/approval-mode", s.signedIn(s.setApprovalMode))
	mux.Handle("PUT /settings/retention", s.signedIn(s.setRetention))
	mux.Handle("PUT /settings/monthly-limit", s.signedIn(s.setMonthlyLimit))
	mux.Handle("POST /categories", s.signedIn(s.createCategory))
	mux.Handle("POST /imports/expenses", s.signedIn(s.importExpenses))
	mux.Handle("GET /activity", s.signedIn(s.searchActivity))

	mux.Handle("POST /agent/connections", s.signedIn(s.connectAgent))
	mux.Handle("DELETE /agent/connections/{id}", s.signedIn(s.disconnectAgent))
	mux.Handle("POST /agent/grants/payroll", s.signedIn(s.approvePayrollGrant))
	mux.HandleFunc("POST /agent/call", s.agentCall)

	mux.Handle("GET /ops/companies/{id}", s.operator(s.opsOverview))
	mux.Handle("GET /ops/companies/{id}/requests", s.operator(s.opsRequests))
	return mux
}
