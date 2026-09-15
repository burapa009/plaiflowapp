# ใช้ Go standard library ร่วมกับ pgx

Go API จะใช้ `net/http`, `http.ServeMux`, `log/slog` และ crypto packages จาก standard library ใช้ `pgx/v5` สำหรับ PostgreSQL และไม่ใช้ web framework, ORM หรือ dependency-injection framework เพื่อให้ boundary ของ Phase 0 ตรงไปตรงมาและลด dependencies ที่ต้องดูแล
