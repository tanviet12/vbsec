# Chunking Strategy

Cách chia file thành chunks cho LARGE mode (sub-agent delegation). Mỗi chunk được 1 sub-agent quét song song.

## Mục tiêu

- Sub-agent có **scope rõ ràng**: 1 chunk = 1 phần logic của repo (không phải 1 file lẻ)
- Mỗi chunk có **đủ context** để reasoning (không bị cắt mid-function)
- **Cân bằng tải**: chunks tương đương kích cỡ
- **Không quá nhiều chunks**: target 3-10 chunks (sub-agent overhead cao, đừng có 50 chunk 2 file mỗi cái)

## Strategy mặc định (v0.8+): Lát dọc theo khu nghiệp vụ

Chia theo **thư mục cấp một** cắt đứt luồng dữ liệu: với kiến trúc nhiều tầng (handler → core/service → repo ở thư mục khác nhau, cộng tool agent / MCP ở chỗ thứ tư), agent con quét `handlers/` không thấy repo lọc tenant thế nào, agent quét `repo/` không biết input đến từ đâu. Lỗi đắt nhất (tenant, quyền lệch giữa các đường, lộ dữ liệu, logic trạng thái) nằm đúng ở chỗ nối đó.

Nên mỗi chunk là **một lát dọc**: mọi file của một khu nghiệp vụ, đủ các tầng.

1. **Tìm khu nghiệp vụ** bằng tên chung xuyên tầng: `handlers/invoice.go`, `service/invoice.go`, `repo/invoice_repo.go`, `mcp/tools/invoice.ts`, `migrations/*invoice*` → khu `invoice`. Nguồn gợi ý: file đăng ký route (`router.go`, `routes/*.ts`), tên bảng trong migration, thư mục `modules/<khu>` / `features/<khu>` / `domain/<khu>` nếu repo đã chia sẵn (khi đó dùng luôn).
2. **Gán file vào khu**: theo tên (stem chung), rồi theo import (handler import service nào). File dùng chung (middleware auth / tenant, db helper, config, logger) → gắn vào **mọi** lát dọc dưới dạng "đọc tham khảo" (không tính vào file_count), và vào chunk liên khu.
3. **Chunk liên khu** (`kind: cross-cutting`), luôn có khi repo có ≥ 2 khu: nhận danh sách mọi file mã, chỉ soát TENANT-ISOLATION, AUTHZ-PATH-PARITY, EXCESSIVE-DATA-EXPOSURE, SENSITIVE-DATA-IN-LOGS và luật dự án — so sánh các đường ghi / đọc cùng đối tượng giữa các khu (xem prompt trong [`sub-agent-prompts.md`](sub-agent-prompts.md)).
4. File không thuộc khu nào (script, cấu hình, frontend thuần UI) → gom theo thư mục như strategy cũ bên dưới.

Không tìm được khu (repo phẳng, ít tầng) → dùng strategy theo thư mục bên dưới.

## Strategy dự phòng: Top-level folder

Chia theo **top-level folder** dưới git root, mỗi folder = 1 chunk.

```
my-repo/
├── api/           ← chunk 1
├── web/           ← chunk 2
├── shared/        ← chunk 3
└── scripts/       ← chunk 4
```

File ở root (vd: `package.json`, `Dockerfile`, `.env.example`) → 1 chunk riêng tên `root`.

## Rules để cân bằng

1. **Chunk có >50 file** → split thành chunk nhỏ hơn theo sub-folder
   ```
   api/  (120 files) → split:
     ├── api/handlers/  ← chunk
     ├── api/middleware/ ← chunk
     └── api/services/   ← chunk
   ```

2. **Folder có <5 file** → merge vào chunk khác cùng tính chất
   ```
   utils/ (3 files) + helpers/ (4 files) → 1 chunk "utils+helpers"
   ```

3. **Detected primary language ≠ folder language** → ưu tiên giữ chunk theo lang để sub-agent có thể apply đúng overlay
   - Vd: `frontend/` toàn `.ts`, `backend/` toàn `.go` → 2 chunk, mỗi cái có lang riêng

