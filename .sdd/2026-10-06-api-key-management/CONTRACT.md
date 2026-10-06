# Contract Traceability Matrix — api-key-management

Contract: PRD.md（v1.0, Finalized）
Design map: ARCH.md
Implementation: `internal/`（domain / application / controller / infrastructure/persistence）、`cmd/server/`
Oracle: Acceptance Criteria（34 clauses：AC-1..24、BR-1..6、NFR-1..4）
Branch: `feature/api-key-management`（HEAD `ec177d4`）

> 本文件為**靜態符合性稽核**：以 PRD 推導的 oracle 為準，分別判斷「測試是否斷言 oracle」與「程式碼是否產生 oracle」，不以測試綠燈作為判定依據，也不執行自行發明的情境。Oracle 皆在閱讀任何程式碼 / 測試之前由 PRD 文字推導。已另以 `TEST_POSTGRES_DSN` 跑過既有測試作佐證（全綠），但判定不依賴該結果。

## 名詞橋接（Phase 3 step 0，依 UL-MAP + ARCH）

| 業務結果 | 具體形式 |
|---|---|
| 狀態「停用中」/「已啟用」 | `status: "inactive"` / `"active"`；持久化 `IsActive=false/true` |
| 「API key 名稱為必填」 | 400 `api_key_name_required` |
| 「API key 名稱不可超過 100 個字」 | 400 `api_key_name_too_long` |
| 「需要提供 API key」 | 401 `api_key_missing`（header `X-API-Key` 缺 / 空白） |
| 「API key 無效」（不存在 / 已撤銷） | 401 `api_key_invalid` |
| 「API key 尚未啟用」 | 403 `api_key_inactive` |
| 「服務暫時無法使用」 | 503 `service_unavailable` |
| 受保護功能 | `ApiKeyController.RequireActiveApiKey()` 關卡（本切片尚無實際受保護路由，測試以 `/protected` 測試路由驗證） |

## Clauses

