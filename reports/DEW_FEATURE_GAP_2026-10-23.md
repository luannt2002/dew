# DEW: còn thiếu gì cho prompt hằng ngày và AI agent — 23/10/2026

Mã soát: `main` @ `2b87e938` (v0.3.3). Audit chỉ đọc; tệp duy nhất được ghi là tệp này.
Cách làm:
- 4 agent chỉ đọc: kiểm kê năng lực, đo cách dùng thật, quét công cụ khác + upstream, điểm đau tính năng cũ;
- tôi tự chạy lại các số mà kết luận dựa vào (mục 8);
- một agent soát độc lập (không có quyền sửa) đã chấm bản nháp KHONG-DAT, bản này đã sửa theo đó.

Không build, không chạy `dew`, không đọc env/credentials, không in nội dung prompt.

Tiếp nối: [DEW_AUDIT_2026-10-21](DEW_AUDIT_2026-10-21.md) (thiếu "daily"), [DEW_AUDIT_2026-10-22](DEW_AUDIT_2026-10-22.md)
(hướng đi), [DEW_CODE_AUDIT_2026-10-22](DEW_CODE_AUDIT_2026-10-22.md) (mã). Báo cáo này không lặp lại các mục đã sửa.

## Kết luận

1. **Việc tương tác hằng ngày đang nằm ở Claude Code, không ở dew.** 14 ngày qua có 543 lượt người gõ vào Claude Code,
   trên 11 ngày. Lần cuối gõ vào TUI dew là 20/10, đúng ngày bắt đầu dùng routine.
   - **Chưa đo nguyên nhân.** Khác biệt lớn nhất nhìn thấy là model: Claude Code chạy model mạnh, 99% tin nhắn ở effort
     high trở lên; dew chỉ chạy qwen3.8.
   - Khác biệt đó chưa được chứng minh là lý do. Không tính năng nào bù được khoảng cách model.
2. **Dew thực tế đã là một bộ chạy việc nền do Claude Code điều khiển.**
   - 218 lượt routine trong 4 ngày; 191 định nghĩa đều `goal`. Routine `goal` chỉ chạy tay theo thiết kế, nên
     "0 lượt theo lịch" không nói gì về ý muốn của bạn.
   - Claude Code gọi `dew` 217 lần trong cùng 4 ngày, 135 lần là `dew ps` (hỏi vòng xem xong chưa). Chỉ 13–19 lệnh
     (tuỳ cách đếm) là `routine run` trực tiếp, so với 218 lượt. Có thể (chưa chứng minh) đa số lượt được khởi động
     qua một launcher trung gian.
3. **Phần tương tác cơ bản dew đã có gần đủ.** Gồm: lệnh tự định nghĩa, `@`, `!`, hàng đợi + lái giữa lượt, plan mode,
   tool hỏi lựa chọn, todos, MCP, LSP, job nền, thông báo.
   - Những thứ bạn dùng trong Claude Code mà dew chưa có: subagent tự định nghĩa (6 agent trong `~/.claude/agents`;
     tối đa ~70/565 lần gọi subagent không phải loại dựng sẵn) và hook đủ sự kiện (bạn tự cài PostToolUse; SessionStart, UserPromptSubmit,
     SubagentStart đến từ plugin).
   - Dew cũng chưa có quay lui (rewind) và memory ghi được; chưa đo bạn dùng hai thứ này trong Claude Code bao nhiêu.
4. **Đề xuất: chốt hướng "engine chạy nền, Claude Code là buồng lái"**, rồi làm theo thứ tự số đo:
   - (a) hàng đợi nhẹ thay cho thoát 75 khi hết slot;
   - (b) bám (attach) vào lượt do launcher khởi động, thay cho hỏi vòng `ps`;
   - (c) cổng soát độc lập tự động sau GREEN: 76/154 lượt GREEN có sửa tệp test hoặc tệp mà verify dùng, dew chỉ in
     cảnh báo;
   - (d) lấy các bản sửa upstream nhắm đúng lỗi stream chậm.

   Phần tương tác chỉ sửa lỗi rẻ (tệp ngữ cảnh viết thường bị bỏ qua), chưa đầu tư thêm cho tới khi bạn quay lại gõ
   vào dew.
