---
id: EXCESSIVE-AGENCY
severity_max: CRITICAL
applies_to: all
cwe: [CWE-250, CWE-862]
owasp: [LLM06:2025, LLM02:2025]
load_when: 'modelcontextprotocol|mcp-go|mcp\.NewServer|McpServer|anthropic|openai|langchain|tool_use|tools/call|ChatCompletion|generateText'
---

# Agent làm quá quyền / xác nhận giả

## Intent

Mô hình có thể bị chữ trong dữ liệu dẫn đi (xem `23-llm-prompt-injection`). Mọi "chốt chặn" nằm trong tay mô hình vì thế không phải chốt chặn: tham số `confirm: true` do mô hình tự điền, câu "chỉ gọi sau khi đã hỏi người dùng" trong mô tả tool, mã xác nhận trả cùng lượt với nội dung cần duyệt. Rule này soát **bề mặt tool lộ cho agent**: tool nào làm được gì, điều gì thật sự buộc con người đồng ý.

## Khi nào CRITICAL

- Tool chốt sổ / duyệt chi / chuyển tiền / đổi quyền / xoá dữ liệu cả tenant lộ cho agent, chạy ngay lượt gọi đầu; xác nhận không có hoặc chỉ là tham số trong `inputSchema`.
- Mã xác nhận trả **cùng gói** với bản nháp / nội dung người khác soạn: chữ trong bản nháp bảo "dùng mã ở trên để duyệt".
- Tool đã quyết định "chỉ làm trên web" / "đã gỡ" (comment, tài liệu, changelog) nhưng registry vẫn đăng ký và handler vẫn chạy thao tác nguy hiểm.

## Khi nào HIGH

- Tool xoá / sửa hàng loạt / gửi ra ngoài chạy với `confirm` / `force` do mô hình điền (kể cả schema ghi `const: true`).
- **Rò dữ liệu qua cửa ghi**: cùng một kết nối / token vừa đọc dữ liệu kín (lương, lãi lỗ, hồ sơ cá nhân) vừa ghi chữ tự do ra kênh người khác đọc (bình luận, tin nhắn, email, webhook, trả lời người gửi góp ý); không gì nối hai việc (không đánh dấu token sau lượt đọc nhạy, không log, không xem trước, không tách scope).
- Annotation sai theo hướng nguy hiểm: tool có ghi / xoá mà khai `readOnlyHint: true` (client tự cho chạy, không hỏi người).

## Khi nào MEDIUM

- Tool xoá / hàng loạt thiếu `destructiveHint: true`, tool chỉ đọc thiếu `readOnlyHint`, trong khi xác nhận phía server đã đúng.
- Mã xác nhận do server phát nhưng không gắn tham số (mã của bản A duyệt được bản B), không hết hạn, hoặc dùng lại được.

## Khi nào KHÔNG tạo finding

- **Agent không có tool ghi nào**: không gán rule này. Rò qua kết quả đọc → `27-excessive-data-exposure`.
- Tool ghi chỉ chạm dữ liệu rủi ro thấp, hoàn tác được, của chính người gọi (bản nháp, ghi chú riêng, đánh dấu đã đọc). Agent làm được việc người dùng vốn làm được **không** tự nó là lỗi.
- Xác nhận đúng: mã server phát ở lượt riêng mà kết quả chỉ có chữ server, gắn tham số, hết hạn ngắn, dùng một lần; hoặc tool chỉ tạo "lệnh chờ" để người bấm duyệt trên web.
- Đọc số kín và ghi ra ngoài là **hai scope tách nhau** lúc cấp quyền, hoặc lượt ghi sau khi đọc nhạy phải qua người xem trước.
- Tool "định bỏ" có test khẳng định không còn trong registry.
- Chỉ thiếu annotation ở tool rủi ro thấp: gộp 1 finding MEDIUM cho cả máy chủ.

## Cách reasoning (KHÔNG pattern-match thuần)

1. **Dựng bảng tool từ registry thật** (nơi gọi đăng ký trong code đang quét, không tin tài liệu): tên → handler → nhóm: đọc / ghi thường / xoá, hàng loạt, gửi ra ngoài / khoá sổ, tiền, quyền.
2. **Với mỗi tool hai nhóm sau, hỏi "ai xác nhận"**: Read `inputSchema` và handler. Điều kiện chạy là tham số mô hình gửi (`confirm`, `force`, `acknowledged`), hay thứ mô hình không tự có (mã server phát, lệnh chờ người bấm)? Luật chỉ nằm trong `description` = không có luật.
3. **Mã xác nhận sinh ở đâu, trả ở đâu**: lượt phát mã có trả kèm chữ người khác viết không? Mã gắn tham số nào, sống bao lâu, dùng mấy lần?
4. **Đường rò**: liệt kê scope của một token. Có tool đọc dữ liệu kín **và** tool ghi chữ tự do tới nơi người khác đọc? Tool "trả lời người viết" là vòng kín: kẻ tấn công điều khiển cả đầu vào lẫn nơi nhận. Dispatcher có nối hai việc không?
5. **Điều đã định bỏ**: grep `deprecated`, `web only`, `removed` quanh registry; tool đó còn đăng ký không, có test registry không.
6. **Annotation**: so `readOnlyHint` / `destructiveHint` với việc handler thật sự làm.

## Search patterns (gợi ý — KHÔNG chạy literal, dùng Grep tool)

