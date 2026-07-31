// Package grow implements morale, one-on-ones, rate-your-manager, skills,
// e-coffee, and discipline (Step 8).
package grow

import (
	"context"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	growhttp "github.com/XoDeR/empops/api-go/internal/modules/grow/adapter/http"
	"github.com/XoDeR/empops/api-go/pkg/companyauth"
	"github.com/XoDeR/empops/api-go/pkg/httpauth"
	"github.com/XoDeR/empops/api-go/pkg/jwt"
	"github.com/XoDeR/empops/api-go/pkg/module"
)

type Module struct {
	pool    *pgxpool.Pool
	jwt     *jwt.Manager
	handler *growhttp.Handler
}

func New() *Module { return &Module{} }

func (m *Module) Name() string { return "grow" }

func (m *Module) Dependencies() []string {
	return []string{"company", "employee", "team", "media"}
}

func (m *Module) Initialize(_ context.Context, core *module.Core) error {
	m.pool, m.jwt = core.DB, core.JWT
	m.handler = growhttp.NewHandler(core.DB)
	return nil
}

func (m *Module) RegisterRoutes(r chi.Router) {
	r.Group(func(sub chi.Router) {
		sub.Use(httpauth.RequireAuth(m.jwt))
		sub.Use(companyauth.RequireMember(m.pool))
		base := "/companies/{companyId}"

		sub.Get(base+"/morale/today", m.handler.TodayMorale)
		sub.With(companyauth.RequirePermission("morale.log")).Post(base+"/morale", m.handler.LogMorale)
		sub.With(companyauth.RequirePermission("morale.view")).Get(base+"/morale/history/company", m.handler.CompanyMoraleHistory)
		sub.With(companyauth.RequirePermission("morale.view")).Get(base+"/morale/history/teams/{teamId}", m.handler.TeamMoraleHistory)
		sub.Get(base+"/employees/{employeeId}/morale", m.handler.EmployeeMorale)

		sub.Get(base+"/one-on-ones/me", m.handler.MyOneOnOnes)
		sub.Get(base+"/one-on-ones/manager", m.handler.ManagerOneOnOnes)
		sub.Get(base+"/one-on-ones/{entryId}", m.handler.ShowOneOnOne)
		sub.Post(base+"/one-on-ones/{entryId}/happened", m.handler.MarkOneOnOneHappened)
		sub.Post(base+"/one-on-ones/{entryId}/talking-points", m.handler.StoreTalkingPoint)
		sub.Post(base+"/one-on-ones/{entryId}/talking-points/{pointId}/toggle", m.handler.ToggleTalkingPoint)
		sub.Delete(base+"/one-on-ones/{entryId}/talking-points/{pointId}", m.handler.DestroyTalkingPoint)
		sub.Post(base+"/one-on-ones/{entryId}/action-items", m.handler.StoreActionItem)
		sub.Post(base+"/one-on-ones/{entryId}/action-items/{itemId}/toggle", m.handler.ToggleActionItem)
		sub.Delete(base+"/one-on-ones/{entryId}/action-items/{itemId}", m.handler.DestroyActionItem)
		sub.Post(base+"/one-on-ones/{entryId}/notes", m.handler.StoreNote)
		sub.Delete(base+"/one-on-ones/{entryId}/notes/{noteId}", m.handler.DestroyNote)

		sub.Get(base+"/rate-your-manager/pending", m.handler.PendingRateAnswers)
		sub.Post(base+"/rate-your-manager/answers/{answerId}", m.handler.SubmitRating)
		sub.Post(base+"/rate-your-manager/answers/{answerId}/comment", m.handler.CommentOnRating)
		sub.Get(base+"/employees/{employeeId}/rate-your-manager-surveys", m.handler.ManagerSurveys)

		sub.With(companyauth.RequirePermission("skills.view")).Get(base+"/skills", m.handler.ListSkills)
		sub.With(companyauth.RequirePermission("skills.view")).Get(base+"/skills/search", m.handler.SearchSkills)
		sub.With(companyauth.RequirePermission("skills.view")).Get(base+"/skills/{skillId}", m.handler.ShowSkill)
		sub.With(companyauth.RequirePermission("skills.manage")).Patch(base+"/skills/{skillId}", m.handler.UpdateSkill)
		sub.With(companyauth.RequirePermission("skills.manage")).Delete(base+"/skills/{skillId}", m.handler.DestroySkill)
		sub.Get(base+"/employees/{employeeId}/skills", m.handler.EmployeeSkills)
		sub.Post(base+"/employees/{employeeId}/skills", m.handler.AttachSkill)
		sub.Delete(base+"/employees/{employeeId}/skills/{skillId}", m.handler.DetachSkill)

		sub.With(companyauth.RequirePermission("e_coffee.manage")).Get(base+"/e-coffee", m.handler.GetECoffee)
		sub.With(companyauth.RequirePermission("e_coffee.manage")).Patch(base+"/e-coffee", m.handler.UpdateECoffee)
		sub.Get(base+"/e-coffee/current", m.handler.CurrentECoffee)
		sub.Post(base+"/e-coffee/matches/{matchId}/happened", m.handler.MarkECoffeeHappened)
		sub.Get(base+"/employees/{employeeId}/e-coffees", m.handler.EmployeeECoffeeHistory)

		sub.With(companyauth.RequirePermission("discipline.view")).Get(base+"/discipline-cases", m.handler.ListDisciplineCases)
		sub.With(companyauth.RequirePermission("discipline.manage")).Post(base+"/discipline-cases", m.handler.StoreDisciplineCase)
		sub.With(companyauth.RequirePermission("discipline.view")).Get(base+"/discipline-cases/{caseId}", m.handler.ShowDisciplineCase)
		sub.With(companyauth.RequirePermission("discipline.manage")).Post(base+"/discipline-cases/{caseId}/toggle", m.handler.ToggleDisciplineCase)
		sub.With(companyauth.RequirePermission("discipline.manage")).Delete(base+"/discipline-cases/{caseId}", m.handler.DestroyDisciplineCase)
		sub.With(companyauth.RequirePermission("discipline.manage")).Post(base+"/discipline-cases/{caseId}/events", m.handler.StoreDisciplineEvent)
		sub.With(companyauth.RequirePermission("discipline.manage")).Delete(base+"/discipline-cases/{caseId}/events/{eventId}", m.handler.DestroyDisciplineEvent)
		sub.With(companyauth.RequirePermission("discipline.manage")).Post(base+"/discipline-cases/{caseId}/events/{eventId}/files", m.handler.AttachDisciplineFile)
	})
}

func (m *Module) Start(context.Context) error { return nil }
func (m *Module) Stop(context.Context) error  { return nil }

var _ module.IModule = (*Module)(nil)