5. Đây là quyết định sản phẩm của bạn (đã treo từ 22/10). Mục 5 ghi rõ phần nào phụ thuộc quyết định đó.

## 1. Số đo: bạn thật sự làm việc thế nào (10–23/10)

| Chỉ số | Giá trị |
|---|---|
| Claude Code: lượt người gõ / số ngày có gõ | **543 / 11** (VS Code chiếm phần lớn) |
| Claude Code: model của tin nhắn chính | Opus 5.5 ~67%; effort high/xhigh/max ~99% |
| Claude Code: gọi subagent | 565 lần (Plan 298, general-purpose 197); 75% tool call nằm trong subagent |
| Claude Code: Workflow / Skill / AskUserQuestion / slash command | 16 / 24 / 22 / 15 (9 là `/model`) |
| Claude Code: hook | tự cài: PostToolUse (Write\|Edit); qua plugin: SessionStart, UserPromptSubmit, SubagentStart |
| Claude Code: memory | 72 tệp ở dự án chính, 33 tạo trong cửa sổ (12 loại feedback) |
| dew TUI: tin nhắn người gõ ở dự án thật | 105 tin, 4 ngày; **lần cuối 20/10 14:21** |
| dew routine: lượt / định nghĩa | **218 lượt** (20–23/10) / **191 định nghĩa, 100% `goal`** (goal chỉ chạy tay theo thiết kế) |
| dew routine: lý do dừng | GREEN 154 · RUN_ERROR 29 · CANCELLED 15 · TIMEOUT 14 · STALLED 2 · RUNNING 2 (báo cáo cũ 20/10, không có pid) · STREAM_STALL 1 · NO_CHANGE 1 |
| dew routine: GREEN có sửa tệp test/verify | **76/154** (`touched_tests`; dew chỉ in cảnh báo) |
| dew routine: thời lượng | p50 21 phút, p90 45 phút. Trùng trần 45 phút của *một lượt agent* (`run_timeout`), vì đa số lượt chỉ có 1 lượt agent |
| Định nghĩa đặt `run_timeout` / `max_duration` | 0/191 / 0/191 (cờ có sẵn ở `routine new`) |
| Gọi `dew` từ Claude Code | 217 lệnh, chỉ 20–23/10; `ps` 135, `routine run` 13–19 (đếm bằng regex, xấp xỉ) |
| TTFT theo số routine chạy cùng lúc | 1: p50 1,5 s / p90 4,0 s · 2: 1,9 / 18,4 s · ≥3: 8,5 / **59,6 s** (vẫn thấy khi tách theo cỡ prompt) |
| Định nghĩa là bản thử lại (`-rN`) | 42/191; goal trung vị ~4,1k ký tự |
| Upstream đi trước điểm fork | 43 commit, 123 tệp |
| GitHub Actions trên fork | 0 lượt chạy |

Đọc số:
- Dew đang làm đúng loại việc "máy móc, kiểm được, lặp lại", nhưng cách điều khiển là Claude Code hỏi vòng `dew ps`.
- Trần thật là gateway: từ 3 lượt song song, TTFT p90 tăng gần 15 lần. `DEW_ROUTINE_SLOTS=2` mặc định là đúng.

## 2. Dew đã có gì, thiếu gì (kiểm 37 mục trong mã)

**Đã có, không cần đề xuất:**
- lệnh tự định nghĩa (đọc cả `~/.claude/commands`), bảng lệnh, `@` tệp + dán ảnh, `!` chạy shell;
- hàng đợi prompt + lái giữa lượt, tóm tắt ngữ cảnh tự động, plan mode (`shift+tab`), tool `question`, `todos`;
- skill (đọc cả `~/.claude/skills`), MCP (stdio/sse/http + OAuth), LSP (8 tool), web (`fetch`, `agentic_fetch`);
- job nền (`alt+j`, `dew jobs`);
- headless: `run --json`, `send`, server + OpenAPI, stream sự kiện SSE `GET /v1/workspaces/{id}/events`;
- nhiều dự án (Home, `ps`, `projects`), loop/routine đầy đủ vòng đời;
- editor ngoài + clipboard + nháp theo session, log, journal JSONL của routine (ghi `Sync` từng dòng), ledger.

