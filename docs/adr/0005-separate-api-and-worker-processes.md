# แยก API และ worker เป็นคนละ process

Go module เดียวจะมี `cmd/server` และ `cmd/worker` เป็นคนละ executable และ deployment เพื่อให้งานเบื้องหลังไม่เพิ่ม latency ของ webhook และสามารถ restart หรือ scale แยกกันได้ โดยยังไม่แยกเป็นหลาย repository หรือ microservice
