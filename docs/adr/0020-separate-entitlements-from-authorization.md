# แยก Entitlement ออกจาก authorization

การเข้าถึง capability ที่มีแผนกำกับต้องผ่าน server-side authorization และ Entitlement แยกกัน โดย frontend plan state และ Usage ไม่สามารถให้สิทธิ์ได้ การเปลี่ยนแผนต้องไม่ปิด login, การอ่านข้อมูลเดิมที่ยังมีสิทธิ์, notification opt-out, security controls, ownership transfer, tenant isolation, audit หรือการลบและส่งออกข้อมูลที่กฎหมายหรือสัญญากำหนด เพื่อไม่ให้ commercial policy กลายเป็นช่องโหว่หรือ lockout