| # | Năng lực | Trạng thái | Chỗ thiếu | Bằng chứng |
|---|---|---|---|---|
| 7 | Quay lui (rewind/checkpoint) | THIẾU | có lưu phiên bản tệp nhưng không có lệnh khôi phục | `internal/history/file.go:28-42` |
| 13 | Subagent | MỘT PHẦN | chỉ agent dựng sẵn (coder/task/plan), không đọc định nghĩa của người dùng, không cô lập worktree. Có `claude_code`/`codex` (tắt mặc định) dùng được như "model thứ hai" | `internal/config/config.go:862` (`Agents json:"-"`), `:1106-1136` |
| 14 | Hook | MỘT PHẦN | chỉ `PreToolUse` và `Stop` | `internal/hooks/hooks.go:14-16` |
| 16 | Memory | MỘT PHẦN | tệp ngữ cảnh ở gốc repo (kể cả `.cursor/rules/`, `.github/copilot-instructions.md`) + 2 tệp toàn cục; không có memory ghi được; 7 tên viết thường bị bỏ qua (P27) | `internal/config/load.go:610-618`, `:661-664` |
| 6 | Session | MỘT PHẦN | không fork/nhánh session; không xuất markdown (chỉ JSON hoặc bản in có màu của `session show`) | `internal/cmd/session.go:265-290` |
| 10 | Quyền | MỘT PHẦN | luật chỉ theo `tool` hoặc `tool:action`, không theo mẫu lệnh; accept-edits có mã trong TUI nhưng cờ bật bị tắt ở bản chạy thật, chỉ dùng được qua API | `internal/permission/permission.go:218-221`, `internal/ui/model/ui.go:4957-4960` |
| 21 | Thông báo | MỘT PHẦN | chỉ từ TUI khi cửa sổ mất focus; không có gì khi routine/loop xong | `internal/ui/notification/` |
| 25 | Headless | MỘT PHẦN | CLI `--json` chỉ in một object cuối; stream đã có qua SSE của server và journal JSONL, nhưng chưa có cờ CLI | `internal/cmd/run.go:258-266`, `internal/server/endpoints.go:103-111` |
| 30 | Điều phối nhiều agent | MỘT PHẦN | chỉ gọi `agent` song song; không có fan-out/workflow | `internal/agent/agent_tool.go:40` |
| 31 | Repo map | THIẾU | chỉ grep/glob/LSP | `internal/config/config.go:1026-1059` |
| 34 | Sandbox hệ điều hành | THIẾU | guard danh sách chặn; README tự nhận không phải sandbox | `README.md:257` |

## 3. Công cụ khác có gì đáng lấy (theo agent quét web; số phiên bản chưa tự kiểm)

| Ý tưởng | Có ở | Hợp với dew vì |
|---|---|---|
| Checkpoint bằng git bóng (repo riêng ngoài dự án), khôi phục mã/hội thoại | Claude Code `/rewind`, Gemini `/restore`, OpenCode `/undo`, Aider `/undo` | model nhỏ đi sai nhiều hơn; chạy được cả khi cây bẩn |
| Subagent định nghĩa bằng markdown (prompt, model, quyền) | Claude Code, OpenCode, Codex | một reviewer chạy model thứ hai = bước soát độc lập của bạn |
| Đánh giá mục tiêu bằng model độc lập, có kết luận "không thể" | Claude Code `/goal`, Codex goal | dừng vòng lặp vô vọng sớm, đỡ tốn GPU |
| Hook đủ sự kiện (PostToolUse, SessionStart, Notification, StopFailure…) | Claude Code, Gemini, Codex | luật tất định; skill chỉ được dùng 2/41 lượt (đo trong đợt Daily, `reports/daily-work/07_DAILY_WORK_GAPS.md` G-04) |
| Monitor: job nền đẩy từng dòng thành sự kiện | Claude Code | dew đã có journal JSONL; chỉ cần chỉ cách `tail -F` |
| Sandbox bubblewrap/Landlock, chỉ ghi trong workspace | Codex, Claude Code | thay danh sách chặn bị lách mỗi lượt soát |
| Hiện hạn mức, báo trước khi chạm trần | Pitboard (cho Claude Code/Codex) | dew từng chết `QUOTA` khi GLM hết hạn mức 5 giờ |

