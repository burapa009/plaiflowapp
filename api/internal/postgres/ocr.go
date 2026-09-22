package postgres

import (
 "context"
 "encoding/hex"
 "encoding/json"
 "errors"
 "time"

 "github.com/jackc/pgx/v5"
 "plaiflow/api/internal/job"
 "plaiflow/api/internal/ocr"
 "plaiflow/api/internal/tenant"
)

func (s *Store) OCRInput(ctx context.Context, c job.LeaseCommand, worker string) (ocr.Input,error) {
 var in ocr.Input
 err:=s.pool.QueryRow(ctx,`SELECT d.organization_id,d.id,encode(d.content_sha256,'hex'),d.detected_mime,d.byte_size,d.storage_key,j.payload->>'model_version',j.payload->>'preprocessing_version'
 FROM durable_jobs j JOIN document_ocr_runs o ON o.job_id=j.id JOIN documents d ON d.id=o.document_id AND d.organization_id=o.organization_id
 WHERE j.id=$1 AND j.kind='ocr' AND d.status NOT IN ('Trash','Purged') AND j.payload->>'sha256'=encode(d.content_sha256,'hex') AND
 ((j.status='Running' AND j.current_attempt_id=$2 AND j.lease_token_hash=$3 AND j.lease_expires_at>$4 AND j.worker_id=$5)
 OR (j.status='Completed' AND j.result->>'attempt_id'=$2 AND j.result->>'lease_hash'=$6 AND j.result->>'worker_id'=$5))`,c.JobID,c.AttemptID,leaseHash(c.LeaseToken),c.Now,worker,hex.EncodeToString(leaseHash(c.LeaseToken))).Scan(&in.OrganizationID,&in.DocumentID,&in.SHA256,&in.MIME,&in.Size,&in.StorageKey,&in.ModelVersion,&in.PreprocessingVersion)
 if errors.Is(err,pgx.ErrNoRows) {return in,job.ErrLeaseLost}; return in,err
}

func (s *Store) CompleteOCR(ctx context.Context,c job.LeaseCommand,worker string,result ocr.Result,key,sha string,size int64)(string,error){
 in,err:=s.OCRInput(ctx,c,worker);if err!=nil{return "",err}
 if err=result.Validate(in);err!=nil{return "",err}
 tx,err:=s.pool.Begin(ctx);if err!=nil{return "",err};defer tx.Rollback(ctx)
 // The document is locked before its jobs, matching the trash trigger's order.
 var status string
 if err=tx.QueryRow(ctx,`SELECT status FROM documents WHERE id=$1 AND organization_id=$2 FOR UPDATE`,in.DocumentID,in.OrganizationID).Scan(&status);err!=nil{return "",err}
 if status=="Trash"||status=="Purged" {return "",job.ErrLeaseLost}
 var savedKey string
 var valid bool
 err=tx.QueryRow(ctx,`SELECT j.status,coalesce(o.object_key,''),
 ((j.status='Running' AND j.current_attempt_id=$2 AND j.lease_token_hash=$3 AND j.lease_expires_at>$4 AND j.worker_id=$5)
 OR (j.status='Completed' AND j.result->>'attempt_id'=$2 AND j.result->>'lease_hash'=$6 AND j.result->>'worker_id'=$5))
 FROM durable_jobs j JOIN document_ocr_runs o ON o.job_id=j.id WHERE j.id=$1 FOR UPDATE OF j,o`,c.JobID,c.AttemptID,leaseHash(c.LeaseToken),c.Now,worker,hex.EncodeToString(leaseHash(c.LeaseToken))).Scan(&status,&savedKey,&valid)
 if err!=nil{return "",err};if !valid{return "",job.ErrLeaseLost};if status=="Completed"{return savedKey,nil}
 if size<1||size>ocr.MaxResultBytes {return "",errors.New("invalid OCR artifact")}
 _,err=tx.Exec(ctx,`UPDATE document_ocr_runs SET superseded_at=$3 WHERE organization_id=$1 AND document_id=$2 AND published_at IS NOT NULL AND superseded_at IS NULL AND deleted_at IS NULL`,in.OrganizationID,in.DocumentID,c.Now);if err!=nil{return "",err}
 _,err=tx.Exec(ctx,`UPDATE document_ocr_runs SET object_key=$2,result_sha256=$3,result_bytes=$4,page_count=$5,published_at=$6 WHERE job_id=$1`,c.JobID,key,sha,size,len(result.Pages),c.Now);if err!=nil{return "",err}
 meta,_:=json.Marshal(map[string]any{"attempt_id":c.AttemptID,"worker_id":worker,"lease_hash":hex.EncodeToString(leaseHash(c.LeaseToken)),"duration_ms":result.DurationMS,"page_count":len(result.Pages)})
 _,err=tx.Exec(ctx,`UPDATE durable_jobs SET status='Completed',result=$2,completed_at=$3,current_attempt_id=NULL,lease_token_hash=NULL,lease_expires_at=NULL,worker_id=NULL WHERE id=$1`,c.JobID,meta,c.Now);if err!=nil{return "",err}
 _,err=tx.Exec(ctx,`INSERT INTO audit_events(organization_id,event_type,target_type,target_id,outcome,occurred_at) VALUES($1,'ocr.completed','document',$2,'succeeded',$3)`,in.OrganizationID,in.DocumentID,c.Now);if err!=nil{return "",err}
 return key,tx.Commit(ctx)
}

