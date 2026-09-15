# แยก provider configuration ตาม environment และวัตถุประสงค์

แต่ละ environment ใช้ LINE Provider ของตนเองและวาง LINE Login channel กับ Messaging API channel ของ environment เดียวกันไว้ใต้ Provider นั้นเพื่อให้ user ID เชื่อมโยงกันได้ ส่วน Google ใช้ Cloud project แยก dev, staging และ production และแยก Login ออกจาก Google Drive connector ในอนาคต Phase 1 ขอเพียง `openid email profile` แบบ online โดยไม่ขอ refresh token, Drive scope หรือ `include_granted_scopes`
