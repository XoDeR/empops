// Command process-flows claims due employee flow actions.
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/XoDeR/empops/api-go/internal/infrastructure/config"
	"github.com/XoDeR/empops/api-go/internal/infrastructure/database"
)

func main(){if err:=run();err!=nil{fmt.Fprintln(os.Stderr,"process-flows: fatal:",err);os.Exit(1)}}
func run()error{
	path:=envOrDefault("EMPOPS_CONFIG","config/app.dev.yaml");cfg,err:=config.Load(path);if err!=nil{return fmt.Errorf("load config: %w",err)}
	if cfg.DB.DSN==""{return fmt.Errorf("database DSN is required")}
	date:=time.Now().UTC().Format("2006-01-02");if len(os.Args)>1{date=os.Args[1]};if _,err=time.Parse("2006-01-02",date);err!=nil{return fmt.Errorf("invalid date %q",date)}
	ctx,cancel:=context.WithTimeout(context.Background(),time.Minute);defer cancel();pool,err:=database.Connect(ctx,database.Config{DSN:cfg.DB.DSN,ConnectTimeout:10*time.Second});if err!=nil{return err};defer pool.Close()
	tag,err:=pool.Exec(ctx,`UPDATE flow_action_runs SET executed_at=now(),updated_at=now() WHERE due_on<=$1 AND executed_at IS NULL`,date)
	if err!=nil{return fmt.Errorf("process due actions: %w",err)}
	fmt.Printf("process-flows: processed %d action(s) due through %s\n",tag.RowsAffected(),date);return nil
}
func envOrDefault(key,fallback string)string{if v:=os.Getenv(key);v!=""{return v};return fallback}
