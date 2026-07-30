package http

import (
	"context"
	"encoding/json"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/XoDeR/empops/api-go/pkg/companyauth"
	"github.com/XoDeR/empops/api-go/pkg/notify"
	"github.com/XoDeR/empops/api-go/pkg/response"
	"github.com/XoDeR/empops/api-go/pkg/uuidv7"
)

type RateProvider interface{ Rate(context.Context,string,string,string)(float64,error) }
type Handler struct{pool *pgxpool.Pool;fx RateProvider}
func NewHandler(pool *pgxpool.Pool,fx RateProvider)*Handler{return &Handler{pool:pool,fx:fx}}

type expense struct{
	ID,CompanyID,Status,Title,Currency,ExpensedAt string
	EmployeeID,CategoryID,ConvertedCurrency,Description *string
	Amount int64
	ConvertedAmount *int64
	ConvertedAt *time.Time
	ExchangeRate *float64
	ManagerApproverID,ManagerReason,AccountingApproverID,AccountingReason *string
	ManagerApprovedAt,AccountingApprovedAt *time.Time
}
const expenseSelect=`SELECT id,company_id,employee_id,expense_category_id,status,title,amount,currency,
	converted_amount,converted_to_currency,converted_at,exchange_rate::float8,description,expensed_at::text,
	manager_approver_id,manager_approver_approved_at,manager_rejection_explanation,
	accounting_approver_id,accounting_approver_approved_at,accounting_rejection_explanation FROM expenses`
func scanExpense(row pgx.Row)(expense,error){var e expense;err:=row.Scan(&e.ID,&e.CompanyID,&e.EmployeeID,&e.CategoryID,&e.Status,&e.Title,&e.Amount,&e.Currency,&e.ConvertedAmount,&e.ConvertedCurrency,&e.ConvertedAt,&e.ExchangeRate,&e.Description,&e.ExpensedAt,&e.ManagerApproverID,&e.ManagerApprovedAt,&e.ManagerReason,&e.AccountingApproverID,&e.AccountingApprovedAt,&e.AccountingReason);return e,err}
func (h *Handler)find(ctx context.Context,companyID,id string)(expense,error){return scanExpense(h.pool.QueryRow(ctx,expenseSelect+` WHERE company_id=$1 AND id=$2`,companyID,id))}

func(h *Handler)Categories(w http.ResponseWriter,r *http.Request){member,_:=companyauth.MemberFromContext(r.Context());rows,err:=h.pool.Query(r.Context(),`SELECT id,company_id,name FROM expense_categories WHERE company_id=$1 ORDER BY name`,member.CompanyID);if err!=nil{response.Fail(w,500,"list categories failed",err.Error());return};defer rows.Close();out:=[]map[string]interface{}{};for rows.Next(){var id,cid,name string;if rows.Scan(&id,&cid,&name)==nil{out=append(out,categoryPayload(id,cid,name))}};response.OK(w,"",out)}
type categoryRequest struct{Name string `json:"name"`}
func(h *Handler)CreateCategory(w http.ResponseWriter,r *http.Request){member,_:=companyauth.MemberFromContext(r.Context());var req categoryRequest;if json.NewDecoder(r.Body).Decode(&req)!=nil||strings.TrimSpace(req.Name)==""{response.Fail(w,422,"name is required",nil);return};req.Name=strings.TrimSpace(req.Name);id:=uuidv7.New();_,err:=h.pool.Exec(r.Context(),`INSERT INTO expense_categories(id,company_id,name)VALUES($1,$2,$3)`,id,member.CompanyID,req.Name);if err!=nil{response.Fail(w,409,"Expense category already exists",nil);return};response.Created(w,"Expense category created",categoryPayload(id,member.CompanyID,req.Name))}
func(h *Handler)UpdateCategory(w http.ResponseWriter,r *http.Request){member,_:=companyauth.MemberFromContext(r.Context());var req categoryRequest;if json.NewDecoder(r.Body).Decode(&req)!=nil||strings.TrimSpace(req.Name)==""{response.Fail(w,422,"name is required",nil);return};tag,err:=h.pool.Exec(r.Context(),`UPDATE expense_categories SET name=$3,updated_at=now() WHERE id=$1 AND company_id=$2`,chi.URLParam(r,"categoryId"),member.CompanyID,strings.TrimSpace(req.Name));if err!=nil{response.Fail(w,409,"Expense category already exists",nil);return};if tag.RowsAffected()==0{response.Fail(w,404,"Expense category not found",nil);return};response.OK(w,"Expense category updated",categoryPayload(chi.URLParam(r,"categoryId"),member.CompanyID,strings.TrimSpace(req.Name)))}
func(h *Handler)DeleteCategory(w http.ResponseWriter,r *http.Request){member,_:=companyauth.MemberFromContext(r.Context());tag,err:=h.pool.Exec(r.Context(),`DELETE FROM expense_categories WHERE id=$1 AND company_id=$2`,chi.URLParam(r,"categoryId"),member.CompanyID);if err!=nil{response.Fail(w,500,"delete category failed",err.Error());return};if tag.RowsAffected()==0{response.Fail(w,404,"Expense category not found",nil);return};response.OK(w,"Expense category deleted",nil)}
func categoryPayload(id,cid,name string)map[string]interface{}{return map[string]interface{}{"id":id,"company_id":cid,"name":name}}