Upstream Crush có 43 commit mới, không có tính năng lớn. Đáng lấy trước, vì nhắm đúng lỗi đo được ở mục 1:
- `1f5d7f0b`: chỉ đo khoảng im giữa các token; token đầu được chờ ít nhất 5 phút.
  - Nhắm STREAM_STALL/RUN_ERROR khi prefill chậm (journal: 28 request `stream_stall`).
  - Đụng `config.go`, `schema.json`; xung đột với `request_timeout.go` (fork đã đổi +119 dòng).
- `510f6ea3`: job đã xong không chiếm ngân sách 50 job.
  - Xung đột với `background.go` (fork +243 dòng).
  - **Có đổi `internal/skills/builtin/crush-config/SKILL.md`, có thể làm đỏ cassette `TestCoderAgent`.** Phải kiểm
    trước khi lấy.
- `7dadde0b`: dò model của provider tự định nghĩa không làm treo lúc khởi động.
- `623ba438`: không để tool call chờ một câu trả lời không bao giờ tới.
- `17e428f6`: Ctrl+C ngắt được lúc khởi động.
- `1dbd303a`: không tạo thư mục cha khi quyền ghi bị từ chối.

## 4. Điểm đau của tính năng cũ còn mở (đã kiểm ở HEAD)

| id | Điểm đau | Bằng chứng ở HEAD |
|---|---|---|
| P27 | **Tệp ngữ cảnh viết thường không bao giờ được nạp trên Linux.** 7 tên chết: `agents.md`, `Agents.md`, `crush.md`, `Crush.md`, `crush.local.md`, `Crush.local.md`, `gemini.md`. Nguyên nhân: danh sách được sắp theo byte (chữ hoa trước), rồi khử trùng lặp theo khoá viết thường *trước khi* biết tệp có tồn tại. **Lỗi kế thừa từ upstream** (có ở `a73e1bfa` và upstream main). Né không cần mã: thêm `"./agents.md"` vào `context_paths` | `internal/config/load.go:661-664`, `internal/agent/prompt/prompt.go:157-161` |
| P13 | GREEN chỉ là verify thoát 0. 76/154 lượt GREEN có sửa tệp test hoặc tệp verify dùng, dew chỉ in cảnh báo | `internal/cmd/routine.go:945`, `:1216` |
| P11 | Không có tín hiệu sống/treo của lượt (tuổi sự kiện cuối, tool cuối); phải tự đọc `crush.db`. Đã có `attempts`, `changed_count` | trường `routine list --json` ở `internal/cmd/routine.go:545-563` |
| P06/P12 | Hết slot thì thoát 75, không chờ; hàng đợi phải dựng ngoài dew | `internal/cmd/routine.go:867-881` |
| P05 | Trần 45 phút là của *mỗi lượt agent*. `routine new --run-timeout/--max-duration` đã có nhưng 0/191 định nghĩa dùng; 14 lượt TIMEOUT | `internal/cmd/routine.go:166` |
| P15 | Lỗi nội bộ `message content is shorter than read bytes` giết một lượt, chưa rõ gốc | `internal/app/app.go:465-467` |
| P18 | Lượt trước của *cùng* routine kết thúc trong cùng giây với lượt sau bắt đầu thì hai lượt ghi chung `runs/<stamp>` (khoá theo routine có trước stamp nên chỉ ca này) | `internal/cmd/routine.go:1030`, `:1086` |
| P21/22 | Guard còn lách qua tiến trình con (`env -C`, `make -C`, python `chdir`, `bash x.sh`); 9 ghi chú `shortcut:` về an toàn | `internal/shell/unattended.go:76` |
| P26 | Model thuộc workspace, không thuộc session | `internal/ui/model/model_restore.go:11-35` |
| P35 | "Modified Files" bỏ sót thay đổi qua bash | `internal/ui/model/session.go:135-143` |
| P40/41 | GitHub Actions trên fork: 0 lượt chạy; upstream đi trước 43 commit, chưa merge lần nào | GitHub API (mục 8) |

