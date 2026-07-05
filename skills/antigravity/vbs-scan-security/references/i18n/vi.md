# i18n — Tiếng Việt

Bảng key → text cho output report khi `lang=vi`. SKILL.md và workflows phải đọc file này và substitute keys vào output. KHÔNG hardcode chuỗi tiếng Việt vào logic.

## Header

| Key | Text |
|---|---|
| `report_title` | Báo cáo quét bảo mật vbsec |
| `header_scope` | Phạm vi |
| `header_files` | Số file |
| `header_primary_lang` | Ngôn ngữ chính |
| `header_mode` | Chế độ |
| `header_date` | Ngày quét |
| `header_lang` | Ngôn ngữ báo cáo |
| `header_using_generic` | dùng rule chung |
| `header_using_specialized` | dùng rule chuyên sâu |
| `header_no_specialized` | chưa có rule chuyên sâu |

## Scope labels

| Key | Text |
|---|---|
| `scope_uncommitted` | Thay đổi chưa commit |
| `scope_staged` | File đã staged |
| `scope_commit_within` | Commit trong {days} ngày gần đây |
| `scope_commit_id` | Commit {sha} |
| `scope_pr` | Pull Request #{number} |
| `scope_all` | Toàn bộ repo |

## Mode

| Key | Text |
|---|---|
| `mode_small` | NHỎ (quét trực tiếp) |
| `mode_large` | LỚN (chia tải qua sub-agent) |

## Verdict

| Key | Text |
|---|---|
| `verdict_label` | KẾT LUẬN |
| `verdict_pass` | ĐẠT |
| `verdict_warn` | ĐẠT CÓ CẢNH BÁO |
| `verdict_fail` | KHÔNG ĐẠT |
| `verdict_pass_desc` | Không phát hiện lỗi nghiêm trọng. Có thể deploy. |
| `verdict_warn_desc` | Có lỗi mức CAO cần khắc phục trước khi lên production. |
| `verdict_fail_desc` | Có lỗi NGHIÊM TRỌNG. KHÔNG được deploy đến khi sửa hết. |

## Severity

| Key | Text |
|---|---|
| `severity_critical` | NGHIÊM TRỌNG |
| `severity_high` | CAO |
| `severity_medium` | TRUNG BÌNH |
| `severity_low` | THẤP |
| `severity_critical_short` | Nghiêm trọng |
| `severity_high_short` | Cao |
| `severity_medium_short` | Trung bình |
| `severity_low_short` | Thấp |

## Section headers

| Key | Text |
|---|---|
| `section_critical` | NGHIÊM TRỌNG (chặn deploy) |
| `section_high` | CAO (cần khắc phục) |
| `section_medium` | TRUNG BÌNH |
| `section_low` | THẤP |
| `section_passed` | ĐÃ ĐẠT |
| `section_warnings` | CẢNH BÁO |
| `section_summary` | Tóm tắt |
| `section_recommendations` | Khuyến nghị tổng thể |

## Table headers

| Key | Text |
|---|---|
| `col_file_line` | File:Dòng |
| `col_rule` | Loại lỗi |
| `col_issue` | Mô tả |
| `col_fix` | Cách sửa |
| `col_severity` | Mức nguy hiểm |
| `col_recommendation` | Khuyến nghị |
| `col_count` | Số lượng |
| `col_status` | Trạng thái |

## Counts & summary

| Key | Text |
|---|---|
| `count_files_reviewed` | Số file đã quét |
| `count_rules_applied` | Số rule áp dụng |
| `count_findings` | Số phát hiện |
| `count_critical` | Lỗi nghiêm trọng |
| `count_high` | Lỗi cao |
| `count_medium` | Lỗi trung bình |
| `count_low` | Lỗi thấp |
| `no_findings` | Không phát hiện vấn đề |
| `all_passed` | Tất cả check đã đạt |

## Common phrases (cho fix recommendations — rule file có thể template)

| Key | Text |
|---|---|
| `phrase_use_parameterized` | Dùng parameterized query / prepared statement |
| `phrase_never_concat` | Không nối chuỗi trực tiếp |
| `phrase_use_env_var` | Chuyển sang biến môi trường, đừng commit secret |
| `phrase_rotate_now` | Xoay key ngay (nếu đã lộ lên Git) |
| `phrase_add_gitignore` | Thêm `.env` vào `.gitignore` |
| `phrase_validate_input` | Kiểm tra và lọc input trước khi dùng |
| `phrase_use_bcrypt` | Dùng bcrypt/argon2/scrypt cho mật khẩu |
| `phrase_use_csrf_token` | Thêm CSRF token cho endpoint thay đổi dữ liệu |
| `phrase_specific_origin` | Đặt origin cụ thể, đừng để `*` |
| `phrase_use_transaction` | Bọc trong transaction + lock để tránh race |
| `phrase_check_authz` | Thêm kiểm tra quyền ở backend, đừng tin frontend |
| `phrase_disable_debug_prod` | Tắt `DEBUG=true` trong môi trường production |
| `phrase_update_dep` | Update thư viện lên phiên bản fix CVE |
| `phrase_use_safe_yaml` | Dùng `yaml.safe_load` (Python) hoặc tương đương |
| `phrase_no_shell_true` | Không dùng `shell=True` với user input |
| `phrase_whitelist_redirect` | Whitelist domain redirect đích |

