package http

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/XoDeR/empops/api-go/pkg/companyauth"
	"github.com/XoDeR/empops/api-go/pkg/response"
	"github.com/XoDeR/empops/api-go/pkg/uuidv7"
)

type Handler struct{ pool *pgxpool.Pool }
func NewHandler(pool *pgxpool.Pool) *Handler { return &Handler{pool: pool} }

type timesheet struct {
	ID, CompanyID, EmployeeID, Status string
	StartedAt, EndedAt time.Time
	ApproverID *string
	ApprovedAt *time.Time
}

func (h *Handler) Timesheet(w http.ResponseWriter, r *http.Request) {
	member, _ := companyauth.MemberFromContext(r.Context())
	employeeID := r.URL.Query().Get("employeeId")
	if employeeID == "" { employeeID = r.URL.Query().Get("employee_id") }
	if employeeID == "" { employeeID = member.EmployeeID }
	if employeeID != member.EmployeeID && !member.HasPermission("timesheets.view") {
		response.Fail(w, http.StatusForbidden, "Forbidden", nil); return
	}
	date := time.Now().UTC()
	if raw := r.URL.Query().Get("date"); raw != "" {
		var err error
		date, err = time.Parse("2006-01-02", raw)
		if err != nil { response.Fail(w, 422, "invalid date", nil); return }
	}
	var exists bool
	if err := h.pool.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM employees WHERE id=$1 AND company_id=$2)`, employeeID, member.CompanyID).Scan(&exists); err != nil || !exists {
		response.Fail(w, http.StatusNotFound, "Employee not found", nil); return
	}
	ts, err := h.createOrGet(r.Context(), member.CompanyID, employeeID, date)
	if err != nil { response.Fail(w, 500, "timesheet lookup failed", err.Error()); return }
	h.writeTimesheet(w, r, ts)
}

func week(date time.Time) (time.Time, time.Time) {
	date = time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC)
	offset := (int(date.Weekday()) + 6) % 7
	start := date.AddDate(0, 0, -offset)
	return start, start.AddDate(0, 0, 6)
}

func (h *Handler) createOrGet(ctx context.Context, companyID, employeeID string, date time.Time) (timesheet, error) {
	start, end := week(date)
	_, err := h.pool.Exec(ctx, `INSERT INTO timesheets
		(id,company_id,employee_id,started_at,ended_at,status) VALUES ($1,$2,$3,$4,$5,'open')
		ON CONFLICT (employee_id,started_at) DO NOTHING`, uuidv7.New(), companyID, employeeID, start, end)
	if err != nil { return timesheet{}, err }
	return h.find(ctx, companyID, "", employeeID, start)
}

func (h *Handler) find(ctx context.Context, companyID, id, employeeID string, start time.Time) (timesheet, error) {
	q := `SELECT id,company_id,employee_id,started_at,ended_at,status,approver_id,approved_at FROM timesheets WHERE company_id=$1`
	args := []interface{}{companyID}
	if id != "" { q += ` AND id=$2`; args = append(args, id) } else { q += ` AND employee_id=$2 AND started_at=$3`; args = append(args, employeeID, start) }
	var t timesheet
	err := h.pool.QueryRow(ctx, q, args...).Scan(&t.ID,&t.CompanyID,&t.EmployeeID,&t.StartedAt,&t.EndedAt,&t.Status,&t.ApproverID,&t.ApprovedAt)
	return t, err
}

func (h *Handler) Show(w http.ResponseWriter, r *http.Request) {
	member, _ := companyauth.MemberFromContext(r.Context())
	t, err := h.find(r.Context(), member.CompanyID, chi.URLParam(r,"timesheetId"), "", time.Time{})
	if err == pgx.ErrNoRows { response.Fail(w,404,"Timesheet not found",nil); return }
	if err != nil { response.Fail(w,500,"timesheet lookup failed",err.Error()); return }
	if t.EmployeeID != member.EmployeeID && !member.HasPermission("timesheets.view") { response.Fail(w,403,"Forbidden",nil); return }
	h.writeTimesheet(w,r,t)
}

type entryRequest struct {
	HappenedAt string `json:"happened_at"`
	Duration int `json:"duration"`
	Description *string `json:"description"`
}

func (h *Handler) UpsertEntry(w http.ResponseWriter, r *http.Request) {
	member, _ := companyauth.MemberFromContext(r.Context())
	t, err := h.find(r.Context(), member.CompanyID, chi.URLParam(r,"timesheetId"), "", time.Time{})
	if err != nil { response.Fail(w,404,"Timesheet not found",nil); return }
	if t.EmployeeID != member.EmployeeID { response.Fail(w,403,"Forbidden",nil); return }
	if t.Status != "open" && t.Status != "rejected" { response.Fail(w,409,"Only open or rejected timesheets can be edited",nil); return }
	var req entryRequest
	if json.NewDecoder(r.Body).Decode(&req) != nil { response.Fail(w,400,"Invalid JSON body",nil); return }
	day, err := time.Parse("2006-01-02",req.HappenedAt)
	if err != nil || req.Duration < 1 || req.Duration > 1440 { response.Fail(w,422,"invalid entry",nil); return }
	if day.Before(t.StartedAt) || day.After(t.EndedAt) { response.Fail(w,422,"Entry date must be within the timesheet week",nil); return }
	var other int
	_ = h.pool.QueryRow(r.Context(),`SELECT COALESCE(SUM(duration),0) FROM time_tracking_entries WHERE timesheet_id=$1 AND happened_at<>$2`,t.ID,day).Scan(&other)
	if other+req.Duration > 10080 { response.Fail(w,422,"Weekly duration cannot exceed 10080 minutes",nil); return }
	_, err = h.pool.Exec(r.Context(),`INSERT INTO time_tracking_entries
		(id,timesheet_id,employee_id,duration,happened_at,description) VALUES($1,$2,$3,$4,$5,$6)
		ON CONFLICT(timesheet_id,employee_id,happened_at) DO UPDATE SET duration=EXCLUDED.duration,description=EXCLUDED.description,updated_at=now()`,
		uuidv7.New(),t.ID,t.EmployeeID,req.Duration,day,req.Description)
	if err != nil { response.Fail(w,500,"save entry failed",err.Error()); return }
	if t.Status=="rejected" {
		_, _ = h.pool.Exec(r.Context(),`UPDATE timesheets SET status='open',approver_id=NULL,approved_at=NULL,updated_at=now() WHERE id=$1`,t.ID)
		t.Status="open"; t.ApproverID=nil; t.ApprovedAt=nil
	}
	h.writeTimesheet(w,r,t)
}

func (h *Handler) DeleteEntry(w http.ResponseWriter, r *http.Request) {
	member,_ := companyauth.MemberFromContext(r.Context())
	t,err := h.find(r.Context(),member.CompanyID,chi.URLParam(r,"timesheetId"),"",time.Time{})
	if err != nil { response.Fail(w,404,"Timesheet not found",nil); return }
	if t.EmployeeID!=member.EmployeeID { response.Fail(w,403,"Forbidden",nil); return }
	if t.Status!="open"&&t.Status!="rejected" { response.Fail(w,409,"Only open or rejected timesheets can be edited",nil); return }
	tag,err:=h.pool.Exec(r.Context(),`DELETE FROM time_tracking_entries WHERE id=$1 AND timesheet_id=$2`,chi.URLParam(r,"entryId"),t.ID)
	if err!=nil { response.Fail(w,500,"delete entry failed",err.Error()); return }
	if tag.RowsAffected()==0 { response.Fail(w,404,"Entry not found",nil); return }
	h.writeTimesheet(w,r,t)
}

func (h *Handler) Submit(w http.ResponseWriter,r *http.Request) {
	member,_:=companyauth.MemberFromContext(r.Context())
	t,err:=h.find(r.Context(),member.CompanyID,chi.URLParam(r,"timesheetId"),"",time.Time{})
	if err!=nil { response.Fail(w,404,"Timesheet not found",nil); return }
	if t.EmployeeID!=member.EmployeeID { response.Fail(w,403,"Forbidden",nil); return }
	if t.Status!="open"&&t.Status!="rejected" { response.Fail(w,409,"Timesheet cannot be submitted from its current status",nil); return }
	_,err=h.pool.Exec(r.Context(),`UPDATE timesheets SET status='ready_to_submit',approver_id=NULL,approved_at=NULL,updated_at=now() WHERE id=$1`,t.ID)
	if err!=nil { response.Fail(w,500,"submit timesheet failed",err.Error()); return }
	t.Status="ready_to_submit"; t.ApproverID=nil; t.ApprovedAt=nil
	h.writeTimesheet(w,r,t)
}

func (h *Handler) Approve(w http.ResponseWriter,r *http.Request){ h.decide(w,r,true) }
func (h *Handler) Reject(w http.ResponseWriter,r *http.Request){ h.decide(w,r,false) }
func (h *Handler) decide(w http.ResponseWriter,r *http.Request,approve bool){
	member,_:=companyauth.MemberFromContext(r.Context())
	t,err:=h.find(r.Context(),member.CompanyID,chi.URLParam(r,"timesheetId"),"",time.Time{})
	if err!=nil { response.Fail(w,404,"Timesheet not found",nil); return }
	if !hasRole(member,"administrator","hr") {
		var manages bool
		_ = h.pool.QueryRow(r.Context(),`SELECT EXISTS(SELECT 1 FROM direct_reports WHERE company_id=$1 AND manager_id=$2 AND employee_id=$3)`,member.CompanyID,member.EmployeeID,t.EmployeeID).Scan(&manages)
		if !manages { response.Fail(w,403,"Forbidden",nil); return }
	}
	if t.Status!="ready_to_submit" { response.Fail(w,409,"Only submitted timesheets can be reviewed",nil); return }
	status:="rejected"; var approvedAt *time.Time
	if approve { status="approved"; now:=time.Now().UTC(); approvedAt=&now }
	_,err=h.pool.Exec(r.Context(),`UPDATE timesheets SET status=$2,approver_id=$3,approved_at=$4,updated_at=now() WHERE id=$1`,t.ID,status,member.EmployeeID,approvedAt)
	if err!=nil { response.Fail(w,500,"review timesheet failed",err.Error()); return }
	t.Status=status;t.ApproverID=&member.EmployeeID;t.ApprovedAt=approvedAt
	h.writeTimesheet(w,r,t)
}

func (h *Handler) Pending(w http.ResponseWriter,r *http.Request){
	member,_:=companyauth.MemberFromContext(r.Context())
	q:=`SELECT t.id,t.company_id,t.employee_id,t.started_at,t.ended_at,t.status,t.approver_id,t.approved_at FROM timesheets t WHERE t.company_id=$1 AND t.status='ready_to_submit'`
	args:=[]interface{}{member.CompanyID}
	if hasRole(member,"administrator","hr") {
		start,_:=week(time.Now().UTC()); q+=` AND t.started_at<$2 AND NOT EXISTS(SELECT 1 FROM direct_reports d WHERE d.employee_id=t.employee_id)`; args=append(args,start)
	} else { q+=` AND EXISTS(SELECT 1 FROM direct_reports d WHERE d.company_id=$1 AND d.manager_id=$2 AND d.employee_id=t.employee_id)`; args=append(args,member.EmployeeID) }
	q+=` ORDER BY t.started_at`
	rows,err:=h.pool.Query(r.Context(),q,args...)
	if err!=nil { response.Fail(w,500,"list pending timesheets failed",err.Error()); return }
	defer rows.Close(); out:=[]map[string]interface{}{}
	for rows.Next(){ var t timesheet; if rows.Scan(&t.ID,&t.CompanyID,&t.EmployeeID,&t.StartedAt,&t.EndedAt,&t.Status,&t.ApproverID,&t.ApprovedAt)!=nil { continue }; p,e:=h.payload(r.Context(),t);if e==nil{out=append(out,p)} }
	response.OK(w,"",out)
}

type wfhRequest struct{ Date string `json:"date"`; WorkFromHome bool `json:"work_from_home"` }
func (h *Handler) SetWorkFromHome(w http.ResponseWriter,r *http.Request){
	member,_:=companyauth.MemberFromContext(r.Context()); employeeID:=chi.URLParam(r,"employeeId")
	if employeeID!=member.EmployeeID&&!hasRole(member,"administrator","hr"){response.Fail(w,403,"Forbidden",nil);return}
	var enabled bool
	if err:=h.pool.QueryRow(r.Context(),`SELECT work_from_home_enabled FROM companies WHERE id=$1`,member.CompanyID).Scan(&enabled);err!=nil||!enabled{response.Fail(w,409,"Work from home is disabled for this company",nil);return}
	var req wfhRequest
	if json.NewDecoder(r.Body).Decode(&req)!=nil {response.Fail(w,400,"Invalid JSON body",nil);return}
	date,err:=time.Parse("2006-01-02",req.Date);if err!=nil{response.Fail(w,422,"invalid date",nil);return}
	if req.WorkFromHome {_,err=h.pool.Exec(r.Context(),`INSERT INTO employee_work_from_home(id,company_id,employee_id,date) VALUES($1,$2,$3,$4) ON CONFLICT(employee_id,date) DO NOTHING`,uuidv7.New(),member.CompanyID,employeeID,date)
	} else {_,err=h.pool.Exec(r.Context(),`DELETE FROM employee_work_from_home WHERE employee_id=$1 AND date=$2`,employeeID,date)}
	if err!=nil{response.Fail(w,500,"update work from home failed",err.Error());return}
	response.OK(w,"",map[string]interface{}{"date":req.Date,"work_from_home":req.WorkFromHome})
}
func (h *Handler) WorkFromHomeSetting(w http.ResponseWriter,r *http.Request){ member,_:=companyauth.MemberFromContext(r.Context());var enabled bool;err:=h.pool.QueryRow(r.Context(),`SELECT work_from_home_enabled FROM companies WHERE id=$1`,member.CompanyID).Scan(&enabled);if err!=nil{response.Fail(w,500,"setting lookup failed",err.Error());return};response.OK(w,"",map[string]bool{"enabled":enabled})}
func (h *Handler) UpdateWorkFromHomeSetting(w http.ResponseWriter,r *http.Request){member,_:=companyauth.MemberFromContext(r.Context());if !hasRole(member,"administrator","hr")&&!member.HasPermission("company.update"){response.Fail(w,403,"Forbidden",nil);return};var req struct{Enabled *bool `json:"enabled"`};if json.NewDecoder(r.Body).Decode(&req)!=nil||req.Enabled==nil{response.Fail(w,422,"enabled is required",nil);return};_,err:=h.pool.Exec(r.Context(),`UPDATE companies SET work_from_home_enabled=$2,updated_at=now() WHERE id=$1`,member.CompanyID,*req.Enabled);if err!=nil{response.Fail(w,500,"update setting failed",err.Error());return};response.OK(w,"",map[string]bool{"enabled":*req.Enabled})}

func (h *Handler) writeTimesheet(w http.ResponseWriter,r *http.Request,t timesheet){p,err:=h.payload(r.Context(),t);if err!=nil{response.Fail(w,500,"timesheet payload failed",err.Error());return};response.OK(w,"",p)}
func (h *Handler) payload(ctx context.Context,t timesheet)(map[string]interface{},error){
	var first,last,email string
	if err:=h.pool.QueryRow(ctx,`SELECT first_name,last_name,email FROM employees WHERE id=$1`,t.EmployeeID).Scan(&first,&last,&email);err!=nil{return nil,err}
	var approverName interface{}
	if t.ApproverID!=nil{var af,al string;if h.pool.QueryRow(ctx,`SELECT first_name,last_name FROM employees WHERE id=$1`,*t.ApproverID).Scan(&af,&al)==nil{approverName=strings.TrimSpace(af+" "+al)}}
	rows,err:=h.pool.Query(ctx,`SELECT id,timesheet_id,employee_id,duration,happened_at,description FROM time_tracking_entries WHERE timesheet_id=$1 ORDER BY happened_at`,t.ID);if err!=nil{return nil,err};defer rows.Close()
	entries:=[]map[string]interface{}{};total:=0
	for rows.Next(){var id,tsid,eid string;var duration int;var day time.Time;var description *string;if err:=rows.Scan(&id,&tsid,&eid,&duration,&day,&description);err!=nil{return nil,err};total+=duration;entries=append(entries,map[string]interface{}{"id":id,"timesheet_id":tsid,"employee_id":eid,"duration":duration,"happened_at":day.Format("2006-01-02"),"description":description})}
	var approved interface{};if t.ApprovedAt!=nil{approved=t.ApprovedAt.UTC().Format(time.RFC3339)}
	return map[string]interface{}{"id":t.ID,"company_id":t.CompanyID,"employee_id":t.EmployeeID,"employee":map[string]interface{}{"id":t.EmployeeID,"first_name":first,"last_name":last,"email":email},"started_at":t.StartedAt.Format("2006-01-02"),"ended_at":t.EndedAt.Format("2006-01-02"),"status":t.Status,"approved_at":approved,"approver_id":t.ApproverID,"approver_name":approverName,"entries":entries,"total_duration":total},nil
}
func hasRole(member companyauth.Member,names ...string)bool{for _,r:=range member.Roles{for _,n:=range names{if r==n{return true}}};return false}