type createExpenseRequest struct{Title string `json:"title"`;Amount int64 `json:"amount"`;Currency string `json:"currency"`;ExpensedAt string `json:"expensed_at"`;CategoryID *string `json:"expense_category_id"`;Description *string `json:"description"`}
func(h *Handler)CreateExpense(w http.ResponseWriter,r *http.Request){
	member,_:=companyauth.MemberFromContext(r.Context());var req createExpenseRequest
	if json.NewDecoder(r.Body).Decode(&req)!=nil{response.Fail(w,400,"Invalid JSON body",nil);return}
	req.Title=strings.TrimSpace(req.Title);req.Currency=strings.ToUpper(strings.TrimSpace(req.Currency))
	if req.Title==""||req.Amount<1||len(req.Currency)!=3{response.Fail(w,422,"invalid expense",nil);return}
	if _,err:=time.Parse("2006-01-02",req.ExpensedAt);err!=nil{response.Fail(w,422,"invalid expensed_at",nil);return}
	if req.CategoryID!=nil{var ok bool;_ = h.pool.QueryRow(r.Context(),`SELECT EXISTS(SELECT 1 FROM expense_categories WHERE id=$1 AND company_id=$2)`,*req.CategoryID,member.CompanyID).Scan(&ok);if !ok{response.Fail(w,404,"Expense category not found",nil);return}}
	var companyCurrency string;if err:=h.pool.QueryRow(r.Context(),`SELECT currency FROM companies WHERE id=$1`,member.CompanyID).Scan(&companyCurrency);err!=nil{response.Fail(w,500,"company lookup failed",err.Error());return}
	var converted *int64;var convertedCurrency *string;var convertedAt *time.Time;var rate *float64
	if req.Currency!=strings.ToUpper(companyCurrency){v,err:=h.fx.Rate(r.Context(),req.ExpensedAt,req.Currency,companyCurrency);if err!=nil{response.Fail(w,502,"Unable to retrieve exchange rate",nil);return};amount:=int64(math.Round(float64(req.Amount)*v));now:=time.Now().UTC();target:=strings.ToUpper(companyCurrency);converted=&amount;convertedCurrency=&target;convertedAt=&now;rate=&v}
	rows,err:=h.pool.Query(r.Context(),`SELECT manager_id FROM direct_reports WHERE company_id=$1 AND employee_id=$2`,member.CompanyID,member.EmployeeID);if err!=nil{response.Fail(w,500,"manager lookup failed",err.Error());return};managers:=[]string{};for rows.Next(){var id string;if rows.Scan(&id)==nil{managers=append(managers,id)}};rows.Close()
	status:="accounting_approval";if len(managers)>0{status="manager_approval"}
	id:=uuidv7.New();_,err=h.pool.Exec(r.Context(),`INSERT INTO expenses(id,company_id,employee_id,expense_category_id,status,title,amount,currency,converted_amount,converted_to_currency,converted_at,exchange_rate,description,expensed_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)`,id,member.CompanyID,member.EmployeeID,req.CategoryID,status,req.Title,req.Amount,req.Currency,converted,convertedCurrency,convertedAt,rate,req.Description,req.ExpensedAt)
	if err!=nil{response.Fail(w,500,"create expense failed",err.Error());return}
	for _,managerID:=range managers{_ = notify.Create(r.Context(),h.pool,member.CompanyID,managerID,"expense.manager_approval_requested",map[string]interface{}{"expense_id":id,"employee_id":member.EmployeeID})}
	e,err:=h.find(r.Context(),member.CompanyID,id);if err!=nil{response.Fail(w,500,"expense payload failed",err.Error());return};p,_:=h.payload(r.Context(),e);response.Created(w,"Expense created",p)
}

