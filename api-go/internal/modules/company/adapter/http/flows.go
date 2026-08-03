package http

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	"github.com/XoDeR/empops/api-go/pkg/companyauth"
	"github.com/XoDeR/empops/api-go/pkg/response"
	"github.com/XoDeR/empops/api-go/pkg/uuidv7"
)

func (h *Handler) ListFlows(w http.ResponseWriter,r *http.Request){
	m,_:=companyauth.MemberFromContext(r.Context());rows,err:=h.pool.Query(r.Context(),`SELECT id,name,type,created_at,updated_at FROM flows WHERE company_id=$1 ORDER BY name`,m.CompanyID)
	if err!=nil{response.Fail(w,500,"list flows failed",err.Error());return};defer rows.Close();out:=[]map[string]interface{}{}
	for rows.Next(){var id,name,typ string;var created,updated time.Time;if rows.Scan(&id,&name,&typ,&created,&updated)==nil{out=append(out,map[string]interface{}{"id":id,"company_id":m.CompanyID,"name":name,"type":typ,"created_at":created.UTC().Format(time.RFC3339),"updated_at":updated.UTC().Format(time.RFC3339)})}}
	response.OK(w,"",out)
}

func (h *Handler) CreateFlow(w http.ResponseWriter,r *http.Request){
	m,_:=companyauth.MemberFromContext(r.Context());var req struct{Name string `json:"name"`;Type string `json:"type"`}
	if json.NewDecoder(r.Body).Decode(&req)!=nil||strings.TrimSpace(req.Name)==""||strings.TrimSpace(req.Type)==""{response.Fail(w,422,"name and type are required",nil);return}
	id:=uuidv7.New();_,err:=h.pool.Exec(r.Context(),`INSERT INTO flows(id,company_id,name,type) VALUES($1,$2,$3,$4)`,id,m.CompanyID,strings.TrimSpace(req.Name),strings.TrimSpace(req.Type))
	if err!=nil{response.Fail(w,500,"create flow failed",err.Error());return};response.Created(w,"Flow created",map[string]interface{}{"id":id,"company_id":m.CompanyID,"name":req.Name,"type":req.Type})
}

func (h *Handler) ShowFlow(w http.ResponseWriter,r *http.Request){
	m,_:=companyauth.MemberFromContext(r.Context());id:=chi.URLParam(r,"flowId");var name,typ string
	err:=h.pool.QueryRow(r.Context(),`SELECT name,type FROM flows WHERE id=$1 AND company_id=$2`,id,m.CompanyID).Scan(&name,&typ)
	if err==pgx.ErrNoRows{response.Fail(w,404,"Flow not found",nil);return};if err!=nil{response.Fail(w,500,"flow lookup failed",err.Error());return}
	rows,err:=h.pool.Query(r.Context(),`SELECT s.id,s.number,s.unit_of_time,s.modifier,s.real_number_of_days,
		a.id,a.type,a.recipient,a.specific_recipient_information
		FROM flow_steps s LEFT JOIN flow_actions a ON a.step_id=s.id WHERE s.flow_id=$1 ORDER BY s.real_number_of_days,s.id,a.id`,id)
	if err!=nil{response.Fail(w,500,"list steps failed",err.Error());return};defer rows.Close()
	steps:=[]map[string]interface{}{};index:=map[string]int{}
	for rows.Next(){var sid,unit,modifier string;var number,days int;var aid,atyp,recipient,specific *string
		if rows.Scan(&sid,&number,&unit,&modifier,&days,&aid,&atyp,&recipient,&specific)!=nil{continue}
		i,ok:=index[sid];if !ok{i=len(steps);index[sid]=i;steps=append(steps,map[string]interface{}{"id":sid,"number":number,"unit_of_time":unit,"modifier":modifier,"real_number_of_days":days,"actions":[]map[string]interface{}{}})}
		if aid!=nil{actions:=steps[i]["actions"].([]map[string]interface{});steps[i]["actions"]=append(actions,map[string]interface{}{"id":*aid,"type":atyp,"recipient":recipient,"specific_recipient_information":specific})}
	}
	response.OK(w,"",map[string]interface{}{"id":id,"company_id":m.CompanyID,"name":name,"type":typ,"steps":steps})
}