func (s *Store) OCRState(ctx context.Context,user,org,doc string)(ocr.State,error){
 d,err:=s.GetDocument(ctx,user,org,doc);if err!=nil{return ocr.State{},err};if d.Status=="Trash"||d.Status=="Purged"{return ocr.State{},tenant.ErrNotFound}
 var state ocr.State
 err=s.pool.QueryRow(ctx,`SELECT j.id,j.status,coalesce(j.failure_code,''),coalesce(o.page_count,0),coalesce(o.object_key,'') FROM document_ocr_runs o JOIN durable_jobs j ON j.id=o.job_id WHERE o.organization_id=$1 AND o.document_id=$2 AND o.deleted_at IS NULL ORDER BY j.created_at DESC,j.id DESC LIMIT 1`,org,doc).Scan(&state.JobID,&state.Status,&state.FailureCode,&state.PageCount,&state.ObjectKey)
 if errors.Is(err,pgx.ErrNoRows){return ocr.State{Status:"NotScheduled"},nil};return state,err
}

func(s *Store) RetryOCR(ctx context.Context,user,org,doc string,now time.Time)(ocr.State,error){
 tx,err:=s.organizationTx(ctx,user,org);if err!=nil{return ocr.State{},err};defer tx.Rollback(ctx)
 role,err:=currentRole(ctx,tx,user,org);if err!=nil||role!=tenant.Owner&&role!=tenant.Admin{return ocr.State{},tenant.ErrForbidden}
 var sha,status,model,pre string
 err=tx.QueryRow(ctx,`SELECT encode(content_sha256,'hex'),status FROM documents WHERE organization_id=$1 AND id=$2 FOR UPDATE`,org,doc).Scan(&sha,&status);if err!=nil||status=="Trash"||status=="Purged"{return ocr.State{},tenant.ErrNotFound}
 if err=tx.QueryRow(ctx,`SELECT model_version,preprocessing_version FROM ocr_settings WHERE singleton AND enabled`).Scan(&model,&pre);err!=nil{return ocr.State{},tenant.ErrForbidden}
 fp:=doc+":"+sha+":"+model+":"+pre
 var previous string
 err=tx.QueryRow(ctx,`SELECT j.id,j.status FROM document_ocr_runs o JOIN durable_jobs j ON j.id=o.job_id WHERE o.organization_id=$1 AND o.document_id=$2 AND o.fingerprint=$3 ORDER BY j.created_at DESC,j.id DESC LIMIT 1`,org,doc,fp).Scan(&previous,&status)
 if err!=nil&&!errors.Is(err,pgx.ErrNoRows){return ocr.State{},err}
 if previous!=""&&(status=="Queued"||status=="Running"||status=="Completed"){return ocr.State{JobID:previous,Status:status},nil}
 id:=postgresUUID();payload,_:=json.Marshal(map[string]string{"document_id":doc,"sha256":sha,"model_version":model,"preprocessing_version":pre})
 _,err=tx.Exec(ctx,`INSERT INTO durable_jobs(id,organization_id,requester_user_id,kind,status,payload,idempotency_key,max_attempts,available_at,created_at) VALUES($1,$2,$3,'ocr','Queued',$4,$5,3,$6,$6)`,id,org,user,payload,id,now);if err!=nil{return ocr.State{},err}
 _,err=tx.Exec(ctx,`INSERT INTO document_ocr_runs(job_id,organization_id,document_id,fingerprint,retry_of) VALUES($1,$2,$3,$4,nullif($5,'')::uuid)`,id,org,doc,fp,previous);if err!=nil{return ocr.State{},err}
 if err=auditTenant(ctx,tx,org,user,"ocr.retry","document",doc,now);err!=nil{return ocr.State{},err}
 return ocr.State{JobID:id,Status:"Queued"},tx.Commit(ctx)
}