| ID | Clause | Spec-expected (oracle) | Impl | Test | Test audit | Code audit | Status |
|----|--------|------------------------|------|------|------------|------------|--------|
| AC-1 | US-01 以有效名稱申請成功：Given 名稱「我的研究腳本」When 申請 Then 申請成功，取得完整 API key；And 狀態為「停用中」 | 申請成功、回傳完整 API key、該 key 狀態為停用中 | `internal/domain/service/api_key_service.go:21-32`；`internal/domain/models/domains/api_key_domain.go:30-32`；`internal/controller/api_key_controller.go:42-54` | `internal/controller/tests/api_key_controller_test.go:69` `TestIssueApiKey_ReturnsTheFullKeyOnceAsInactive`；`internal/application/tests/api_key_application_test.go:35`（持久化 `IsActive=false`） | asserts-oracle | produces-oracle | ✅ conforms |
| AC-2 | US-01 名稱前後的空白會被去除：Given 名稱「  研究  」When 申請 Then 申請成功，名稱記為「研究」 | 申請成功，被保存 / 回傳的名稱為去除前後空白後的「研究」 | `internal/domain/models/vo/api_key_name_vo.go:21` | `internal/domain/models/vo/tests/api_key_name_vo_test.go:19`；`internal/application/tests/api_key_application_test.go:44-51`（斷言保存的 entity 名稱已去空白） | asserts-oracle | produces-oracle | ✅ conforms |
| AC-3 | US-01 名稱剛好 100 個字可以申請 | 申請成功 | `api_key_name_vo.go:25`（以字元數 `utf8.RuneCountInString` 判斷）；`internal/domain/models/entities/api_key.go:7`（`size:100`，PostgreSQL varchar 以字元計） | `api_key_name_vo_test.go:20` | asserts-oracle | produces-oracle | ✅ conforms |
| AC-4 | US-01 名稱超過 100 個字被拒：101 字 Then 申請被拒，告知「API key 名稱不可超過 100 個字」 | 申請被拒，提示「API key 名稱不可超過 100 個字」，不產生 API key | `api_key_name_vo.go:25-27`；`api_key_controller.go:24` | `api_key_controller_test.go:92`；`api_key_application_test.go:74`；`api_key_name_vo_test.go:21` | asserts-oracle | produces-oracle | ✅ conforms |
| AC-5 | US-01 未提供名稱被拒：Then 申請被拒，告知「API key 名稱為必填」 | 申請被拒，提示「API key 名稱為必填」 | `api_key_name_vo.go:22-24`；`api_key_controller.go:23` | `api_key_controller_test.go:93`（body `{}`）；`api_key_application_test.go:73` | asserts-oracle | produces-oracle | ✅ conforms（另見 Orphan O-1：完全不帶 body 時回的是「請求內容格式錯誤」） |
| AC-6 | US-01 名稱只有空白被拒：Then 告知「API key 名稱為必填」 | 申請被拒，提示「API key 名稱為必填」 | `api_key_name_vo.go:21-24` | `api_key_controller_test.go:94`；`api_key_name_vo_test.go:23` | asserts-oracle | produces-oracle | ✅ conforms |
| AC-7 | US-01 名稱可與其他 API key 重複：Given 已有名稱「研究」When 以「研究」申請 Then 申請成功，取得不同的新 API key | 申請成功，新 key 與既有 key 不同 | `entities/api_key.go:7`（`Name` 無唯一限制）；`api_key_service.go:26`（每次產生新 secret） | `api_key_application_test.go:54` `TestIssueApiKey_SameNameTwiceYieldsDifferentKeys` | asserts-oracle | produces-oracle | ✅ conforms |
| AC-8 | US-02 申請當下看得到完整 API key | 申請結果含完整 API key | `domains/api_key_domain.go:59-66`；`dto/issued_api_key_dto.go:6` | `api_key_controller_test.go:80` | asserts-oracle | produces-oracle | ✅ conforms |
| AC-9 | US-02 之後查詢看不到完整 API key：Then 結果只含名稱與狀態，不含完整 API key | 查詢結果只有名稱與狀態，無完整 key | `dto/api_key_status_dto.go:3-6`；`api_key_domain.go:52-57` | `api_key_controller_test.go:115-134`（`JSONEq` 精確比對 `{"name","status"}`） | asserts-oracle | produces-oracle | ✅ conforms |
| AC-10 | US-03 尚未啟用的 API key 顯示停用中 | 狀態為停用中 | `api_key_domain.go:52-57,80-85` | `api_key_controller_test.go:121`；`domains/tests/api_key_domain_test.go:66` | asserts-oracle | produces-oracle | ✅ conforms |
| AC-11 | US-03 已啟用的 API key 顯示已啟用 | 狀態為已啟用 | `api_key_domain.go:80-83` | `api_key_controller_test.go:122`；`api_key_domain_test.go:67` | asserts-oracle | produces-oracle | ✅ conforms |
| AC-12 | US-03 已撤銷的 API key 無法查詢：Then 查詢被拒，告知「API key 無效」 | 查詢被拒，提示「API key 無效」 | `api_key_domain.go:53-55` | `api_key_controller_test.go:178`（GET `/api-keys/me`）；`api_key_application_test.go:104` | asserts-oracle | produces-oracle | ✅ conforms |
| AC-13 | US-03 不存在的 API key 無法查詢：告知「API key 無效」 | 查詢被拒，提示「API key 無效」 | `api_key_service.go:77-79` | `api_key_controller_test.go:177` | asserts-oracle | produces-oracle | ✅ conforms |
| AC-14 | US-03 未提供 API key 無法查詢：告知「需要提供 API key」 | 查詢被拒，提示「需要提供 API key」 | `vo/api_key_secret_vo.go:33-36` | `api_key_controller_test.go:176` | asserts-oracle | produces-oracle | ✅ conforms |
| AC-15 | US-04 撤銷已啟用的 API key：Then 撤銷成功；And 之後以它使用受保護功能被拒「API key 無效」 | 撤銷成功；之後使用受保護功能被拒並提示「API key 無效」 | `api_key_service.go:42-55`；`api_key_domain.go:44-50`；`persistence/api_key_repository.go:35-37`；`api_key_domain.go:34-37` | `api_key_controller_test.go:137`（204）；`persistence/tests/api_key_repository_test.go:46`（revoked_at 落地）；`api_key_controller_test.go:178`（已撤銷 → `/protected` 401 無效） | asserts-oracle（由三段測試組合釘住；無單一「撤銷後再使用」串接測試） | produces-oracle | ✅ conforms |
| AC-16 | US-04 尚未啟用也能撤銷 | 撤銷成功 | `api_key_domain.go:44-50` | `api_key_application_test.go:129-139`（`isActive=false`）；`api_key_domain_test.go:44` | asserts-oracle | produces-oracle | ✅ conforms |
| AC-17 | US-04 已撤銷的 API key 不能再撤銷：告知「API key 無效」 | 撤銷被拒，提示「API key 無效」 | `api_key_domain.go:45-47` | `api_key_controller_test.go:178`（DELETE）；`api_key_application_test.go:150`；`api_key_domain_test.go:45` | asserts-oracle | produces-oracle | ✅ conforms（併發雙撤銷見備註 N-2） |
| AC-18 | US-04 撤銷後即使 administrator 重新啟用仍無法使用：告知「API key 無效」 | 使用受保護功能被拒，提示「API key 無效」 | `api_key_domain.go:34-37` | `api_key_controller_test.go:178`（`IsActive:true, RevokedAt` → `/protected`）；`api_key_domain_test.go:24` | asserts-oracle | produces-oracle | ✅ conforms |
| AC-19 | US-04 不存在的 API key 不能撤銷：告知「API key 無效」 | 撤銷被拒，提示「API key 無效」 | `api_key_service.go:77-79` | `api_key_controller_test.go:177`（DELETE） | asserts-oracle | produces-oracle | ✅ conforms |
| AC-20 | US-05 已啟用且未撤銷的 API key 放行：Then 受保護功能照常執行 | 受保護功能被執行 | `api_key_controller.go:73-84`；`api_key_domain.go:34-42` | `api_key_controller_test.go:147` | asserts-oracle | produces-oracle | ✅ conforms |
| AC-21 | US-05 停用中的 API key 被拒（從未啟用，或被 administrator 停用）：告知「API key 尚未啟用」 | 被拒，提示「API key 尚未啟用」 | `api_key_domain.go:38-40`；`api_key_controller.go:27` | `api_key_controller_test.go:198`；`api_key_domain_test.go:23` | asserts-oracle | produces-oracle | ✅ conforms |
| AC-22 | US-05 已撤銷的 API key 被拒：告知「API key 無效」 | 被拒，提示「API key 無效」 | `api_key_domain.go:35-37` | `api_key_controller_test.go:178`；`api_key_domain_test.go:24-25` | asserts-oracle | produces-oracle | ✅ conforms |
| AC-23 | US-05 不存在的 API key 被拒：告知「API key 無效」 | 被拒，提示「API key 無效」 | `api_key_service.go:77-79` | `api_key_controller_test.go:177`（`/protected`） | asserts-oracle | produces-oracle | ✅ conforms |
| AC-24 | US-05 未提供 API key 被拒：告知「需要提供 API key」 | 被拒，提示「需要提供 API key」 | `api_key_secret_vo.go:33-36` | `api_key_controller_test.go:176`（`/protected`） | asserts-oracle | produces-oracle | ✅ conforms |
| BR-1 | API key 狀態：停用中 ⇄ 已啟用（由 administrator 切換）；任一狀態皆可 → 已撤銷（由持有者觸發，不可逆） | 新 key 為停用中；啟用旗標由 administrator 切換；停用中 / 已啟用皆可撤銷；撤銷後無任何途徑回到有效 | `api_key_domain.go:30-32,44-50`；`api_key_repository.go:35-37`（只寫 revoked_at，不碰 is_active；無任何清除 revoked_at 的路徑） | `api_key_domain_test.go:36-57,80-88`；`api_key_repository_test.go:46` | asserts-oracle | produces-oracle | ✅ conforms |
| BR-2 | 已撤銷優先於啟用狀態：已撤銷的 API key 無論是否啟用，一律視為無效 | 已撤銷 → 無效（查詢、撤銷、受保護功能皆然），與啟用旗標無關 | `api_key_domain.go:35,45,53`（三處皆先判 `isRevoked()`） | `api_key_domain_test.go:24-25,45,68`；`api_key_controller_test.go:178` | asserts-oracle | produces-oracle | ✅ conforms |
| BR-3 | API key 名稱：必填，去除前後空白後長度 1–100 字，可重複 | 去空白後空 → 必填錯誤；>100 字 → 過長錯誤；1–100 字通過；重名允許 | `api_key_name_vo.go:20-29`；`entities/api_key.go:7` | `api_key_name_vo_test.go:11-33`；`api_key_application_test.go:54` | asserts-oracle | produces-oracle | ✅ conforms |
| BR-4 | 「無效」涵蓋「不存在」與「已撤銷」兩種情況，對外不區分 | 不存在與已撤銷對外回應完全相同（同訊息、同結果） | `api_key_service.go:78` 與 `api_key_domain.go:36,46,54` 回同一 `ErrApiKeyInvalid` → `api_key_controller.go:26` | `api_key_controller_test.go:177-178`（三條路由，兩情境斷言相同 status / code / message） | asserts-oracle | produces-oracle | ✅ conforms |
| BR-5 | Edge：申請過程中資料無法保存 → 申請失敗，使用者不會取得任何 API key | 申請失敗，回應中沒有任何 API key | `api_key_service.go:28-30`；`api_key_controller.go:93` | `api_key_application_test.go:75-87`（錯誤 + `ApiKey` 為空）；`api_key_controller_test.go:96-98`（503 + 訊息） | asserts-oracle | produces-oracle | ✅ conforms |
| BR-6 | Edge：驗證 API key 時資料無法讀取 → 拒絕本次請求並告知服務暫時無法使用，不可放行 | 請求被拒、提示「服務暫時無法使用」、受保護功能未被執行 | `api_key_service.go:73-76`；`api_key_controller.go:76-79,93` | `api_key_controller_test.go:179`（`/protected` → 503 + 訊息，handler 未執行）；`api_key_repository_test.go:63` | asserts-oracle | produces-oracle | ✅ conforms |
| NFR-1 | Performance：單次驗證不應明顯增加回應時間 | 每次驗證只做一次輕量雜湊 + 一次索引查找，不引入昂貴運算 | `api_key_secret_vo.go:37`（SHA-256）；`entities/api_key.go:8`（`uniqueIndex`）；`api_key_repository.go:25` | 無效能測試；`api_key_repository_test.go:37` 只間接證明唯一索引存在 | no-test | produces-oracle | 🟡 partial |
| NFR-2 | Security：系統不保存可還原的完整 API key | 只保存不可逆衍生值，持久化資料中不含完整 key | `entities/api_key.go:5-12`（只有 `SecretHash`）；`api_key_domain.go:31` | `api_key_application_test.go:51`（斷言保存的 entity 只含雜湊、不含明文） | asserts-oracle | produces-oracle | ✅ conforms |
| NFR-3 | Security：完整 API key 需具備足夠的隨機性，無法被猜測或列舉 | 以密碼學安全亂數產生、具足夠熵（不可猜測、不可列舉） | `api_key_secret_vo.go:24-30`（`crypto/rand` 32 bytes） | `api_key_secret_vo_test.go:11-20`（只斷言長度 43 字元 + 兩次不同） | shallow | produces-oracle | 🟠 mis-asserted |
| NFR-4 | Security：完整 API key 帶有可辨識本服務的固定開頭 | 每把完整 key 以本服務固定前綴開頭 | `api_key_secret_vo.go:13,28`（`snh_`） | `api_key_secret_vo_test.go:15`；`api_key_controller_test.go:80` | asserts-oracle | produces-oracle | ✅ conforms |

