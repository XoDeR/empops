// Command calculate-timeoff accrues PTO balances for one calendar day.
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/XoDeR/empops/api-go/internal/infrastructure/config"
	"github.com/XoDeR/empops/api-go/internal/infrastructure/database"
)

func main(){if err:=run();err!=nil{fmt.Fprintln(os.Stderr,"calculate-timeoff: fatal:",err);os.Exit(1)}}
func run()error{
	path:=envOrDefault("EMPOPS_CONFIG","config/app.dev.yaml");cfg,err:=config.Load(path);if err!=nil{return fmt.Errorf("load config: %w",err)}
	if cfg.DB.DSN==""{return fmt.Errorf("database DSN is required")}
	date:=time.Now().UTC().Format("2006-01-02");if len(os.Args)>1{date=os.Args[1]};if _,err=time.Parse("2006-01-02",date);err!=nil{return fmt.Errorf("invalid date %q",date)}
	ctx,cancel:=context.WithTimeout(context.Background(),time.Minute);defer cancel();pool,err:=database.Connect(ctx,database.Config{DSN:cfg.DB.DSN,ConnectTimeout:10*time.Second});if err!=nil{return err};defer pool.Close()
	tag,err:=pool.Exec(ctx,`WITH eligible AS (
		SELECT e.id AS employee_id,e.company_id,e.holiday_balance,
			COALESCE(e.amount_of_allowed_holidays,p.default_amount_of_allowed_holidays,0) AS yearly,
			NULLIF(p.total_worked_days,0) AS worked_days,
			COALESCE((SELECT CASE WHEN h.full THEN 1.0 ELSE 0.5 END FROM employee_planned_holidays h
				WHERE h.employee_id=e.id AND h.planned_date=$1 AND h.actually_taken=true
				AND h.type IN ('holiday','pto') ORDER BY h.created_at LIMIT 1),0) AS taken
		FROM employees e JOIN company_pto_policies p ON p.company_id=e.company_id AND p.year=EXTRACT(YEAR FROM $1::date)
		JOIN company_calendars c ON c.company_pto_policy_id=p.id AND c.day=$1
		WHERE e.locked=false AND c.is_worked=true
	), accrued AS (
		SELECT *,yearly/worked_days-taken AS delta FROM eligible WHERE worked_days IS NOT NULL
	), logged AS (
		INSERT INTO employee_daily_calendar_entries(id,employee_id,log_date,new_balance,daily_accrued_amount,
			current_holidays_per_year,default_amount_of_allowed_holidays_in_company)
		SELECT gen_random_uuid(),a.employee_id,$1,a.holiday_balance+a.delta,a.delta,a.yearly,p.default_amount_of_allowed_holidays
		FROM accrued a JOIN company_pto_policies p ON p.company_id=a.company_id AND p.year=EXTRACT(YEAR FROM $1::date)
		ON CONFLICT(employee_id,log_date) DO NOTHING RETURNING employee_id,new_balance
	)
	UPDATE employees e SET holiday_balance=l.new_balance,updated_at=now() FROM logged l WHERE e.id=l.employee_id`,date)
	if err!=nil{return fmt.Errorf("calculate accrual: %w",err)}
	fmt.Printf("calculate-timeoff: accrued %d employee(s) for %s\n",tag.RowsAffected(),date);return nil
}
func envOrDefault(key,fallback string)string{if v:=os.Getenv(key);v!=""{return v};return fallback}
