# Test fixtures

Bộ code mẫu có lỗi biết trước, dùng để đo skill bắt được bao nhiêu lỗi (recall) và báo nhầm bao nhiêu (false positive) sau mỗi lần sửa rule.

```
tests/
├── fixtures/<lang>/      # câu dễ: python, typescript, go, php, dotnet
├── fixtures/<lang>-hard/ # câu khó: go-hard, typescript-hard, php-hard
└── expected/<set>.json   # đáp án cho từng bộ
```

Code trong `fixtures/` **cố tình có lỗ hổng**. Không copy đi dùng, không deploy.

## Nguyên tắc viết fixture

- **Không để lộ đáp án trong code.** Tên file trung tính (`users.py`, không phải `sqli_vuln.py`), không comment kiểu `// vulnerable`. Skill phải tự suy luận như với code thật.
- **Đáp án để ngoài thư mục được quét** (`tests/expected/`), để skill không đọc được.
- **Mỗi bộ có cả bẫy false positive:** file trông giống lỗi nhưng an toàn (query có bind param, `yaml.safe_load`, `htmlspecialchars`...).
- **Secret giả không theo format thật** của nhà cung cấp (Stripe, AWS...), để không bị GitHub push protection chặn.

## Câu dễ và câu khó

- **Câu dễ** (`<lang>/`): lỗi dạng sách giáo khoa, input đi thẳng từ request vào chỗ nguy hiểm trong cùng 1 hàm. Dùng để bắt regression thô.
- **Câu khó** (`<lang>-hard/`): lỗi giống code thật. Input đi qua 2-3 file (handler → service → repo), hàm "làm sạch" viết sai (`HasPrefix` sau `Join`, `endsWith("example.com")`, check `..` trước `decodeURIComponent`), second-order SQL injection, race condition khi trừ số dư, middleware phân quyền đặt sai chỗ. Kèm bẫy an toàn trông rất giống lỗi (`fmt.Sprintf` với allowlist, `Prisma.sql`, `DOMPurify`).

## Đáp án (`expected/<set>.json`)

| Field | Ý nghĩa |
|---|---|
| `must_find` | Finding bắt buộc có: `file` + `rule_id` (string hoặc list các ID chấp nhận được) + `min_severity`. Thêm `lines: [start, end]` khi 1 file có cả đoạn lỗi lẫn đoạn an toàn cùng loại |
| `must_not_find` | Finding KHÔNG được có, kèm `reason` |
| `clean_files` | File không được có finding nào |

Finding ngoài đáp án (vd `BRUTE-FORCE` ở endpoint login) không tính điểm, script in ra để xem tay. Nếu hợp lý và ổn định qua nhiều lần chạy, cân nhắc thêm vào `must_find`.

## Chạy

```bash
# Scan bằng Claude Code headless rồi chấm (mỗi ngôn ngữ = 1 lần scan thật, tốn token)
./scripts/run-fixtures.sh
./scripts/run-fixtures.sh python php

# Chỉ chấm report mới nhất đã có (vd sau khi tự chạy /vbs-scan-security trong thư mục fixture)
./scripts/run-fixtures.sh --check-only
python3 scripts/check-fixtures.py python --report tests/fixtures/python/vbsec-reports/scan-xxx.md
```

Scope `all` dùng `git ls-files`, nên fixture phải được commit trước khi scan. Report lưu ở `tests/fixtures/<lang>/vbsec-reports/` (đã gitignore).

Kết quả của LLM không cố định giữa các lần chạy. Khi so trước/sau một thay đổi rule, nên chạy 2-3 lần.

## Lưu ý khi quét chính repo vbsec

`/vbs-scan-security all` ở root repo sẽ quét cả `tests/fixtures/` và báo rất nhiều lỗi. Đó là lỗi cố ý, không phải lỗi của vbsec.
