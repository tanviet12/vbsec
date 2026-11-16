package repo

import (
	"context"
	"database/sql"

	"example.com/tallyboard/internal/model"
)

type Comments struct {
	db *DB
}

func NewComments(db *DB) *Comments { return &Comments{db: db} }

const commentCols = `c.id, c.task_id, c.author_id, u.name, m.role, c.body, c.visibility, c.created_at`

const commentJoins = ` FROM comments c
	JOIN members m ON m.id = c.author_id AND m.company_id = c.company_id
	JOIN users u ON u.id = m.user_id`

func collectComments(rows *sql.Rows) ([]model.Comment, error) {
	defer rows.Close()
	var out []model.Comment
	for rows.Next() {
		var c model.Comment
		if err := rows.Scan(&c.ID, &c.TaskID, &c.AuthorID, &c.AuthorName, &c.AuthorRole, &c.Body, &c.Visibility, &c.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// ListForTask returns the thread of a task. Client guests only get the
// comments shared with them.
func (r *Comments) ListForTask(ctx context.Context, companyID, taskID int64, teamToo bool) ([]model.Comment, error) {
	q := "SELECT " + commentCols + commentJoins + " WHERE c.company_id = ? AND c.task_id = ?"
	if !teamToo {
		q += " AND c.visibility = 'client'"
	}
	rows, err := r.db.Query(ctx, q+" ORDER BY c.id", companyID, taskID)
	if err != nil {
		return nil, err
	}
	return collectComments(rows)
}

func (r *Comments) RecentForProject(ctx context.Context, companyID, projectID int64, limit int) ([]model.Comment, error) {
	rows, err := r.db.Query(ctx,
		"SELECT "+commentCols+commentJoins+`
		   JOIN tasks t ON t.id = c.task_id AND t.company_id = c.company_id
		  WHERE c.company_id = ? AND t.project_id = ?
		  ORDER BY c.id DESC LIMIT ?`, companyID, projectID, limit)
	if err != nil {
		return nil, err
	}
	return collectComments(rows)
}

func (r *Comments) Insert(ctx context.Context, companyID int64, c *model.Comment) error {
	res, err := r.db.Exec(ctx,
		"INSERT INTO comments (company_id, task_id, author_id, body, visibility) VALUES (?, ?, ?, ?, ?)",
		companyID, c.TaskID, c.AuthorID, c.Body, c.Visibility)
	if err != nil {
		return err
	}
	c.ID, _ = res.LastInsertId()
	return nil
}