func(h *Handler)Expenses(w http.ResponseWriter,r *http.Request){member,_:=companyauth.MemberFromContext(r.Context());q:=expenseSelect+` WHERE company_id=$1`;args:=[]interface{}{member.CompanyID};employeeFilter:=r.URL.Query().Get("employeeId");if hasRole(member,"administrator","hr","accountant"){if employeeFilter!=""{q+=` AND employee_id=$2`;args=append(args,employeeFilter)}}else{q+=` AND (employee_id=$2 OR EXISTS(SELECT 1 FROM direct_reports d WHERE d.company_id=$1 AND d.manager_id=$2 AND d.employee_id=expenses.employee_id))`;args=append(args,member.EmployeeID);if employeeFilter!=""{q+=` AND employee_id=$3`;args=append(args,employeeFilter)}};q+=` ORDER BY created_at DESC`;h.list(w,r,q,args...)}
func(h *Handler)PendingManager(w http.ResponseWriter,r *http.Request){member,_:=companyauth.MemberFromContext(r.Context());h.list(w,r,expenseSelect+` WHERE company_id=$1 AND status='manager_approval' AND EXISTS(SELECT 1 FROM direct_reports d WHERE d.company_id=$1 AND d.manager_id=$2 AND d.employee_id=expenses.employee_id) ORDER BY created_at`,member.CompanyID,member.EmployeeID)}
func(h *Handler)PendingAccounting(w http.ResponseWriter,r *http.Request){member,_:=companyauth.MemberFromContext(r.Context());h.list(w,r,expenseSelect+` WHERE company_id=$1 AND status='accounting_approval' ORDER BY created_at`,member.CompanyID)}
func(h *Handler)list(w http.ResponseWriter,r *http.Request,q string,args ...interface{}){rows,err:=h.pool.Query(r.Context(),q,args...);if err!=nil{response.Fail(w,500,"list expenses failed",err.Error());return};defer rows.Close();out:=[]map[string]interface{}{};for rows.Next(){e,err:=scanExpense(rows);if err!=nil{response.Fail(w,500,"scan expense failed",err.Error());return};p,err:=h.payload(r.Context(),e);if err!=nil{response.Fail(w,500,"expense payload failed",err.Error());return};out=append(out,p)};response.OK(w,"",out)}
func(h *Handler)ShowExpense(w http.ResponseWriter,r *http.Request){member,_:=companyauth.MemberFromContext(r.Context());e,err:=h.find(r.Context(),member.CompanyID,chi.URLParam(r,"expenseId"));if err!=nil{response.Fail(w,404,"Expense not found",nil);return};if !h.canView(r.Context(),member,e.EmployeeID){response.Fail(w,403,"Forbidden",nil);return};p,_:=h.payload(r.Context(),e);response.OK(w,"",p)}
func(h *Handler)DeleteExpense(w http.ResponseWriter,r *http.Request){member,_:=companyauth.MemberFromContext(r.Context());e,err:=h.find(r.Context(),member.CompanyID,chi.URLParam(r,"expenseId"));if err!=nil{response.Fail(w,404,"Expense not found",nil);return};if e.Status=="accepted"{response.Fail(w,409,"Accepted expenses cannot be deleted",nil);return};if e.EmployeeID==nil||*e.EmployeeID!=member.EmployeeID&&!member.HasPermission("expenses.delete"){response.Fail(w,403,"Forbidden",nil);return};_,err=h.pool.Exec(r.Context(),`DELETE FROM expenses WHERE id=$1`,e.ID);if err!=nil{response.Fail(w,500,"delete expense failed",err.Error());return};response.OK(w,"Expense deleted",nil)}
func(h *Handler)ManagerApprove(w http.ResponseWriter,r *http.Request){h.managerDecision(w,r,true)}
func(h *Handler)ManagerReject(w http.ResponseWriter,r *http.Request){h.managerDecision(w,r,false)}
func(h *Handler)managerDecision(w http.ResponseWriter,r *http.Request,approve bool){member,_:=companyauth.MemberFromContext(r.Context());e,err:=h.find(r.Context(),member.CompanyID,chi.URLParam(r,"expenseId"));if err!=nil{response.Fail(w,404,"Expense not found",nil);return};if e.Status!="manager_approval"{response.Fail(w,409,"Expense is not awaiting manager approval",nil);return};var manages bool;if e.EmployeeID!=nil{_ = h.pool.QueryRow(r.Context(),`SELECT EXISTS(SELECT 1 FROM direct_reports WHERE company_id=$1 AND manager_id=$2 AND employee_id=$3)`,member.CompanyID,member.EmployeeID,*e.EmployeeID).Scan(&manages)};if !manages{response.Fail(w,403,"Forbidden",nil);return};reason,ok:=decisionReason(w,r,approve);if !ok{return};status:="rejected_by_manager";var approvedAt *time.Time;if approve{status="accounting_approval";now:=time.Now().UTC();approvedAt=&now;reason=nil};_,err=h.pool.Exec(r.Context(),`UPDATE expenses SET status=$2,manager_approver_id=$3,manager_approver_approved_at=$4,manager_rejection_explanation=$5,updated_at=now() WHERE id=$1`,e.ID,status,member.EmployeeID,approvedAt,reason);if err!=nil{response.Fail(w,500,"review expense failed",err.Error());return};e.Status=status;e.ManagerApproverID=&member.EmployeeID;e.ManagerApprovedAt=approvedAt;e.ManagerReason=reason;p,_:=h.payload(r.Context(),e);response.OK(w,"",p)}
func(h *Handler)AccountingApprove(w http.ResponseWriter,r *http.Request){h.accountingDecision(w,r,true)}
func(h *Handler)AccountingReject(w http.ResponseWriter,r *http.Request){h.accountingDecision(w,r,false)}
func(h *Handler)accountingDecision(w http.ResponseWriter,r *http.Request,approve bool){member,_:=companyauth.MemberFromContext(r.Context());e,err:=h.find(r.Context(),member.CompanyID,chi.URLParam(r,"expenseId"));if err!=nil{response.Fail(w,404,"Expense not found",nil);return};if e.Status!="accounting_approval"{response.Fail(w,409,"Expense is not awaiting accounting approval",nil);return};reason,ok:=decisionReason(w,r,approve);if !ok{return};status:="rejected_by_accounting";var approvedAt *time.Time;if approve{status="accepted";now:=time.Now().UTC();approvedAt=&now;reason=nil};_,err=h.pool.Exec(r.Context(),`UPDATE expenses SET status=$2,accounting_approver_id=$3,accounting_approver_approved_at=$4,accounting_rejection_explanation=$5,updated_at=now() WHERE id=$1`,e.ID,status,member.EmployeeID,approvedAt,reason);if err!=nil{response.Fail(w,500,"finalize expense failed",err.Error());return};e.Status=status;e.AccountingApproverID=&member.EmployeeID;e.AccountingApprovedAt=approvedAt;e.AccountingReason=reason;p,_:=h.payload(r.Context(),e);response.OK(w,"",p)}
func decisionReason(w http.ResponseWriter,r *http.Request,approve bool)(*string,bool){if approve{return nil,true};var req struct{Reason string `json:"reason"`};if json.NewDecoder(r.Body).Decode(&req)!=nil||strings.TrimSpace(req.Reason)==""{response.Fail(w,422,"reason is required",nil);return nil,false};req.Reason=strings.TrimSpace(req.Reason);return &req.Reason,true}

