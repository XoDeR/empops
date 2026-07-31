// Package project implements projects, boards, tasks, and collaboration features.
package project

import (
	"context"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	projecthttp "github.com/XoDeR/empops/api-go/internal/modules/project/adapter/http"
	"github.com/XoDeR/empops/api-go/pkg/companyauth"
	"github.com/XoDeR/empops/api-go/pkg/httpauth"
	"github.com/XoDeR/empops/api-go/pkg/jwt"
	"github.com/XoDeR/empops/api-go/pkg/module"
)

type Module struct {
	pool    *pgxpool.Pool
	jwt     *jwt.Manager
	handler *projecthttp.Handler
}

func New() *Module { return &Module{} }

func (m *Module) Name() string { return "project" }

func (m *Module) Dependencies() []string {
	return []string{"company", "employee", "team", "time"}
}

func (m *Module) Initialize(_ context.Context, core *module.Core) error {
	m.pool, m.jwt = core.DB, core.JWT
	m.handler = projecthttp.NewHandler(core.DB)
	return nil
}

func (m *Module) RegisterRoutes(r chi.Router) {
	requireAuth := httpauth.RequireAuth(m.jwt)
	requireMember := companyauth.RequireMember(m.pool)

	r.Group(func(sub chi.Router) {
		sub.Use(requireAuth)
		sub.Use(requireMember)
		base := "/companies/{companyId}"

		sub.With(companyauth.RequirePermission("projects.view")).Get(base+"/issue-types", m.handler.IssueTypes)

		sub.With(companyauth.RequirePermission("projects.view")).Get(base+"/projects", m.handler.ListProjects)
		sub.With(companyauth.RequirePermission("projects.create")).Post(base+"/projects", m.handler.CreateProject)
		sub.With(companyauth.RequirePermission("projects.view")).Get(base+"/projects/{projectId}", m.handler.ShowProject)
		sub.With(companyauth.RequirePermission("projects.update")).Patch(base+"/projects/{projectId}", m.handler.UpdateProject)
		sub.With(companyauth.RequirePermission("projects.delete")).Delete(base+"/projects/{projectId}", m.handler.DeleteProject)

		sub.With(companyauth.RequirePermission("projects.manage_members")).Post(base+"/projects/{projectId}/members/{employeeId}", m.handler.AddMember)
		sub.With(companyauth.RequirePermission("projects.manage_members")).Delete(base+"/projects/{projectId}/members/{employeeId}", m.handler.RemoveMember)
		sub.With(companyauth.RequirePermission("projects.manage_members")).Put(base+"/projects/{projectId}/lead", m.handler.SetLead)

		sub.With(companyauth.RequirePermission("projects.manage_members")).Post(base+"/projects/{projectId}/teams/{teamId}", m.handler.AttachTeam)
		sub.With(companyauth.RequirePermission("projects.manage_members")).Delete(base+"/projects/{projectId}/teams/{teamId}", m.handler.DetachTeam)

		sub.With(companyauth.RequirePermission("projects.view")).Get(base+"/projects/{projectId}/links", m.handler.ListLinks)
		sub.With(companyauth.RequirePermission("projects.update")).Post(base+"/projects/{projectId}/links", m.handler.CreateLink)
		sub.With(companyauth.RequirePermission("projects.update")).Patch(base+"/projects/{projectId}/links/{linkId}", m.handler.UpdateLink)
		sub.With(companyauth.RequirePermission("projects.update")).Delete(base+"/projects/{projectId}/links/{linkId}", m.handler.DeleteLink)

		sub.With(companyauth.RequirePermission("projects.view")).Get(base+"/projects/{projectId}/statuses", m.handler.ListStatuses)
		sub.With(companyauth.RequirePermission("projects.update")).Post(base+"/projects/{projectId}/statuses", m.handler.CreateStatus)

		sub.With(companyauth.RequirePermission("projects.view")).Get(base+"/projects/{projectId}/files", m.handler.ListFiles)
		sub.With(companyauth.RequirePermission("projects.update")).Post(base+"/projects/{projectId}/files", m.handler.AttachFile)
		sub.With(companyauth.RequirePermission("projects.update")).Delete(base+"/projects/{projectId}/files/{mediaId}", m.handler.DeleteFile)

		sub.With(companyauth.RequirePermission("projects.view")).Get(base+"/projects/{projectId}/messages", m.handler.ListMessages)
		sub.With(companyauth.RequirePermission("projects.update")).Post(base+"/projects/{projectId}/messages", m.handler.CreateMessage)
		sub.With(companyauth.RequirePermission("projects.view")).Get(base+"/projects/{projectId}/messages/{messageId}", m.handler.ShowMessage)
		sub.With(companyauth.RequirePermission("projects.update")).Patch(base+"/projects/{projectId}/messages/{messageId}", m.handler.UpdateMessage)
		sub.With(companyauth.RequirePermission("projects.update")).Delete(base+"/projects/{projectId}/messages/{messageId}", m.handler.DeleteMessage)
		sub.With(companyauth.RequirePermission("projects.update")).Post(base+"/projects/{projectId}/messages/{messageId}/comments", m.handler.CreateMessageComment)
		sub.With(companyauth.RequirePermission("projects.update")).Patch(base+"/projects/{projectId}/messages/{messageId}/comments/{commentId}", m.handler.UpdateMessageComment)
		sub.With(companyauth.RequirePermission("projects.update")).Delete(base+"/projects/{projectId}/messages/{messageId}/comments/{commentId}", m.handler.DeleteMessageComment)

		sub.With(companyauth.RequirePermission("projects.view")).Get(base+"/projects/{projectId}/decisions", m.handler.ListDecisions)
		sub.With(companyauth.RequirePermission("projects.update")).Post(base+"/projects/{projectId}/decisions", m.handler.CreateDecision)
		sub.With(companyauth.RequirePermission("projects.update")).Delete(base+"/projects/{projectId}/decisions/{decisionId}", m.handler.DeleteDecision)

		sub.With(companyauth.RequirePermission("projects.view")).Get(base+"/projects/{projectId}/task-lists", m.handler.ListTaskLists)
		sub.With(companyauth.RequirePermission("projects.update")).Post(base+"/projects/{projectId}/task-lists", m.handler.CreateTaskList)
		sub.With(companyauth.RequirePermission("projects.update")).Patch(base+"/projects/{projectId}/task-lists/{listId}", m.handler.UpdateTaskList)
		sub.With(companyauth.RequirePermission("projects.update")).Delete(base+"/projects/{projectId}/task-lists/{listId}", m.handler.DeleteTaskList)

		sub.With(companyauth.RequirePermission("projects.update")).Post(base+"/projects/{projectId}/tasks", m.handler.CreateTask)
		sub.With(companyauth.RequirePermission("projects.view")).Get(base+"/projects/{projectId}/tasks/{taskId}", m.handler.ShowTask)
		sub.With(companyauth.RequirePermission("projects.update")).Patch(base+"/projects/{projectId}/tasks/{taskId}", m.handler.UpdateTask)
		sub.With(companyauth.RequirePermission("projects.update")).Delete(base+"/projects/{projectId}/tasks/{taskId}", m.handler.DeleteTask)
		sub.With(companyauth.RequirePermission("projects.update")).Post(base+"/projects/{projectId}/tasks/{taskId}/toggle", m.handler.ToggleTask)
		sub.With(companyauth.RequirePermission("projects.view")).Get(base+"/projects/{projectId}/tasks/{taskId}/time-entries", m.handler.TaskTimeEntries)
		sub.With(companyauth.RequirePermission("projects.update")).Post(base+"/projects/{projectId}/tasks/{taskId}/comments", m.handler.CreateTaskComment)
		sub.With(companyauth.RequirePermission("projects.update")).Patch(base+"/projects/{projectId}/tasks/{taskId}/comments/{commentId}", m.handler.UpdateTaskComment)
		sub.With(companyauth.RequirePermission("projects.update")).Delete(base+"/projects/{projectId}/tasks/{taskId}/comments/{commentId}", m.handler.DeleteTaskComment)

		sub.With(companyauth.RequirePermission("projects.view")).Get(base+"/projects/{projectId}/boards", m.handler.ListBoards)
		sub.With(companyauth.RequirePermission("projects.update")).Post(base+"/projects/{projectId}/boards", m.handler.CreateBoard)
		sub.With(companyauth.RequirePermission("projects.view")).Get(base+"/projects/{projectId}/boards/{boardId}", m.handler.ShowBoard)
		sub.With(companyauth.RequirePermission("projects.update")).Patch(base+"/projects/{projectId}/boards/{boardId}", m.handler.UpdateBoard)
		sub.With(companyauth.RequirePermission("projects.update")).Delete(base+"/projects/{projectId}/boards/{boardId}", m.handler.DeleteBoard)
		sub.With(companyauth.RequirePermission("projects.view")).Get(base+"/projects/{projectId}/boards/{boardId}/backlog", m.handler.Backlog)
		sub.With(companyauth.RequirePermission("projects.update")).Post(base+"/projects/{projectId}/boards/{boardId}/sprints/{sprintId}/start", m.handler.StartSprint)
		sub.With(companyauth.RequirePermission("projects.update")).Post(base+"/projects/{projectId}/boards/{boardId}/sprints/{sprintId}/toggle", m.handler.ToggleSprint)
		sub.With(companyauth.RequirePermission("projects.update")).Post(base+"/projects/{projectId}/boards/{boardId}/sprints/{sprintId}/issues", m.handler.CreateIssue)
		sub.With(companyauth.RequirePermission("projects.update")).Post(base+"/projects/{projectId}/boards/{boardId}/sprints/{sprintId}/issues/{issueId}/order", m.handler.ReorderIssue)
		sub.With(companyauth.RequirePermission("projects.update")).Delete(base+"/projects/{projectId}/boards/{boardId}/sprints/{sprintId}/issues/{issueId}", m.handler.DeleteIssue)
		sub.With(companyauth.RequirePermission("projects.update")).Post(base+"/projects/{projectId}/boards/{boardId}/issues/{issueId}/assignees", m.handler.AddIssueAssignee)
		sub.With(companyauth.RequirePermission("projects.update")).Delete(base+"/projects/{projectId}/boards/{boardId}/issues/{issueId}/assignees/{assigneeId}", m.handler.RemoveIssueAssignee)
		sub.With(companyauth.RequirePermission("projects.update")).Post(base+"/projects/{projectId}/boards/{boardId}/issues/{issueId}/points", m.handler.SetIssuePoints)
	})
}

func (m *Module) Start(context.Context) error { return nil }

func (m *Module) Stop(context.Context) error { return nil }

var _ module.IModule = (*Module)(nil)
