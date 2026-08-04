package http

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	"github.com/XoDeR/empops/api-go/pkg/companyauth"
	"github.com/XoDeR/empops/api-go/pkg/response"
	"github.com/XoDeR/empops/api-go/pkg/uuidv7"
)

type ptoPolicyRequest struct {
	Year                            int     `json:"year"`
	TotalWorkedDays                 int     `json:"total_worked_days"`
	DefaultAmountOfAllowedHolidays float64 `json:"default_amount_of_allowed_holidays"`
	DefaultAmountOfSickDays        float64 `json:"default_amount_of_sick_days"`
	DefaultAmountOfPTODays         float64 `json:"default_amount_of_pto_days"`
}

func (h *Handler) ListPTOPolicies(w http.ResponseWriter, r *http.Request) {
	member, _ := companyauth.MemberFromContext(r.Context())
	rows, err := h.pool.Query(r.Context(), `SELECT id,year,total_worked_days,
		default_amount_of_allowed_holidays,default_amount_of_sick_days,default_amount_of_pto_days,
		created_at,updated_at FROM company_pto_policies WHERE company_id=$1 ORDER BY year DESC`, member.CompanyID)
	if err != nil { response.Fail(w, 500, "list PTO policies failed", err.Error()); return }
	defer rows.Close()
	out := []map[string]interface{}{}
	for rows.Next() {
		item, err := scanPTOPolicy(rows)
		if err != nil { response.Fail(w, 500, "scan failed", err.Error()); return }
		out = append(out, item)
	}
	response.OK(w, "", out)
}

func (h *Handler) CreatePTOPolicy(w http.ResponseWriter, r *http.Request) {
	member, _ := companyauth.MemberFromContext(r.Context())
	var req ptoPolicyRequest
	if json.NewDecoder(r.Body).Decode(&req) != nil || req.Year < 2000 || req.Year > 9999 ||
		req.DefaultAmountOfAllowedHolidays < 0 || req.DefaultAmountOfSickDays < 0 || req.DefaultAmountOfPTODays < 0 {
		response.Fail(w, 422, "invalid PTO policy", nil); return
	}
	id := uuidv7.New()
	tx, err := h.pool.Begin(r.Context())
	if err != nil { response.Fail(w, 500, "transaction failed", err.Error()); return }
	defer tx.Rollback(r.Context())
	_, err = tx.Exec(r.Context(), `INSERT INTO company_pto_policies
		(id,company_id,year,total_worked_days,default_amount_of_allowed_holidays,default_amount_of_sick_days,default_amount_of_pto_days)
		VALUES($1,$2,$3,0,$4,$5,$6)`, id, member.CompanyID, req.Year, req.DefaultAmountOfAllowedHolidays, req.DefaultAmountOfSickDays, req.DefaultAmountOfPTODays)
	if err != nil { response.Fail(w, 409, "PTO policy already exists", err.Error()); return }
	worked := 0
	for day := time.Date(req.Year, 1, 1, 0, 0, 0, 0, time.UTC); day.Year() == req.Year; day = day.AddDate(0, 0, 1) {
		isWorked := day.Weekday() != time.Saturday && day.Weekday() != time.Sunday
		if isWorked { worked++ }
		_, err = tx.Exec(r.Context(), `INSERT INTO company_calendars
			(id,company_pto_policy_id,day,day_of_week,day_of_year,is_worked) VALUES($1,$2,$3,$4,$5,$6)`,
			uuidv7.New(), id, day, int(day.Weekday()), day.YearDay(), isWorked)
		if err != nil { response.Fail(w, 500, "create calendar failed", err.Error()); return }
	}
	_, _ = tx.Exec(r.Context(), `UPDATE company_pto_policies SET total_worked_days=$2 WHERE id=$1`, id, worked)
	if err = tx.Commit(r.Context()); err != nil { response.Fail(w, 500, "commit failed", err.Error()); return }
	item, _ := h.loadPTOPolicy(r, member.CompanyID, id)
	response.Created(w, "PTO policy created", item)
}

func (h *Handler) ShowPTOPolicy(w http.ResponseWriter, r *http.Request) {
	member, _ := companyauth.MemberFromContext(r.Context())
	item, err := h.loadPTOPolicy(r, member.CompanyID, chi.URLParam(r, "policyId"))
	if err == pgx.ErrNoRows { response.Fail(w, 404, "PTO policy not found", nil); return }
	if err != nil { response.Fail(w, 500, "lookup failed", err.Error()); return }
	response.OK(w, "", item)
}