func(h *Handler)GrantAccountant(w http.ResponseWriter,r *http.Request){h.accountant(w,r,true)}
func(h *Handler)RevokeAccountant(w http.ResponseWriter,r *http.Request){h.accountant(w,r,false)}
func(h *Handler)accountant(w http.ResponseWriter,r *http.Request,grant bool){member,_:=companyauth.MemberFromContext(r.Context());if !hasRole(member,"administrator","hr"){response.Fail(w,403,"Forbidden",nil);return};employeeID:=chi.URLParam(r,"employeeId");var exists bool;_ = h.pool.QueryRow(r.Context(),`SELECT EXISTS(SELECT 1 FROM employees WHERE id=$1 AND company_id=$2)`,employeeID,member.CompanyID).Scan(&exists);if !exists{response.Fail(w,404,"Employee not found",nil);return};var err error;if grant{_,err=h.pool.Exec(r.Context(),`INSERT INTO employee_roles(employee_id,role_id) SELECT $1,id FROM roles WHERE name='accountant' ON CONFLICT DO NOTHING`,employeeID)}else{_,err=h.pool.Exec(r.Context(),`DELETE FROM employee_roles WHERE employee_id=$1 AND role_id=(SELECT id FROM roles WHERE name='accountant')`,employeeID)};if err!=nil{response.Fail(w,500,"update accountant failed",err.Error());return};if grant{response.OK(w,"Accountant granted",nil)}else{response.OK(w,"Accountant revoked",nil)}}