Cổng WIN (10 kịch bản tương tác): 0/10 đã đi. Đọc mã cho thấy:
- 7 kịch bản "có thể đạt";
- 3 kịch bản "chưa rõ": sửa tệp + diff, delegation, lỗi mạng giữa lượt;
- 0 kịch bản "chắc trượt".

Vẫn cần bạn đi trên binary thật mới tính. Số ghi chú `shortcut:` trong mã: 35 (24 ở v0.3.2).

## 5. Đề xuất

Cột "Phí" là ước lượng (S ≤ nửa ngày, M 1–2 ngày, L > 2 ngày), không phải số đo.

### Bậc 1 — bằng chứng mạnh, phí nhỏ, không phụ thuộc quyết định sản phẩm

| id | Đề xuất | Vì sao (số đo) | Bản nhỏ nhất | Phí | Xong khi |
|---|---|---|---|---|---|
| I3 | `routine run --wait-slot[=DUR]` | hết slot thoát 75, phải dựng hàng đợi ngoài dew | chờ slot trống tới DUR thay vì thoát 75; `lock_held` vẫn từ chối ngay | S | test tranh slot: lượt thứ 3 chờ rồi chạy; `lock_held` không đổi hành vi |
| I1 | `dew routine wait NAME [--json] [--timeout]`: bám vào lượt *do nơi khác khởi động* | 135 lần `dew ps` hỏi vòng; có thể đa số lượt qua launcher. `routine run` gọi thẳng thì đã chặn chờ và trả mã thoát (0/1/75) | chặn tới khi báo cáo mới nhất hết RUNNING, in báo cáo cuối, dùng cùng mã thoát với `routine run` | S | E2E: launcher chạy nền, `wait` trả đúng báo cáo; script điều phối không còn gọi `ps` |
| I2 | Sửa nạp tệp ngữ cảnh viết thường | P27, 7 tên chết | khử trùng lặp sau khi biết tệp tồn tại. Đây là lỗi upstream: sửa trong fork thêm chi phí merge; tạm thời có cách né bằng `context_paths` | S | test: thư mục chỉ có `agents.md` → có trong context; cassette VCR không đổi |
| I5 | `routine new NAME --from OLD` | 42/191 định nghĩa là bản thử lại `-rN`, goal ~4k ký tự | chép định nghĩa cũ, ghi đè các cờ được truyền | S | `routine show` của bản mới đúng; không cần sửa JSON tay |
| N1 | Lấy `1f5d7f0b`, `7dadde0b`, `623ba438` từ upstream; `510f6ea3` sau khi kiểm cassette | 28 request stream_stall, 29 RUN_ERROR | cherry-pick từng commit, giải xung đột | M | cổng `scripts/release-gates.sh` xanh (có `TestCoderAgent`); đo lại tỉ lệ stream_stall sau 1 ngày chạy |
| — | Trần mỗi lượt agent: để launcher truyền `--run-timeout` khi `routine new`, hoặc đổi mặc định | 14 TIMEOUT; cờ có sẵn mà 0/191 dùng | không cần mã nếu sửa phía launcher | 0–S | lượt dài không còn TIMEOUT ở đúng 45 phút |

### Bậc 2 — làm khi đã chốt "engine chạy nền"