## System messages

| Key | Text |
|---|---|
| `msg_no_git` | Không phải git repository. Vui lòng `cd` vào thư mục git repo trước. |
| `msg_scope_needs_git` | Scope này yêu cầu git repository. Dùng skill không có scope (mặc định scan toàn bộ folder), hoặc khởi tạo git rồi gọi lại. Tip: `git init && git add .` |
| `msg_no_git_note` | ⚠️ Folder không phải git repository — đã quét tất cả file trong folder. Không bỏ qua theo `.gitignore` (vì không có git). |
| `msg_no_files` | Không có file nào trong phạm vi quét. |
| `msg_routing_small` | Chế độ NHỎ: quét trực tiếp |
| `msg_routing_large` | Chế độ LỚN: chia thành {n} chunk, spawn sub-agent |
| `msg_scanning` | Đang quét... |
| `msg_done` | Hoàn tất quét |
| `msg_invalid_lang` | Tham số `lang` không hợp lệ. Dùng `vi` hoặc `en`. Mặc định: `vi`. |

## Footer

| Key | Text |
|---|---|
| `footer_generated_by` | Báo cáo tạo bởi |
| `footer_repo` | https://github.com/tanviet12/vbsec |
| `footer_disclaimer` | Báo cáo này tham khảo — không thay thế cho audit bảo mật chuyên nghiệp. |
| `footer_next_steps` | Bước tiếp theo |
| `footer_next_steps_text` | Sửa các lỗi NGHIÊM TRỌNG trước. Sau đó re-scan để xác nhận. |

## Verbose finding blocks (v0.3+)

| Key | Text |
|---|---|
| `header_short_description` | Mô tả ngắn |
| `header_why_dangerous` | Tại sao nguy hiểm? |
| `header_attack_scenario` | Hacker khai thác như thế nào? |
| `header_current_code` | Code hiện tại (NGUY HIỂM) |
| `header_safe_code` | Code an toàn (sửa thành) |
| `header_impact` | Tác động |
| `header_fix_code` | Cách sửa |
| `header_read_more` | Đọc thêm |
| `header_overview_table` | Tổng quan (chi tiết phía dưới) |
| `phrase_input_untrusted` | input từ user — KHÔNG TIN |
| `phrase_concat_to_sink` | ghép thẳng vào |

## Persistence (v0.3+)

| Key | Text |
|---|---|
| `msg_report_saved` | Báo cáo đã lưu tại |
| `msg_gitignore_warning_title` | Khuyến nghị |
| `msg_gitignore_warning_text` | Thư mục `vbsec-reports/` chưa có trong `.gitignore`. Để tránh commit báo cáo vào Git, thêm dòng `vbsec-reports/` vào `.gitignore`. |
| `msg_reports_dir_created` | Đã tạo thư mục `vbsec-reports/` để lưu báo cáo |

## Hardening notes

| Key | Text |
|---|---|
| `header_hardening_title` | Gợi ý tăng cường (không phải lỗ hổng) |
| `msg_hardening_intro` | Các điểm dưới đây không khai thác được, chỉ là gợi ý phòng thủ thêm. Không tính vào kết quả. |

## Auto-fix (`--auto-fix`, v0.7+)

| Key | Text |
|---|---|
| `header_autofix_title` | Kết quả tự động sửa lỗi |
| `msg_autofix_needs_git` | ⚠️ `--auto-fix` yêu cầu git repository (cần khả năng revert an toàn khi build fail). Bỏ qua bước auto-fix. |
| `msg_autofix_dirty_tree` | ⚠️ Working tree có thay đổi chưa commit ngoài phạm vi quét. Khuyến nghị commit hoặc backup trước khi chạy `--auto-fix`. |
| `msg_autofix_snapshot_scope` | ℹ️ Scope `commit id`/`pr id` quét trên snapshot tạm, không phải working tree — auto-fix chỉ ghi gợi ý patch ra `vbsec-reports/patches/`, không sửa file nào. |
| `autofix_status_applied` | Đã sửa và verify build thành công |
| `autofix_status_failed` | Không sửa được sau {n} lần thử — cần sửa tay |
| `autofix_status_suggested` | Chỉ tạo gợi ý patch (chưa có lệnh build để verify) — xem `vbsec-reports/patches/` |
| `autofix_status_skipped` | Bỏ qua (severity MEDIUM/LOW, không auto-fix) |
| `msg_autofix_summary` | Đã tự sửa {applied}/{total} lỗi CRITICAL+HIGH. {failed} lỗi cần sửa tay, {suggested} lỗi chỉ có gợi ý patch. |

## SCA / Live CVE (`--sca`, v0.7+)

| Key | Text |
|---|---|
| `header_sca_title` | Quét dependency (SCA — OSV.dev live) |
| `msg_sca_unavailable` | ⚠️ Không kết nối được OSV.dev — dùng static list offline (rule OUTDATED-DEPENDENCY) thay thế. |
| `msg_sca_no_manifest` | Không tìm thấy manifest dependency cho ngôn ngữ {lang} — bỏ qua SCA cho ngôn ngữ này. |
| `col_cve` | CVE |
| `col_installed_fixed` | Version hiện tại → Version fix |