func(h *Handler)canView(ctx context.Context,m companyauth.Member,employeeID *string)bool{if hasRole(m,"administrator","hr","accountant")||employeeID!=nil&&*employeeID==m.EmployeeID{return true};if employeeID==nil{return false};var ok bool;_ = h.pool.QueryRow(ctx,`SELECT EXISTS(SELECT 1 FROM direct_reports WHERE company_id=$1 AND manager_id=$2 AND employee_id=$3)`,m.CompanyID,m.EmployeeID,*employeeID).Scan(&ok);return ok}
func(h *Handler)payload(ctx context.Context,e expense)(map[string]interface{},error){
	var employeeName interface{};if e.EmployeeID!=nil{var f,l string;if err:=h.pool.QueryRow(ctx,`SELECT first_name,last_name FROM employees WHERE id=$1`,*e.EmployeeID).Scan(&f,&l);err==nil{employeeName=strings.TrimSpace(f+" "+l)}}
	var category interface{};if e.CategoryID!=nil{var id,cid,name string;if err:=h.pool.QueryRow(ctx,`SELECT id,company_id,name FROM expense_categories WHERE id=$1`,*e.CategoryID).Scan(&id,&cid,&name);err==nil{category=categoryPayload(id,cid,name)}}
	name:=func(id *string)interface{}{if id==nil{return nil};var f,l string;if h.pool.QueryRow(ctx,`SELECT first_name,last_name FROM employees WHERE id=$1`,*id).Scan(&f,&l)==nil{return strings.TrimSpace(f+" "+l)};return nil}
	format:=func(t *time.Time)interface{}{if t==nil{return nil};return t.UTC().Format(time.RFC3339)}
	return map[string]interface{}{"id":e.ID,"company_id":e.CompanyID,"employee_id":e.EmployeeID,"employee_name":employeeName,"expense_category_id":e.CategoryID,"category":category,"status":e.Status,"title":e.Title,"amount":e.Amount,"currency":e.Currency,"converted_amount":e.ConvertedAmount,"converted_to_currency":e.ConvertedCurrency,"converted_at":format(e.ConvertedAt),"exchange_rate":e.ExchangeRate,"description":e.Description,"expensed_at":e.ExpensedAt,"manager_approver_id":e.ManagerApproverID,"manager_approver_name":name(e.ManagerApproverID),"manager_approver_approved_at":format(e.ManagerApprovedAt),"manager_rejection_explanation":e.ManagerReason,"accounting_approver_id":e.AccountingApproverID,"accounting_approver_name":name(e.AccountingApproverID),"accounting_approver_approved_at":format(e.AccountingApprovedAt),"accounting_rejection_explanation":e.AccountingReason},nil
}
func hasRole(m companyauth.Member,names ...string)bool{for _,r:=range m.Roles{for _,n:=range names{if r==n{return true}}};return false}