| id | Đề xuất | Vì sao | Bản nhỏ nhất | Phí | Xong khi |
|---|---|---|---|---|---|
| N2 | **Cổng soát độc lập sau GREEN** (`--review`) | 76/154 GREEN sửa tệp test/verify; luật "người chấm ≠ người làm" của bạn đang làm bằng tay | sau GREEN chạy một agent chỉ đọc (không có tool ghi), model có thể khác (vd. qua `claude_code`/`codex` hoặc model thứ hai), đọc diff + báo cáo, ghi `review_status` và dòng `KET-LUAN:` vào báo cáo | M | lượt có lỗi cố cài sẵn bị chấm KHONG-DAT; lượt sạch DAT; reviewer không sửa được tệp (test) |
| N5 | Tín hiệu sống trong `routine list --json` | P11 | thêm `last_event_age`, `last_event_kind` đọc từ journal của lượt (đã có `attempts`, `changed_count`) | S | lượt treo hiện tuổi sự kiện tăng dần |
| N6 | Subagent tự định nghĩa, đọc cả `~/.claude/agents/*.md` | bạn có 6 agent tự định nghĩa; dew đã đọc `~/.claude/commands` và `skills` | nạp prompt + danh sách tool + model từ frontmatter; gọi qua tool `agent` có tham số tên | M–L | agent chỉ đọc không gọi được tool ghi (test); dùng được cho N2 |
| N3 | `--notify CMD` khi lượt kết thúc | chỉ cần khi lượt do launcher chạy; lệnh gọi thẳng qua `run_in_background` của Claude Code đã được báo khi xong | chạy một lệnh với báo cáo cuối qua stdin | S | lệnh được gọi đúng một lần với JSON báo cáo |
| — | Tài liệu: theo dõi lượt bằng `tail -F runs/<stamp>/*.jsonl` hoặc SSE | stream sự kiện đã có, chỉ chưa ai biết | một đoạn trong `docs/dew-local-setup.md` | 0 | — |

### Bậc 3 — cần quyết định hoặc cần đo thêm

| id | Đề xuất | Điều kiện |
|---|---|---|
| N7 | Checkpoint git bóng + `/rewind` trong TUI | chỉ khi bạn quay lại gõ vào dew hằng ngày |
| N8 | Hook `PostToolUse`, `SessionStart`, `UserPromptSubmit` | như N7; trong Claude Code, bạn tự cài PostToolUse, hai sự kiện kia đến qua plugin |
| N9 | Sandbox bubblewrap/Landlock cho routine (chỉ Linux) | thay 9 `shortcut:` về an toàn; phí L; nên làm trước khi routine chạy trên repo không phải của bạn |
| N10 | Ghi chú bài học theo repo cho routine (memory nhẹ) | đo trước: 29 RUN_ERROR + 16 verify FAIL có lặp cùng kiểu không. Bạn chăm 72 tệp memory trong Claude Code nên giá trị là có thật, nhưng chưa có số cho routine |
| N11 | Hiện hạn mức provider, cảnh báo trước khi chạm trần | cần kiểm gateway có endpoint báo hạn mức không; không có thì bỏ |
| N12 | Model theo session (G-02) | chỉ khi dùng lại TUI |

## 6. Đã cân nhắc và KHÔNG đề xuất

| Ứng viên | Vì sao không |
|---|---|
| `routine schedule --install` | mọi định nghĩa là `goal`, mà goal chỉ chạy tay theo thiết kế; lịch nằm ngoài dew là đúng |
| Cờ `--run-timeout` cho `routine run` | gần như mỗi lượt là một định nghĩa mới (191/218); cờ ở `routine new` đã đủ |
| `--stream-json` cho CLI | SSE của server và journal JSONL đã là stream sự kiện |
| Repo map, giọng nói, ACP, điều khiển từ xa, plugin | không có số đo nào cho thấy cần |
| Workflow fan-out trong dew | Claude Code đã làm phần điều phối (565 lần gọi subagent); dew chỉ cần dễ lái (I1, I3) |
| Đổi model mạnh hơn cho dew | trái quyết định 22/10 (chỉ qwen3.8 chuẩn); tăng tốc bằng cắt việc nhỏ |
| Auto mode bằng model phân loại | guard + sandbox (N9) cho routine rẻ và kiểm được hơn |

## 7. Bốn câu hỏi điểm mù

| Câu hỏi | Trả lời |
|---|---|
| Cái gì vỡ trước? | Gateway: từ 3 lượt song song, TTFT p90 59,6 s (đo ở mục 1). Không phải RAM hay đĩa của máy này. |
| Ca xấu nhất? | 3–4 lượt cùng gửi ngữ cảnh ~95k token: TTFT tối đa 116 s; lượt agent dính trần 45 phút rồi TIMEOUT. |
| Trần cứng? | Throughput prefill của gateway dùng chung; hạn mức của provider (GLM 5 giờ). Không nới được bằng mã dew. |
| Hỏng âm thầm? | (1) GREEN được hiểu là "xong" dù 76/154 lượt có sửa tệp test/verify nên cần soát (tín hiệu rủi ro, không phải 76 lượt sai: thêm test là bình thường với việc `goal`); (2) tệp ngữ cảnh viết thường bị bỏ qua mà không báo; (3) hết slot thì thoát 75, lượt không chạy nếu launcher không đọc mã thoát; (4) prompt trong hàng đợi mất khi server bị SIGKILL. |