## Orphans (code with no clause)

Out of Scope 負面清單（administrator 啟用介面、帳號 / 登入 / API key 清單、頻率限制 / 用量計次 / 到期、找回遺失 key）逐項檢查：**均未實作，無 scope creep。**

| Code | Description | Verdict |
|------|-------------|---------|
| O-1 `internal/controller/api_key_controller.go:44-46` | 申請請求 body 無法解析（JSON 格式錯誤，**或完全不帶 body**、`name` 非字串）時回 400 `invalid_request_body`「請求內容格式錯誤」。PRD 無此訊息；且「完全不帶 body」在業務上同樣屬「使用者沒有提供 API key 名稱」（AC-5），卻得到不同於「API key 名稱為必填」的提示 | undocumented — 需決定：補 PRD 條款，或讓空 body 落入「名稱為必填」 |
| O-2 `internal/domain/models/dto/issued_api_key_dto.go:4`；`api_key_domain.go:61` | 申請結果額外回傳內部流水號 `id`（ARCH 設計，PRD 未要求）。流水號會讓外部推得已發出的 key 數量 | undocumented |
| O-3 `internal/domain/models/vo/api_key_secret_vo.go:33` | 出示的 API key 會先去除前後空白再比對；只含空白的 header 視為「未提供」 | undocumented（行為合理，未入 PRD） |
| O-4 `internal/controller/api_key_controller.go:81`；`AuthorizedApiKeyIDContextKey` | 關卡放行後把 API key ID 寫入 request context，供後續切片使用 | undocumented（ARCH 預留） |
| O-5 `internal/controller/health_controller.go:15-17`；`cmd/server/dependencies.go:39` | `GET /health` 健康檢查 | undocumented（ARCH 定為服務骨架，非 PRD 範圍） |