func (h *Handler) UpdateFlow(w http.ResponseWriter,r *http.Request){
	m,_:=companyauth.MemberFromContext(r.Context());var req map[string]interface{};if json.NewDecoder(r.Body).Decode(&req)!=nil{response.Fail(w,400,"Invalid JSON body",nil);return}
	id:=chi.URLParam(r,"flowId");if name,ok:=req["name"].(string);ok&&strings.TrimSpace(name)!=""{_,_=h.pool.Exec(r.Context(),`UPDATE flows SET name=$1,updated_at=now() WHERE id=$2 AND company_id=$3`,strings.TrimSpace(name),id,m.CompanyID)}
	if typ,ok:=req["type"].(string);ok&&strings.TrimSpace(typ)!=""{_,_=h.pool.Exec(r.Context(),`UPDATE flows SET type=$1,updated_at=now() WHERE id=$2 AND company_id=$3`,strings.TrimSpace(typ),id,m.CompanyID)}
	h.ShowFlow(w,r)
}

func (h *Handler) DeleteFlow(w http.ResponseWriter,r *http.Request){m,_:=companyauth.MemberFromContext(r.Context());tag,err:=h.pool.Exec(r.Context(),`DELETE FROM flows WHERE id=$1 AND company_id=$2`,chi.URLParam(r,"flowId"),m.CompanyID);writeDeleted(w,tag.RowsAffected(),err,"Flow")}

func (h *Handler) CreateFlowStep(w http.ResponseWriter,r *http.Request){
	m,_:=companyauth.MemberFromContext(r.Context());var req struct{Number int `json:"number"`;Unit string `json:"unit_of_time"`;Modifier string `json:"modifier"`;RealDays *int `json:"real_number_of_days"`}
	if json.NewDecoder(r.Body).Decode(&req)!=nil{response.Fail(w,400,"Invalid JSON body",nil);return};if req.Unit==""{req.Unit="days"};if req.Modifier!="before"&&req.Modifier!="after"{response.Fail(w,422,"modifier must be before or after",nil);return}
	days:=req.Number;if req.Unit=="weeks"{days*=7};if req.RealDays!=nil{days=*req.RealDays};if req.Modifier=="before"&&days<0{days=-days}
	id:=uuidv7.New();tag,err:=h.pool.Exec(r.Context(),`INSERT INTO flow_steps(id,flow_id,number,unit_of_time,modifier,real_number_of_days)
		SELECT $1,f.id,$2,$3,$4,$5 FROM flows f WHERE f.id=$6 AND f.company_id=$7`,id,req.Number,req.Unit,req.Modifier,days,chi.URLParam(r,"flowId"),m.CompanyID)
	if err!=nil{response.Fail(w,500,"create step failed",err.Error());return};if tag.RowsAffected()==0{response.Fail(w,404,"Flow not found",nil);return}
	response.Created(w,"Flow step created",map[string]interface{}{"id":id,"number":req.Number,"unit_of_time":req.Unit,"modifier":req.Modifier,"real_number_of_days":days})
}

