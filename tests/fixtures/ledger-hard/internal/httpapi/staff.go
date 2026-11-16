package httpapi

import (
	"net/http"

	"example.com/ledger/internal/service"
)

func (a *API) inviteEmployee(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Email        string `json:"email"`
		DepartmentID int64  `json:"department_id"`
	}
	if !decode(w, r, &body) {
		return
	}
	if err := a.Staff.Invite(r.Context(), actorFrom(r), body.Email, body.DepartmentID); err != nil {
		a.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusAccepted)
}

func (a *API) updatePayroll(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var body struct {
		BaseSalary  int64  `json:"base_salary"`
		TaxCode     string `json:"tax_code"`
		BankAccount string `json:"bank_account"`
	}
	if !decode(w, r, &body) {
		return
	}
	in := service.PayrollInput{BaseSalary: body.BaseSalary, TaxCode: body.TaxCode, BankAccount: body.BankAccount}
	if err := a.Staff.UpdatePayroll(r.Context(), actorFrom(r), id, in); err != nil {
		a.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) offboardEmployee(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	if err := a.Staff.Offboard(r.Context(), actorFrom(r), id); err != nil {
		a.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) changeHead(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var body struct {
		UserID int64 `json:"user_id"`
	}
	if !decode(w, r, &body) {
		return
	}
	if err := a.Staff.ChangeHead(r.Context(), actorFrom(r), id, body.UserID); err != nil {
		a.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
