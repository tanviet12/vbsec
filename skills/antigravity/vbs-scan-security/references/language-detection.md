# Language Detection

Cách detect ngôn ngữ code chính trong repo để chọn rule overlay.

## Algorithm

1. **Lọc trước:** Bỏ các path không phải code:
   ```
   node_modules/, vendor/, dist/, build/, .next/, .nuxt/, target/, .venv/, __pycache__/, .git/,
   *.min.js, *.bundle.js, *.lock, *.lock.json, package-lock.json
   ```

2. **Count theo extension:**

   | Lang code | Extensions tính vào | Ghi chú |
   |---|---|---|
   | `go` | `.go` | |
   | `php` | `.php`, `.phtml` | |
   | `python` | `.py`, `.pyw` | bỏ `__init__.py` rỗng |
   | `typescript` | `.ts`, `.tsx`, `.js`, `.jsx`, `.mjs`, `.cjs` | gộp cả JS vào — vibe coders thường mix; patterns runtime giống nhau |
   | `ruby` | `.rb` | |
   | `java` | `.java` | |
   | `rust` | `.rs` | |
   | `dotnet` | `.cs`, `.csproj`, `.sln` | ASP.NET Core / EF Core; marker phụ: `global.json`, `appsettings.json` |
   | `kotlin` | `.kt`, `.kts` | |

3. **Primary language** = lang nào có **≥30% tổng file code** trong scope.

4. **Multi-language detection:** Nếu có ≥2 lang vượt 30% mỗi cái, treat repo là multi-lang. Apply overlay của TẤT CẢ lang đạt ngưỡng.

   Ví dụ phổ biến:
   - Backend Go + Frontend Vue/React → `go` + `typescript`
   - Laravel app → `php` (frontend Blade không tính)
   - Django + React → `python` + `typescript`

5. **Nếu không lang nào ≥30%:** Repo "polyglot" — chỉ dùng generic rules. Note trong header report: `"polyglot — using generic rules only"`.

6. **Overlay availability:**
   - Có `rules/languages/<lang>/` → load overlay
   - Không có → ghi `header_no_specialized` vào report header

## Phase chuyên sâu

| Lang | Status | Files trong `rules/languages/<lang>/` |
|---|---|---|
| `go` | ✅ v0.1 | GORM SQLi, command injection (exec.Command), slog secret leak, Colly SSRF |
| `php` | ✅ v0.1 | mysqli vs PDO, `$_GET/$_POST` direct, `eval`/`include` variable, Laravel CSRF, `unserialize` |
| `typescript` | ✅ v0.2 | Sequelize/Prisma/TypeORM/Mongoose SQLi+NoSQLi, React/Vue/Angular XSS, Express/NestJS/Next.js mass-assignment/SSRF/CSRF/CORS, js-yaml deserialize, child_process injection, JWT none/algorithm-confusion |
| `python` | ✅ v0.4 | SQLAlchemy text() + Django .raw/.extra, pickle/yaml.load (RCE), subprocess shell=True, Flask Werkzeug debugger RCE, Django DEBUG/FastAPI debug, Django ModelForm/Flask `**request.json`/FastAPI Pydantic mass-assignment, PyJWT algorithms allowlist, flask-cors/django-cors-headers/FastAPI CORSMiddleware, Django CSRF middleware |
| `dotnet` | ✅ v0.6 | ASP.NET Core / Minimal API model binding, EF Core raw SQL, Newtonsoft TypeNameHandling / legacy formatter deserialization, Process.Start command injection |
| Khác | Phase v0.7+ | Ruby, Java, Rust theo nhu cầu cộng đồng |

## Frontend framework detection (sub-classification)

Khi primary = `javascript` hoặc `typescript`, detect thêm framework để chỉnh sửa rule reasoning:

| Framework | Marker file | Ảnh hưởng đến rule |
|---|---|---|
| React | `package.json` có `"react"` | XSS: `dangerouslySetInnerHTML` mới là nguy hiểm (JSX auto-escape mặc định) |
| Vue | `package.json` có `"vue"` | XSS: `v-html` mới là nguy hiểm (mustache auto-escape) |
| Next.js | `next.config.*` | SSRF: kiểm tra `getServerSideProps`, `getStaticProps` |
| Express | `package.json` có `"express"` | CORS: middleware config; CSRF: `csurf` |
| Fastify | `package.json` có `"fastify"` | Tương tự Express |
| NestJS | `package.json` có `"@nestjs/core"` | Decorators (`@Body()`, `@Query()`) — Mass Assignment via DTO whitelist |

Framework markers chỉ là gợi ý — vẫn áp dụng generic rules cho mọi case. Markers chỉ dùng để **giảm false positive** (vd: không flag XSS cho mustache trong Vue vì auto-escape).

## Pseudocode

```python
def detect_primary_language(files):
    code_files = [f for f in files if not is_vendored(f)]
    total = len(code_files)
    if total == 0:
        return {"primary": None, "multi": [], "overlay_available": []}

    counts = {}
    for f in code_files:
        lang = ext_to_lang(f)
        if lang:
            counts[lang] = counts.get(lang, 0) + 1

    threshold = 0.30
    primaries = [lang for lang, n in counts.items() if n / total >= threshold]

    if not primaries:
        return {"primary": "polyglot", "multi": [], "overlay_available": []}

    overlay_available = [lang for lang in primaries if has_dir(f"rules/languages/{lang}")]

    return {
        "primary": primaries[0] if len(primaries) == 1 else "+".join(primaries),
        "multi": primaries if len(primaries) > 1 else [],
        "overlay_available": overlay_available
    }
```

LLM agent dùng Grep/Glob để count files (không cần chạy Python literal). Output dùng để fill `header_primary_lang` trong report.

## Edge cases

- **Monorepo** (vd: `apps/web` Next.js + `apps/api` Go): detect theo scope hiện tại. Nếu scope chỉ touch 1 sub-folder → detect theo sub-folder.
- **Generated files** (vd: `.pb.go`, `*_pb2.py`): vẫn count nhưng note trong report nếu chiếm >50%.
- **Test files**: count bình thường. Rule generic vẫn áp dụng (test code lộ secret cũng nguy hiểm).
- **Config files** (`.yaml`, `.json`, `.toml`, `Dockerfile`): KHÔNG count vào language stats nhưng vẫn scan bằng rules cụ thể (HARDCODED-SECRET, VERBOSE-ERROR-DEBUG-MODE applies cho config).
