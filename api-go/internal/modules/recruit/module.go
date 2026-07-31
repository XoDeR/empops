// Package recruit implements ATS job openings, candidates, and public careers.
package recruit

import (
	"context"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	recruithttp "github.com/XoDeR/empops/api-go/internal/modules/recruit/adapter/http"
	"github.com/XoDeR/empops/api-go/pkg/companyauth"
	"github.com/XoDeR/empops/api-go/pkg/httpauth"
	"github.com/XoDeR/empops/api-go/pkg/jwt"
	"github.com/XoDeR/empops/api-go/pkg/module"
)

type Module struct {
	pool    *pgxpool.Pool
	jwt     *jwt.Manager
	handler *recruithttp.Handler
}

func New() *Module { return &Module{} }

func (m *Module) Name() string { return "recruit" }

func (m *Module) Dependencies() []string {
	return []string{"company", "employee", "team"}
}

func (m *Module) Initialize(_ context.Context, core *module.Core) error {
	m.pool, m.jwt = core.DB, core.JWT
	m.handler = recruithttp.NewHandler(core.DB)
	return nil
}

func (m *Module) RegisterRoutes(r chi.Router) {
	// Public careers (no auth).
	r.Get("/jobs", m.handler.ListCompanies)
	r.Get("/jobs/{companySlug}", m.handler.ListCompanyJobs)
	r.Get("/jobs/{companySlug}/jobs/{jobSlug}", m.handler.ShowJob)
	r.Post("/jobs/{companySlug}/jobs/{jobSlug}", m.handler.Apply)
	r.Get("/jobs/{companySlug}/jobs/{jobSlug}/apply/{candidateUuid}/files", m.handler.PublicListFiles)
	r.Post("/jobs/{companySlug}/jobs/{jobSlug}/apply/{candidateUuid}/files", m.handler.PublicAttachFile)
	r.Delete("/jobs/{companySlug}/jobs/{jobSlug}/apply/{candidateUuid}/files/{mediaId}", m.handler.PublicDeleteFile)
	r.Post("/jobs/{companySlug}/jobs/{jobSlug}/apply/{candidateUuid}/complete", m.handler.CompleteApplication)
	r.Delete("/jobs/{companySlug}/jobs/{jobSlug}/apply/{candidateUuid}", m.handler.AbandonApplication)

	requireAuth := httpauth.RequireAuth(m.jwt)
	requireMember := companyauth.RequireMember(m.pool)

	r.Group(func(sub chi.Router) {
		sub.Use(requireAuth)
		sub.Use(requireMember)
		base := "/companies/{companyId}"

		sub.With(companyauth.RequirePermission("recruiting.manage_templates")).Get(base+"/recruiting/stage-templates", m.handler.ListTemplates)
		sub.With(companyauth.RequirePermission("recruiting.manage_templates")).Post(base+"/recruiting/stage-templates", m.handler.CreateTemplate)
		sub.With(companyauth.RequirePermission("recruiting.manage_templates")).Get(base+"/recruiting/stage-templates/{templateId}", m.handler.ShowTemplate)
		sub.With(companyauth.RequirePermission("recruiting.manage_templates")).Patch(base+"/recruiting/stage-templates/{templateId}", m.handler.UpdateTemplate)
		sub.With(companyauth.RequirePermission("recruiting.manage_templates")).Delete(base+"/recruiting/stage-templates/{templateId}", m.handler.DeleteTemplate)

		sub.With(companyauth.RequirePermission("recruiting.manage_templates")).Post(base+"/recruiting/stage-templates/{templateId}/stages", m.handler.CreateStage)
		sub.With(companyauth.RequirePermission("recruiting.manage_templates")).Patch(base+"/recruiting/stage-templates/{templateId}/stages/{stageId}", m.handler.UpdateStage)
		sub.With(companyauth.RequirePermission("recruiting.manage_templates")).Delete(base+"/recruiting/stage-templates/{templateId}/stages/{stageId}", m.handler.DeleteStage)

		sub.Get(base+"/job-openings", m.handler.ListOpenings)
		sub.With(companyauth.RequirePermission("recruiting.create")).Post(base+"/job-openings", m.handler.CreateOpening)
		sub.Get(base+"/job-openings/{jobOpeningId}", m.handler.ShowOpening)
		sub.With(companyauth.RequirePermission("recruiting.update")).Patch(base+"/job-openings/{jobOpeningId}", m.handler.UpdateOpening)
		sub.With(companyauth.RequirePermission("recruiting.delete")).Delete(base+"/job-openings/{jobOpeningId}", m.handler.DeleteOpening)
		sub.With(companyauth.RequirePermission("recruiting.update")).Post(base+"/job-openings/{jobOpeningId}/toggle", m.handler.ToggleOpening)

		sub.With(companyauth.RequirePermission("recruiting.update")).Post(base+"/job-openings/{jobOpeningId}/sponsors/{employeeId}", m.handler.AddSponsor)
		sub.With(companyauth.RequirePermission("recruiting.update")).Delete(base+"/job-openings/{jobOpeningId}/sponsors/{employeeId}", m.handler.RemoveSponsor)

		sub.Get(base+"/job-openings/{jobOpeningId}/candidates", m.handler.ListCandidates)
		sub.Get(base+"/job-openings/{jobOpeningId}/candidates/{candidateId}", m.handler.ShowCandidate)

		sub.Post(base+"/job-openings/{jobOpeningId}/candidates/{candidateId}/stages/{stageId}", m.handler.ProcessStage)

		sub.Get(base+"/job-openings/{jobOpeningId}/candidates/{candidateId}/stages/{stageId}/notes", m.handler.ListNotes)
		sub.Post(base+"/job-openings/{jobOpeningId}/candidates/{candidateId}/stages/{stageId}/notes", m.handler.CreateNote)
		sub.Patch(base+"/job-openings/{jobOpeningId}/candidates/{candidateId}/stages/{stageId}/notes/{noteId}", m.handler.UpdateNote)
		sub.Delete(base+"/job-openings/{jobOpeningId}/candidates/{candidateId}/stages/{stageId}/notes/{noteId}", m.handler.DeleteNote)

		sub.Post(base+"/job-openings/{jobOpeningId}/candidates/{candidateId}/stages/{stageId}/participants", m.handler.AddParticipant)
		sub.Delete(base+"/job-openings/{jobOpeningId}/candidates/{candidateId}/stages/{stageId}/participants/{participantId}", m.handler.RemoveParticipant)

		sub.With(companyauth.RequirePermission("recruiting.hire")).Post(base+"/job-openings/{jobOpeningId}/candidates/{candidateId}/hire", m.handler.Hire)

		sub.Get(base+"/job-openings/{jobOpeningId}/candidates/{candidateId}/files", m.handler.ListFiles)
		sub.With(companyauth.RequirePermission("recruiting.update")).Post(base+"/job-openings/{jobOpeningId}/candidates/{candidateId}/files", m.handler.AttachFile)
		sub.With(companyauth.RequirePermission("recruiting.update")).Delete(base+"/job-openings/{jobOpeningId}/candidates/{candidateId}/files/{mediaId}", m.handler.DeleteFile)
	})
}

func (m *Module) Start(context.Context) error { return nil }

func (m *Module) Stop(context.Context) error { return nil }

var _ module.IModule = (*Module)(nil)