## 備註（非條款判定，但值得實作者留意）

- **N-1 受保護功能尚未存在：** `cmd/server/dependencies.go:38-43` 未將 `RequireActiveApiKey()` 掛到任何路由（ARCH §2 明訂「受保護的業務路由 Not touched」）。AC-15/18/20–24 僅以測試專用 `/protected` 路由（`api_key_controller_test.go:41`）驗證關卡本身；真正掛上時需在後續切片補整合測試。
- **N-2 併發雙撤銷：** `api_key_repository.go:36` 的更新沒有「`revoked_at` 仍為空」條件，兩個同時的撤銷請求都會讀到未撤銷而都回成功（第二次應為「API key 無效」），且第二次會覆寫撤銷時間。循序情境（AC-17）符合；不可逆性（BR-1）不受影響。
- **N-3 GORM struct 條件忽略零值：** `api_key_repository.go:25` 以 `Where(&entities.ApiKey{SecretHash: secretHash})` 查詢，若 `secretHash` 為空字串，條件會被 GORM 丟棄而回傳第一筆資料。目前呼叫端恆傳 64 字元 SHA-256 hex、且空 key 已在 `api_key_secret_vo.go:34` 擋下，故不可達；屬潛在風險。

## Summary

- Conforms: 32/34 clauses ✅ (94%)
- Violations: —
- Mis-asserted: NFR-3
- Partial: NFR-1
- Gaps: —
- Unclear: —
- Orphans: 5（O-1 ~ O-5；皆為 undocumented，無 out-of-scope violation）

---

## 稽核後修正紀錄（2026-10-06）

| 項目 | 處理 |
| :--- | :--- |
| NFR-3 斷言過弱 | 補斷言：產生的 API key 隨機段解碼後為 32 bytes（256 bits） |
| NFR-1 無測試 | 不補效能測試；ARCH 記錄每次驗證只做一次雜湊與一次唯一索引查找 |
| O-1 空 body 回「請求內容格式錯誤」 | 修正：空 body 視同未提供名稱，回「API key 名稱為必填」 |
| O-2 申請結果外露流水號 | 修正：申請結果不再回傳流水號 |
| O-3 出示 API key 先去空白 | 保留，記入 ARCH |
| O-4 關卡放行後寫入 API key 識別 | 保留，供後續切片使用 |
| O-5 健康檢查 | 保留，服務骨架 |
| N-2 同時撤銷皆回成功 | 修正：僅在尚未撤銷時寫入，競爭失敗者回「API key 無效」 |
| N-3 空雜湊查詢條件被忽略 | 修正：改用明確欄位條件，空雜湊不會比對到任何資料 |
