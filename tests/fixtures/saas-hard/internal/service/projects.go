package service

import (
	"context"
	"strings"
	"time"

	"example.com/tallyboard/internal/authz"
	"example.com/tallyboard/internal/model"
	"example.com/tallyboard/internal/repo"
	"example.com/tallyboard/internal/tenant"
)

type Projects struct {
	Repo     *repo.Projects
	Comments *repo.Comments
	Confirm  *Confirmations
}

const ActionDeleteProject = "project.delete"

// visible loads a project the actor may open. Staff see every project of
// their company; client guests only the ones they were added to.
func (s *Projects) visible(ctx context.Context, a *tenant.Actor, id int64) (*model.Project, error) {
	p, err := s.Repo.Get(ctx, a.CompanyID, id)
	if err != nil {
		return nil, err
	}
	if authz.IsStaff(a) {
		return p, nil
	}
	ok, err := s.Repo.IsParticipant(ctx, a.CompanyID, id, a.MemberID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, repo.ErrNotFound
	}
	return p, nil
}

func (s *Projects) List(ctx context.Context, a *tenant.Actor, sort string, archived bool) ([]*model.Project, error) {
	all, err := s.Repo.List(ctx, a.CompanyID, sort, archived)
	if err != nil || authz.IsStaff(a) {
		return all, err
	}
	var mine []*model.Project
	for _, p := range all {
		ok, err := s.Repo.IsParticipant(ctx, a.CompanyID, p.ID, a.MemberID)
		if err != nil {
			return nil, err
		}
		if ok {
			mine = append(mine, p)
		}
	}
	return mine, nil
}

func (s *Projects) Get(ctx context.Context, a *tenant.Actor, id int64) (*model.Project, []model.Task, error) {
	p, err := s.visible(ctx, a, id)
	if err != nil {
		return nil, nil, err
	}
	tasks, err := s.Repo.Tasks(ctx, a.CompanyID, id)
	return p, tasks, err
}

func (s *Projects) Archive(ctx context.Context, a *tenant.Actor, id int64) error {
	p, err := s.Repo.Get(ctx, a.CompanyID, id)
	if err != nil {
		return err
	}
	if !authz.CanManageProject(a, p) {
		return ErrForbidden
	}
	now := time.Now().UTC()
	return s.Repo.SetArchived(ctx, a.CompanyID, id, &now)
}

func (s *Projects) Restore(ctx context.Context, a *tenant.Actor, id int64) error {
	p, err := s.Repo.Get(ctx, a.CompanyID, id)
	if err != nil {
		return err
	}
	if !authz.CanManageProject(a, p) {
		return ErrForbidden
	}
	return s.Repo.SetArchived(ctx, a.CompanyID, id, nil)
}

func (s *Projects) RequestDelete(ctx context.Context, a *tenant.Actor, id int64) (time.Time, error) {
	p, err := s.Repo.Get(ctx, a.CompanyID, id)
	if err != nil {
		return time.Time{}, err
	}
	if !authz.CanManageProject(a, p) {
		return time.Time{}, ErrForbidden
	}
	return s.Confirm.Issue(ctx, a, ActionDeleteProject, id, "delete project "+p.Code)
}

func (s *Projects) Delete(ctx context.Context, a *tenant.Actor, id int64, code string) error {
	p, err := s.Repo.Get(ctx, a.CompanyID, id)
	if err != nil {
		return err
	}
	if !authz.CanManageProject(a, p) {
		return ErrForbidden
	}
	if err := s.Confirm.Verify(ctx, a, ActionDeleteProject, id, code); err != nil {
		return err
	}
	return s.Repo.Delete(ctx, a.CompanyID, id)
}

// AddExpense books a cost against a project. Any participant can submit
// expenses; the project is returned so callers can refresh their view.
func (s *Projects) AddExpense(ctx context.Context, a *tenant.Actor, projectID, amountCents int64, note string) (*model.Expense, *model.Project, int64, error) {
	p, err := s.visible(ctx, a, projectID)
	if err != nil {
		return nil, nil, 0, err
	}
	if !authz.IsStaff(a) || p.ArchivedAt != nil {
		return nil, nil, 0, ErrForbidden
	}
	note = strings.TrimSpace(note)
	if amountCents <= 0 || amountCents > 500_000_00 || len(note) > 255 {
		return nil, nil, 0, ErrInvalid
	}
	e := &model.Expense{ProjectID: projectID, MemberID: a.MemberID, AmountCents: amountCents, Note: note}
	if err := s.Repo.AddExpense(ctx, a.CompanyID, e); err != nil {
		return nil, nil, 0, err
	}
	spent, err := s.Repo.SpentCents(ctx, a.CompanyID, projectID)
	if err != nil {
		return nil, nil, 0, err
	}
	return e, p, spent, nil
}

func (s *Projects) ListComments(ctx context.Context, a *tenant.Actor, taskID int64) ([]model.Comment, error) {
	projectID, err := s.Repo.TaskProject(ctx, a.CompanyID, taskID)
	if err != nil {
		return nil, err
	}
	if _, err := s.visible(ctx, a, projectID); err != nil {
		return nil, err
	}
	return s.Comments.ListForTask(ctx, a.CompanyID, taskID, authz.IsStaff(a))
}

// AddComment posts to a task thread. Guests always post to the shared
// thread; staff choose between the team thread and the shared one.
func (s *Projects) AddComment(ctx context.Context, a *tenant.Actor, taskID int64, body, visibility string) (*model.Comment, error) {
	projectID, err := s.Repo.TaskProject(ctx, a.CompanyID, taskID)
	if err != nil {
		return nil, err
	}
	if _, err := s.visible(ctx, a, projectID); err != nil {
		return nil, err
	}
	body = strings.TrimSpace(body)
	if body == "" || len(body) > 4000 {
		return nil, ErrInvalid
	}
	if !authz.IsStaff(a) {
		visibility = "client"
	} else if visibility != "client" {
		visibility = "team"
	}
	c := &model.Comment{TaskID: taskID, AuthorID: a.MemberID, Body: body, Visibility: visibility, CreatedAt: time.Now().UTC()}
	if err := s.Comments.Insert(ctx, a.CompanyID, c); err != nil {
		return nil, err
	}
	return c, nil
}
