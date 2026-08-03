package http

import (
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/XoDeR/empops/api-go/pkg/companyauth"
	"github.com/XoDeR/empops/api-go/pkg/response"
)

type Handler struct{pool *pgxpool.Pool}
func NewHandler(pool *pgxpool.Pool)*Handler{return &Handler{pool:pool}}

func(h *Handler)ListInvoices(w http.ResponseWriter,r *http.Request){
	m,_:=companyauth.MemberFromContext(r.Context())
	rows,err:=h.pool.Query(r.Context(),`SELECT i.id,i.usage_history_id,i.sent_to_customer,i.customer_has_paid,
		i.email_address_invoice_sent_to,i.created_at,i.updated_at,u.logged_on,u.number_of_active_employees
		FROM company_invoices i LEFT JOIN company_daily_usage_history u ON u.id=i.usage_history_id
		WHERE i.company_id=$1 ORDER BY i.created_at DESC`,m.CompanyID)
	if err!=nil{response.Fail(w,500,"list invoices failed",err.Error());return};defer rows.Close()
	out:=[]map[string]interface{}{}
	for rows.Next(){var id string;var usageID,email *string;var sent,paid bool;var created,updated time.Time;var loggedOn *time.Time;var employees *int
		if err:=rows.Scan(&id,&usageID,&sent,&paid,&email,&created,&updated,&loggedOn,&employees);err!=nil{response.Fail(w,500,"scan failed",err.Error());return}
		var logged interface{};if loggedOn!=nil{logged=loggedOn.Format("2006-01-02")}
		out=append(out,map[string]interface{}{"id":id,"company_id":m.CompanyID,"usage_history_id":usageID,
			"sent_to_customer":sent,"customer_has_paid":paid,"email_address_invoice_sent_to":email,
			"logged_on":logged,"number_of_active_employees":employees,
			"created_at":created.UTC().Format(time.RFC3339),"updated_at":updated.UTC().Format(time.RFC3339)})
	}
	response.OK(w,"",out)
}