func (h *Handler) UpdateFlowStep(w http.ResponseWriter,r *http.Request){
	m,_:=companyauth.MemberFromContext(r.Context());var req struct{Number int `json:"number"`;Unit string `json:"unit_of_time"`;Modifier string `json:"modifier"`;RealDays int `json:"real_number_of_days"`}
	if json.NewDecoder(r.Body).Decode(&req)!=nil||req.Unit==""||(req.Modifier!="before"&&req.Modifier!="after"){response.Fail(w,422,"invalid step",nil);return}
	tag,err:=h.pool.Exec(r.Context(),`UPDATE flow_steps s SET number=$1,unit_of_time=$2,modifier=$3,real_number_of_days=$4,updated_at=now()
		FROM flows f WHERE s.id=$5 AND s.flow_id=f.id AND f.company_id=$6`,req.Number,req.Unit,req.Modifier,req.RealDays,chi.URLParam(r,"stepId"),m.CompanyID)
	if err!=nil{response.Fail(w,500,"update step failed",err.Error());return};if tag.RowsAffected()==0{response.Fail(w,404,"Flow step not found",nil);return};response.OK(w,"Flow step updated",nil)
}
func (h *Handler) DeleteFlowStep(w http.ResponseWriter,r *http.Request){m,_:=companyauth.MemberFromContext(r.Context());tag,err:=h.pool.Exec(r.Context(),`DELETE FROM flow_steps s USING flows f WHERE s.id=$1 AND s.flow_id=f.id AND f.company_id=$2`,chi.URLParam(r,"stepId"),m.CompanyID);writeDeleted(w,tag.RowsAffected(),err,"Flow step")}

func (h *Handler) CreateFlowAction(w http.ResponseWriter,r *http.Request){
	m,_:=companyauth.MemberFromContext(r.Context());var req struct{Type string `json:"type"`;Recipient string `json:"recipient"`;Specific *string `json:"specific_recipient_information"`}
	if json.NewDecoder(r.Body).Decode(&req)!=nil||strings.TrimSpace(req.Type)==""||strings.TrimSpace(req.Recipient)==""{response.Fail(w,422,"type and recipient are required",nil);return}
	id:=uuidv7.New();tag,err:=h.pool.Exec(r.Context(),`INSERT INTO flow_actions(id,step_id,type,recipient,specific_recipient_information)
		SELECT $1,s.id,$2,$3,$4 FROM flow_steps s JOIN flows f ON f.id=s.flow_id WHERE s.id=$5 AND f.company_id=$6`,id,req.Type,req.Recipient,req.Specific,chi.URLParam(r,"stepId"),m.CompanyID)
	if err!=nil{response.Fail(w,500,"create action failed",err.Error());return};if tag.RowsAffected()==0{response.Fail(w,404,"Flow step not found",nil);return};response.Created(w,"Flow action created",map[string]interface{}{"id":id,"type":req.Type,"recipient":req.Recipient,"specific_recipient_information":req.Specific})
}
func (h *Handler) UpdateFlowAction(w http.ResponseWriter,r *http.Request){m,_:=companyauth.MemberFromContext(r.Context());var req struct{Type string `json:"type"`;Recipient string `json:"recipient"`;Specific *string `json:"specific_recipient_information"`};if json.NewDecoder(r.Body).Decode(&req)!=nil||req.Type==""||req.Recipient==""{response.Fail(w,422,"invalid action",nil);return};tag,err:=h.pool.Exec(r.Context(),`UPDATE flow_actions a SET type=$1,recipient=$2,specific_recipient_information=$3,updated_at=now() FROM flow_steps s,flows f WHERE a.id=$4 AND a.step_id=s.id AND s.flow_id=f.id AND f.company_id=$5`,req.Type,req.Recipient,req.Specific,chi.URLParam(r,"actionId"),m.CompanyID);if err!=nil{response.Fail(w,500,"update action failed",err.Error());return};if tag.RowsAffected()==0{response.Fail(w,404,"Flow action not found",nil);return};response.OK(w,"Flow action updated",nil)}
func (h *Handler) DeleteFlowAction(w http.ResponseWriter,r *http.Request){m,_:=companyauth.MemberFromContext(r.Context());tag,err:=h.pool.Exec(r.Context(),`DELETE FROM flow_actions a USING flow_steps s,flows f WHERE a.id=$1 AND a.step_id=s.id AND s.flow_id=f.id AND f.company_id=$2`,chi.URLParam(r,"actionId"),m.CompanyID);writeDeleted(w,tag.RowsAffected(),err,"Flow action")}

func writeDeleted(w http.ResponseWriter,rows int64,err error,name string){if err!=nil{response.Fail(w,500,"delete failed",err.Error());return};if rows==0{response.Fail(w,404,name+" not found",nil);return};response.OK(w,name+" deleted",nil)}
