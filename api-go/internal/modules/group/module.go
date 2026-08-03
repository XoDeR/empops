// Package group implements employee groups and meeting management.
package group

import (
	"context"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	grouphttp "github.com/XoDeR/empops/api-go/internal/modules/group/adapter/http"
	"github.com/XoDeR/empops/api-go/pkg/companyauth"
	"github.com/XoDeR/empops/api-go/pkg/httpauth"
	"github.com/XoDeR/empops/api-go/pkg/jwt"
	"github.com/XoDeR/empops/api-go/pkg/module"
)

type Module struct{ pool *pgxpool.Pool; jwt *jwt.Manager; handler *grouphttp.Handler }
func New()*Module{return &Module{}}
func(m *Module)Name()string{return "group"}
func(m *Module)Dependencies()[]string{return []string{"company","employee"}}
func(m *Module)Initialize(_ context.Context,core *module.Core)error{m.pool,m.jwt=core.DB,core.JWT;m.handler=grouphttp.NewHandler(core.DB);return nil}
func(m *Module)RegisterRoutes(r chi.Router){r.Group(func(sub chi.Router){sub.Use(httpauth.RequireAuth(m.jwt));sub.Use(companyauth.RequireMember(m.pool));base:="/companies/{companyId}"
	sub.With(companyauth.RequirePermission("groups.view")).Get(base+"/groups",m.handler.ListGroups)
	sub.With(companyauth.RequirePermission("groups.manage")).Post(base+"/groups",m.handler.CreateGroup)
	sub.With(companyauth.RequirePermission("groups.view")).Get(base+"/groups/{groupId}",m.handler.ShowGroup)
	sub.With(companyauth.RequirePermission("groups.manage")).Patch(base+"/groups/{groupId}",m.handler.UpdateGroup)
	sub.With(companyauth.RequirePermission("groups.manage")).Delete(base+"/groups/{groupId}",m.handler.DeleteGroup)
	sub.With(companyauth.RequirePermission("groups.manage")).Post(base+"/groups/{groupId}/members/{employeeId}",m.handler.AddMember)
	sub.With(companyauth.RequirePermission("groups.manage")).Delete(base+"/groups/{groupId}/members/{employeeId}",m.handler.RemoveMember)
	sub.With(companyauth.RequirePermission("groups.view")).Get(base+"/groups/{groupId}/meetings",m.handler.ListMeetings)
	sub.With(companyauth.RequirePermission("groups.manage")).Post(base+"/groups/{groupId}/meetings",m.handler.CreateMeeting)
	sub.With(companyauth.RequirePermission("groups.view")).Get(base+"/groups/{groupId}/meetings/{meetingId}",m.handler.ShowMeeting)
	sub.With(companyauth.RequirePermission("groups.manage")).Patch(base+"/groups/{groupId}/meetings/{meetingId}",m.handler.UpdateMeeting)
	sub.With(companyauth.RequirePermission("groups.manage")).Delete(base+"/groups/{groupId}/meetings/{meetingId}",m.handler.DeleteMeeting)
	sub.With(companyauth.RequirePermission("groups.manage")).Post(base+"/groups/{groupId}/meetings/{meetingId}/attendees",m.handler.AddAttendee)
	sub.With(companyauth.RequirePermission("groups.manage")).Delete(base+"/groups/{groupId}/meetings/{meetingId}/attendees/{employeeId}",m.handler.RemoveAttendee)
	sub.With(companyauth.RequirePermission("groups.manage")).Post(base+"/groups/{groupId}/meetings/{meetingId}/agenda",m.handler.CreateAgendaItem)
	sub.With(companyauth.RequirePermission("groups.manage")).Patch(base+"/groups/{groupId}/meetings/{meetingId}/agenda/{agendaItemId}",m.handler.UpdateAgendaItem)
	sub.With(companyauth.RequirePermission("groups.manage")).Delete(base+"/groups/{groupId}/meetings/{meetingId}/agenda/{agendaItemId}",m.handler.DeleteAgendaItem)
	sub.With(companyauth.RequirePermission("groups.manage")).Post(base+"/groups/{groupId}/meetings/{meetingId}/agenda/{agendaItemId}/decisions",m.handler.CreateDecision)
	sub.With(companyauth.RequirePermission("groups.manage")).Patch(base+"/groups/{groupId}/meetings/{meetingId}/agenda/{agendaItemId}/decisions/{decisionId}",m.handler.UpdateDecision)
	sub.With(companyauth.RequirePermission("groups.manage")).Delete(base+"/groups/{groupId}/meetings/{meetingId}/agenda/{agendaItemId}/decisions/{decisionId}",m.handler.DeleteDecision)
})}
func(m *Module)Start(context.Context)error{return nil};func(m *Module)Stop(context.Context)error{return nil}
var _ module.IModule=(*Module)(nil)
