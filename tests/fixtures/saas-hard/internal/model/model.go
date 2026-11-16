package model

import "time"

type Member struct {
	ID          int64      `json:"id"`
	CompanyID   int64      `json:"-"`
	UserID      int64      `json:"user_id"`
	Name        string     `json:"name"`
	Email       string     `json:"email"`
	Role        string     `json:"role"`
	Status      string     `json:"status"`
	InviteToken string     `json:"invite_token,omitempty"`
	InvitedAt   *time.Time `json:"invited_at,omitempty"`
}

type Project struct {
	ID              int64      `json:"id"`
	CompanyID       int64      `json:"-"`
	Code            string     `json:"code"`
	Name            string     `json:"name"`
	ManagerID       int64      `json:"manager_id"`
	BudgetCents     int64      `json:"budget_cents"`
	HourlyRateCents int64      `json:"hourly_rate_cents"`
	Notes           string     `json:"notes"`
	ArchivedAt      *time.Time `json:"archived_at,omitempty"`
}

type Task struct {
	ID         int64  `json:"id"`
	ProjectID  int64  `json:"project_id"`
	Title      string `json:"title"`
	Status     string `json:"status"`
	AssigneeID *int64 `json:"assignee_id,omitempty"`
}

type Comment struct {
	ID         int64     `json:"id"`
	TaskID     int64     `json:"task_id"`
	AuthorID   int64     `json:"author_id"`
	AuthorName string    `json:"author_name"`
	AuthorRole string    `json:"author_role"`
	Body       string    `json:"body"`
	Visibility string    `json:"visibility"`
	CreatedAt  time.Time `json:"created_at"`
}

type Invoice struct {
	ID            int64      `json:"id"`
	CompanyID     int64      `json:"-"`
	ProjectCode   string     `json:"project_code"`
	Number        string     `json:"number"`
	CustomerName  string     `json:"customer_name"`
	CustomerEmail string     `json:"customer_email"`
	CustomerNote  string     `json:"customer_note,omitempty"`
	AmountCents   int64      `json:"amount_cents"`
	Status        string     `json:"status"`
	DueDate       time.Time  `json:"due_date"`
	IssuedAt      time.Time  `json:"issued_at"`
	VoidedAt      *time.Time `json:"voided_at,omitempty"`
}

type Expense struct {
	ID          int64     `json:"id"`
	ProjectID   int64     `json:"project_id"`
	MemberID    int64     `json:"member_id"`
	AmountCents int64     `json:"amount_cents"`
	Note        string    `json:"note"`
	CreatedAt   time.Time `json:"created_at"`
}

type ExportJob struct {
	ID             int64
	CompanyID      int64
	RequestedBy    int64
	RequesterEmail string
	Kind           string
	From           *time.Time
	To             *time.Time
}

type ProjectBilling struct {
	ProjectID   int64  `json:"project_id"`
	Code        string `json:"code"`
	Name        string `json:"name"`
	BilledCents int64  `json:"billed_cents"`
	Invoices    int    `json:"invoices"`
}

type MemberLoad struct {
	MemberID  int64  `json:"member_id"`
	Name      string `json:"name"`
	OpenTasks int    `json:"open_tasks"`
}