func (h *Handler) UpdatePTOPolicy(w http.ResponseWriter, r *http.Request) {
	member, _ := companyauth.MemberFromContext(r.Context())
	id := chi.URLParam(r, "policyId")
	var req map[string]interface{}
	if json.NewDecoder(r.Body).Decode(&req) != nil { response.Fail(w, 400, "Invalid JSON body", nil); return }
	fields := map[string]string{
		"default_amount_of_allowed_holidays": "default_amount_of_allowed_holidays",
		"default_amount_of_sick_days": "default_amount_of_sick_days",
		"default_amount_of_pto_days": "default_amount_of_pto_days",
	}
	for key, col := range fields {
		if value, ok := req[key].(float64); ok && value >= 0 {
			_, _ = h.pool.Exec(r.Context(), `UPDATE company_pto_policies SET `+col+`=$1,updated_at=now() WHERE id=$2 AND company_id=$3`, value, id, member.CompanyID)
		}
	}
	item, err := h.loadPTOPolicy(r, member.CompanyID, id)
	if err == pgx.ErrNoRows { response.Fail(w, 404, "PTO policy not found", nil); return }
	if err != nil { response.Fail(w, 500, "lookup failed", err.Error()); return }
	response.OK(w, "PTO policy updated", item)
}

func (h *Handler) PTOPolicyCalendar(w http.ResponseWriter, r *http.Request) {
	member, _ := companyauth.MemberFromContext(r.Context())
	rows, err := h.pool.Query(r.Context(), `SELECT c.id,c.day,c.day_of_week,c.day_of_year,c.is_worked
		FROM company_calendars c JOIN company_pto_policies p ON p.id=c.company_pto_policy_id
		WHERE p.id=$1 AND p.company_id=$2 ORDER BY c.day`, chi.URLParam(r, "policyId"), member.CompanyID)
	if err != nil { response.Fail(w, 500, "calendar lookup failed", err.Error()); return }
	defer rows.Close()
	out := []map[string]interface{}{}
	for rows.Next() {
		var id string; var day time.Time; var weekday, yearDay int; var worked bool
		if rows.Scan(&id, &day, &weekday, &yearDay, &worked) != nil { continue }
		out = append(out, map[string]interface{}{"id":id,"day":day.Format("2006-01-02"),"day_of_week":weekday,"day_of_year":yearDay,"is_worked":worked})
	}
	response.OK(w, "", out)
}

func (h *Handler) UpdatePTOPolicyCalendarDay(w http.ResponseWriter, r *http.Request) {
	member, _ := companyauth.MemberFromContext(r.Context())
	day, err := time.Parse("2006-01-02", chi.URLParam(r, "day"))
	var req struct { IsWorked *bool `json:"is_worked"` }
	if err != nil || json.NewDecoder(r.Body).Decode(&req) != nil || req.IsWorked == nil {
		response.Fail(w, 422, "is_worked and a valid day are required", nil); return
	}
	tag, err := h.pool.Exec(r.Context(), `UPDATE company_calendars c SET is_worked=$1,updated_at=now()
		FROM company_pto_policies p WHERE c.company_pto_policy_id=p.id AND p.id=$2 AND p.company_id=$3 AND c.day=$4`,
		*req.IsWorked, chi.URLParam(r, "policyId"), member.CompanyID, day)
	if err != nil { response.Fail(w, 500, "calendar update failed", err.Error()); return }
	if tag.RowsAffected() == 0 { response.Fail(w, 404, "Calendar day not found", nil); return }
	_, _ = h.pool.Exec(r.Context(), `UPDATE company_pto_policies p SET total_worked_days=(
		SELECT count(*) FROM company_calendars c WHERE c.company_pto_policy_id=p.id AND c.is_worked),updated_at=now()
		WHERE p.id=$1 AND p.company_id=$2`, chi.URLParam(r, "policyId"), member.CompanyID)
	response.OK(w, "Calendar updated", map[string]interface{}{"day":day.Format("2006-01-02"),"is_worked":*req.IsWorked})
}

func (h *Handler) HolidayBalance(w http.ResponseWriter, r *http.Request) {
	member, _ := companyauth.MemberFromContext(r.Context())
	employeeID := chi.URLParam(r, "employeeId")
	if employeeID != member.EmployeeID && !member.HasPermission("pto.view") { response.Fail(w, 403, "Forbidden", nil); return }
	var balance float64; var allowed, sick, pto *float64
	err := h.pool.QueryRow(r.Context(), `SELECT holiday_balance,amount_of_allowed_holidays,amount_of_sick_days,amount_of_pto_days
		FROM employees WHERE id=$1 AND company_id=$2`, employeeID, member.CompanyID).Scan(&balance,&allowed,&sick,&pto)
	if err == pgx.ErrNoRows { response.Fail(w,404,"Employee not found",nil);return }
	if err != nil { response.Fail(w,500,"balance lookup failed",err.Error());return }
	response.OK(w,"",map[string]interface{}{"employee_id":employeeID,"holiday_balance":balance,
		"amount_of_allowed_holidays":allowed,"amount_of_sick_days":sick,"amount_of_pto_days":pto})
}