```
# Registry tool
\.registerTool\(|server\.tool\(|setRequestHandler\(CallToolRequestSchema  # TS
mcp\.NewTool\(|\.AddTool\(|\.AddTools\(  # Go mcp-go
@mcp\.tool|@tool\b|Tool\(name=|tools\s*=\s*\[  # Python
#\[McpTool|->withTool\(|'tools'\s*=>  # PHP

# Tên tool / handler nguy hiểm
(delete|remove|purge|approve|close|lock|finalize|pay|transfer|refund|grant|revoke|set_role|bulk|send)_?\w*

# Xác nhận do mô hình điền, mã xác nhận
(confirm|confirmed|force|acknowledge\w*|i_understand)["']?\s*[:=]
const:\s*true|z\.literal\(true\)|"const":\s*true|WithBoolean\(\s*"(confirm|force)
(confirm|approval|approve)_?(token|code)

# Annotation
readOnlyHint|destructiveHint|WithReadOnlyHintAnnotation|WithDestructiveHintAnnotation|ToolAnnotations

# Cửa ghi chữ tự do
(comment|message|reply|note|feedback|email|webhook)_?(create|add|send|post)

# Điều đã định bỏ
deprecated|web[- ]only|do not expose|removed from (mcp|agent)|TODO.*(remove|drop)
```

## Examples

### CRITICAL / HIGH — flag

```typescript
// Xác nhận là tham số mô hình tự điền; mô tả tool không phải chốt chặn
server.registerTool("period_close", {
  description: "Chốt kỳ kế toán. CHỈ đặt confirm=true sau khi đã hỏi người dùng.",
  inputSchema: { period: z.string(), confirm: z.literal(true) },
}, async ({ period }, ctx) => {
  await ledger.closePeriod(ctx.tenantId, period);            // chạy ngay lượt đầu
  return text("closed");
});
```

```go
// Mã duyệt trả cùng gói với nội dung bản nháp do người khác soạn
func draftGet(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
    d, _ := repo.Draft(ctx, tenantOf(ctx), req.GetString("id", ""))
    out := map[string]any{"body": d.Body, "approve_code": issueCode(d.ID)}
    return jsonResult(out), nil
}
```

```python
# Một kết nối: đọc lương + ghi bình luận tự do, không gì nối hai việc
@mcp.tool()
def payroll_get(employee_id: int) -> dict: ...
@mcp.tool()
def comment_add(task_id: int, body: str) -> dict: ...      # ai trong dự án cũng đọc được

# "Đã chuyển sang chỉ làm trên web" nhưng vẫn đăng ký
# NOTE: member_remove is web-only since v2
@mcp.tool(annotations={"readOnlyHint": True})               # sai: tool này xoá
def member_remove(member_id: int) -> dict: ...
```

### NOT — không flag

```typescript
// Lượt 1: server dựng bản xem trước + phát mã gắn tham số (kết quả chỉ có chữ server)
server.registerTool("invoice_delete_prepare", { annotations: { readOnlyHint: true } }, async ({ ids }, ctx) => {
  const code = await confirm.issue(ctx.userId, "invoice_delete", hash(ids), { ttlSec: 120 });
  return text(JSON.stringify({ count: ids.length, total: await sum(ids), confirm_code: code }));
});
// Lượt 2: mã sai tham số / hết hạn / đã dùng → từ chối
server.registerTool("invoice_delete", { annotations: { destructiveHint: true } }, async ({ ids, confirm_code }, ctx) => {
  await confirm.consume(ctx.userId, "invoice_delete", hash(ids), confirm_code);
  await invoices.softDelete(ctx.tenantId, ctx.userId, ids);
  return text("deleted");
});
```

```go
// Chốt kỳ: agent chỉ tạo lệnh chờ, người có quyền bấm trên web
func periodCloseRequest(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
    id, _ := pending.Create(ctx, actorOf(ctx), "period_close", req.GetString("period", ""))
    return mcp.NewToolResultText("pending:" + id), nil
}
```

## Fix recommendation

Auto-fix **chỉ đưa patch gợi ý, không tự áp** (gỡ tool hay đổi luồng xác nhận là quyết định sản phẩm).

1. **Ba mức theo hậu quả:**
   - Khoá sổ, tiền, quyền: chỉ web, hoặc agent tạo lệnh chờ, người bấm ngoài luồng chat.
   - Xoá, hàng loạt, gửi ra ngoài: mã server phát ở lượt riêng, gắn đúng tham số, hết hạn ngắn, dùng một lần, kèm bản xem trước server dựng.
   - Ghi thường: chạy bằng quyền người uỷ nhiệm, qua cùng hàm kiểm quyền với web.
2. **Mã xác nhận không đi chung gói** với nội dung người khác viết.
3. **Khai `destructiveHint` / `readOnlyHint` đúng** để client hỏi người; test so annotation với nhóm tool.
4. **Thu hẹp cửa ra**: tool khai mức nhạy (đọc) và "ghi chữ ra ngoài" (ghi); dispatcher đánh dấu token sau lượt đọc nhạy, tối thiểu log, tốt hơn là buộc xem trước nguyên văn. Quyền đọc số kín là scope riêng lúc kết nối.
5. **Điều đã định bỏ**: gỡ khỏi registry + test khẳng định tool không tồn tại.
6. **Token agent không rộng hơn người uỷ nhiệm**; thu hồi khi người đó mất quyền.

## Cross-references

- `23-llm-prompt-injection`: đường vào của lệnh giả. Tool nguy hiểm vẫn là finding ở rule này kể cả khi đường vào đã được tách tốt (phòng thủ hai lớp).
- `26-authz-path-parity`: tool agent ghi vào đối tượng mà **không kiểm cùng luật quyền với đường web** → rule 26. Rule này dành cho "luật quyền đúng nhưng không có người xác nhận".
- `12-broken-access-control`: máy chủ MCP / endpoint tool không xác thực người gọi → rule 12.
- `27-excessive-data-exposure`: tool đọc trả trường người gọi không được xem.