4. **Test files**: gộp `__tests__/`, `*_test.go`, `*.spec.ts` vào CHUNK CHA của code chúng test. Lý do: rule áp dụng cho test code khác với prod code (vd: HARDCODED-SECRET trong test với fixture data thường không phải critical).

## Strategy thay thế: Theo extension cluster

Khi repo không có cấu trúc folder rõ (flat repo, mix mọi thứ ở root), chia theo extension cluster:

```
flat-repo/
├── *.go (8 files)        ← chunk "go"
├── *.py (5 files)        ← chunk "python"
├── *.js (3 files)        ← chunk "js"
├── Dockerfile, .env*     ← chunk "config"
```

## Strategy theo file count

| Tổng files | Số chunks target | Files/chunk |
|---|---|---|
| 30-60 | 3-5 | ~10-15 |
| 60-150 | 5-8 | ~15-25 |
| 150-300 | 8-12 | ~20-30 |
| >300 | 12-15 | ~25-30 |

KHÔNG quá 15 chunks. Sub-agent spawn overhead + main-agent aggregate context = chậm hơn nếu split nhỏ.

## Pseudocode

```python
def chunk_files(files, max_files_per_chunk=30, target_chunks_max=15):
    # 1. Group by top-level folder
    groups = {}
    for f in files:
        top = f.split("/")[0] if "/" in f else "root"
        groups.setdefault(top, []).append(f)

    # 2. Split big chunks
    final = []
    for folder, fs in groups.items():
        if len(fs) > max_files_per_chunk:
            # Split by sub-folder
            sub_groups = {}
            for f in fs:
                parts = f.split("/")
                sub = parts[1] if len(parts) > 1 else parts[0]
                sub_groups.setdefault(f"{folder}/{sub}", []).append(f)
            for sub_folder, sub_fs in sub_groups.items():
                final.append({"name": sub_folder, "files": sub_fs})
        else:
            final.append({"name": folder, "files": fs})

    # 3. Merge tiny chunks
    tiny = [c for c in final if len(c["files"]) < 5]
    if len(tiny) >= 2:
        merged_name = "+".join(c["name"] for c in tiny)
        merged_files = [f for c in tiny for f in c["files"]]
        final = [c for c in final if len(c["files"]) >= 5]
        final.append({"name": merged_name, "files": merged_files})

    # 4. If still too many chunks, merge sequentially
    while len(final) > target_chunks_max:
        final.sort(key=lambda c: len(c["files"]))
        c1, c2 = final.pop(0), final.pop(0)
        final.append({"name": f"{c1['name']}+{c2['name']}", "files": c1["files"] + c2["files"]})

    return final
```

LLM agent dùng Glob + lý luận để chia, không cần chạy Python literal. Pseudocode trên là strategy dự phòng; với lát dọc, bước 1 thay `top = f.split("/")[0]` bằng "khu nghiệp vụ của f" (stem chung / import), các bước cân bằng (split chunk > 30 file theo khu con, merge khu < 5 file) giữ nguyên.

## Output format cho main orchestrator

Sau khi chunk, main agent có list dạng:

```
[
  {"name": "api/handlers", "files": ["api/handlers/user.ts", "api/handlers/order.ts", ...], "count": 12},
  {"name": "api/middleware", "files": [...], "count": 8},
  {"name": "frontend/src/components", "files": [...], "count": 25},
  ...
]
```

Mỗi chunk được TodoWrite tạo 1 task. Sub-agent nhận chunk + prompt template (xem [`sub-agent-prompts.md`](sub-agent-prompts.md)).

## Edge cases

| Scenario | Cách xử lý |
|---|---|
| Repo có submodule | Skip submodule (không scan). Note trong report. |
| Generated code chiếm 80% chunk | Split: chunk "generated" (low priority) + chunk source. Có thể skip "generated" nếu user confirm. |
| Monorepo với 50+ apps | Chunk theo app (ưu tiên top-level `apps/<name>`), không split deeper. |
| 1 file >5000 dòng | Để nguyên 1 chunk (đừng split 1 file). Note "large file, đọc kỹ". |
| Repo siêu nhỏ (10 files) | Không LARGE mode — fallback SMALL. SKILL.md đã guard điều này. |