func (h *Handler) ListHolidays(w http.ResponseWriter, r *http.Request) {
	member, _ := companyauth.MemberFromContext(r.Context()); employeeID := chi.URLParam(r,"employeeId")
	if employeeID != member.EmployeeID && !member.HasPermission("pto.view") { response.Fail(w,403,"Forbidden",nil);return }
	q := `SELECT h.id,h.planned_date,h.type,h."full",h.actually_taken,h.created_at FROM employee_planned_holidays h
		JOIN employees e ON e.id=h.employee_id WHERE e.id=$1 AND e.company_id=$2`
	args := []interface{}{employeeID, member.CompanyID}
	if year := r.URL.Query().Get("year"); year != "" {
		if y, err := strconv.Atoi(year); err == nil { q += ` AND EXTRACT(YEAR FROM h.planned_date)=$3`; args=append(args,y) }
	}
	q += ` ORDER BY h.planned_date`
	rows,err:=h.pool.Query(r.Context(),q,args...);if err!=nil{response.Fail(w,500,"list holidays failed",err.Error());return};defer rows.Close()
	out:=[]map[string]interface{}{}
	for rows.Next(){var id,typ string;var day,created time.Time;var full,taken bool;if rows.Scan(&id,&day,&typ,&full,&taken,&created)==nil{out=append(out,map[string]interface{}{"id":id,"employee_id":employeeID,"planned_date":day.Format("2006-01-02"),"type":typ,"full":full,"actually_taken":taken,"created_at":created.UTC().Format(time.RFC3339)})}}
	response.OK(w,"",out)
}

func (h *Handler) CreateHoliday(w http.ResponseWriter, r *http.Request) {
	member,_:=companyauth.MemberFromContext(r.Context()); employeeID:=chi.URLParam(r,"employeeId")
	if employeeID!=member.EmployeeID&&!member.HasPermission("pto.manage"){response.Fail(w,403,"Forbidden",nil);return}
	var req struct{PlannedDate string `json:"planned_date"`;Type string `json:"type"`;Full *bool `json:"full"`;ActuallyTaken bool `json:"actually_taken"`}
	if json.NewDecoder(r.Body).Decode(&req)!=nil{response.Fail(w,400,"Invalid JSON body",nil);return}
	day,err:=time.Parse("2006-01-02",req.PlannedDate);req.Type=strings.TrimSpace(req.Type)
	if err!=nil||req.Type==""{response.Fail(w,422,"planned_date and type are required",nil);return}
	full:=true;if req.Full!=nil{full=*req.Full};id:=uuidv7.New()
	tag,err:=h.pool.Exec(r.Context(),`INSERT INTO employee_planned_holidays(id,employee_id,planned_date,type,"full",actually_taken)
		SELECT $1,e.id,$2,$3,$4,$5 FROM employees e WHERE e.id=$6 AND e.company_id=$7`,
		id,day,req.Type,full,req.ActuallyTaken,employeeID,member.CompanyID)
	if err!=nil{response.Fail(w,409,"holiday already exists",err.Error());return};if tag.RowsAffected()==0{response.Fail(w,404,"Employee not found",nil);return}
	response.Created(w,"Holiday created",map[string]interface{}{"id":id,"employee_id":employeeID,"planned_date":req.PlannedDate,"type":req.Type,"full":full,"actually_taken":req.ActuallyTaken})
}

func (h *Handler) DeleteHoliday(w http.ResponseWriter,r *http.Request){
	member,_:=companyauth.MemberFromContext(r.Context());employeeID:=chi.URLParam(r,"employeeId")
	if employeeID!=member.EmployeeID&&!member.HasPermission("pto.manage"){response.Fail(w,403,"Forbidden",nil);return}
	tag,err:=h.pool.Exec(r.Context(),`DELETE FROM employee_planned_holidays h USING employees e
		WHERE h.id=$1 AND h.employee_id=e.id AND e.id=$2 AND e.company_id=$3`,chi.URLParam(r,"holidayId"),employeeID,member.CompanyID)
	if err!=nil{response.Fail(w,500,"delete holiday failed",err.Error());return};if tag.RowsAffected()==0{response.Fail(w,404,"Holiday not found",nil);return}
	response.OK(w,"Holiday deleted",nil)
}

func scanPTOPolicy(row pgx.Row)(map[string]interface{},error){
	var id string;var year,worked int;var holidays,sick,pto float64;var created,updated time.Time
	if err:=row.Scan(&id,&year,&worked,&holidays,&sick,&pto,&created,&updated);err!=nil{return nil,err}
	return map[string]interface{}{"id":id,"year":year,"total_worked_days":worked,
		"default_amount_of_allowed_holidays":holidays,"default_amount_of_sick_days":sick,"default_amount_of_pto_days":pto,
		"created_at":created.UTC().Format(time.RFC3339),"updated_at":updated.UTC().Format(time.RFC3339)},nil
}

func(h *Handler)loadPTOPolicy(r *http.Request,companyID,id string)(map[string]interface{},error){
	return scanPTOPolicy(h.pool.QueryRow(r.Context(),`SELECT id,year,total_worked_days,
		default_amount_of_allowed_holidays,default_amount_of_sick_days,default_amount_of_pto_days,created_at,updated_at
		FROM company_pto_policies WHERE id=$1 AND company_id=$2`,id,companyID))
}