## 8. Đã chứng minh / mới nghi / chưa kiểm

**Đã chứng minh, tôi tự chạy lại và agent soát cũng chạy lại độc lập:**
- 218 lượt và lý do dừng; 191 định nghĩa `goal`, 0 có lịch, 0 đặt `run_timeout`/`max_duration`;
- 76/154 GREEN có `touched_tests`; 187/218 lượt chỉ có 1 lượt agent;
- TTFT theo số lượt song song; upstream đi trước 43 commit; 0 lượt GitHub Actions;
- P27 chứng minh bằng đọc mã (thứ tự sắp và khử trùng lặp, giống hệt upstream); cách né `./agents.md` đã kiểm theo
  mã. Chưa chạy test.

**Đã chứng minh, chỉ tôi tự chạy lại (agent soát chưa chạy lại):** lần gõ cuối vào TUI dew (20/10 14:21); 543 lượt gõ
Claude Code trên 11 ngày.

**Agent đo, tôi chưa chạy lại:**
- số tool call/subagent/skill của Claude Code, 217 lệnh `dew` (regex), số tệp memory;
- bảng năng lực mục 2: file:line do agent đọc. Tôi kiểm lại hooks, `Agents json:"-"`, lịch chỉ in, không có hàm khôi
  phục; agent soát kiểm lại từng dòng THIẾU/MỘT PHẦN.

**Mới nghi:** vì sao bạn ngừng gõ vào TUI dew; hỏi vòng `ps` tốn bao nhiêu thời gian của phiên điều phối; N10 có đáng
làm hay không.

**Chưa kiểm:**
- tính năng và phiên bản của công cụ khác ngoài phần agent đã tải trang (một số số phiên bản đến từ trang thứ ba);
- `510f6ea3` có thật làm đỏ cassette hay không;
- TUI chạy thật.

Lệnh tái lập các số chính:

```bash
# lượt routine, lý do dừng, GREEN có sửa test
python3 -I -c 'import json,glob,os,collections as C; g=[json.load(open(f)) for f in glob.glob(os.path.expanduser("~/.local/share/dew/routines/*/runs/*/report.json"))]; print(len(g),C.Counter(d.get("reason") for d in g)); print(sum(bool(d.get("touched_tests")) for d in g if d.get("reason")=="GREEN"))'
# định nghĩa: số lượng, goal, có lịch, có run_timeout
python3 -I -c 'import json,glob,os; d=[json.load(open(f)) for f in glob.glob(os.path.expanduser("~/.config/dew/routines/*.json"))]; print(len(d),sum(x.get("mode")=="goal" for x in d),sum(bool(x.get("schedule")) for x in d),sum(bool(x.get("run_timeout")) for x in d))'
# lần gõ cuối vào TUI dew ở một dự án
python3 -I -c 'import sqlite3,datetime,sys; c=sqlite3.connect("file:"+sys.argv[1]+"?mode=ro&immutable=1",uri=True); print(datetime.datetime.fromtimestamp(c.execute("select max(created_at) from messages where role=\"user\"").fetchone()[0]))' <dự-án>/.crush/crush.db
# upstream và CI của fork
curl -s "https://api.github.com/repos/charmbracelet/crush/compare/$(git rev-parse a73e1bfa)...main" | python3 -c 'import json,sys; print(json.load(sys.stdin)["ahead_by"])'
curl -s https://api.github.com/repos/luannt2002/dew/actions/runs | python3 -c 'import json,sys; print(json.load(sys.stdin)["total_count"])'
```

TTFT theo số lượt song song: với mỗi sự kiện `llm_request` trong `runs/*/*.jsonl`, đếm số báo cáo có
`started ≤ time ≤ finished`, rồi lấy p50/p90 của `ttft_ms` theo nhóm 1 / 2 / ≥3.
