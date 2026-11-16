package httpapi

import (
	"net/http"

	"example.com/tallyboard/internal/assistant"
	"example.com/tallyboard/internal/repo"
	"example.com/tallyboard/internal/service"
)

type Server struct {
	Sessions *repo.Members
	Projects *service.Projects
	Invoices *service.Invoices
	Members  *service.Members
	Imports  *service.Imports
	Reports  *service.Reports
	Writer   *assistant.Writer
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()

	mux.Handle("GET /projects", s.auth(s.listProjects))
	mux.Handle("GET /projects/{id}", s.auth(s.getProject))
	mux.Handle("POST /projects/{id}/archive", s.auth(s.archiveProject))
	mux.Handle("POST /projects/{id}/restore", s.auth(s.restoreProject))
	mux.Handle("POST /projects/{id}/delete-request", s.auth(s.requestProjectDelete))
	mux.Handle("DELETE /projects/{id}", s.auth(s.deleteProject))
	mux.Handle("POST /projects/{id}/expenses", s.auth(s.addExpense))
	mux.Handle("GET /projects/{id}/digest", s.auth(s.projectDigest))

	mux.Handle("GET /tasks/{id}/comments", s.auth(s.listComments))
	mux.Handle("POST /tasks/{id}/comments", s.auth(s.addComment))

	mux.Handle("GET /invoices", s.auth(s.listInvoices))
	mux.Handle("POST /invoices", s.auth(s.createInvoice))
	mux.Handle("GET /invoices/{id}", s.auth(s.getInvoice))
	mux.Handle("POST /invoices/{id}/paid", s.auth(s.markInvoicePaid))
	mux.Handle("POST /invoices/{id}/void", s.auth(s.voidInvoice))
	mux.Handle("POST /invoices/{id}/restore", s.auth(s.restoreInvoice))
	mux.Handle("GET /invoices/{id}/reply-draft", s.auth(s.replyDraft))
	mux.Handle("POST /imports/invoices", s.auth(s.importInvoices))
	mux.Handle("POST /exports", s.auth(s.requestExport))

	mux.Handle("GET /reports/billing", s.auth(s.billingReport))
	mux.Handle("GET /reports/load", s.auth(s.loadReport))
	mux.Handle("GET /reports/projects/{code}", s.auth(s.projectDashboard))

	mux.Handle("GET /members", s.auth(s.listMembers))
	mux.Handle("PUT /members/{id}/role", s.auth(s.setMemberRole))
	mux.Handle("PUT /members/{id}/permissions", s.auth(s.setMemberPermissions))
	mux.Handle("POST /members/{id}/copy-permissions", s.auth(s.copyMemberPermissions))

	mux.HandleFunc("POST /invites/{token}/accept", s.acceptInvite)
	mux.HandleFunc("POST /pay/{token}/note", s.leaveCustomerNote)

	return mux
}
