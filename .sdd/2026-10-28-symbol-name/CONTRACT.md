# Contract Traceability Matrix — 標的名稱

Contract: PRD.md
Design map: ARCH.md
Implementation: `internal/`（domain / application / infrastructure）、`cmd/server/dependencies.go`
Oracle: Acceptance Criteria + business rules + NFR（16 clauses）

> 靜態一致性稽核：以 PRD 驗收條件為 oracle，逐條判斷測試斷言與程式路徑，不靠整套測試綠燈下結論。不執行自創情境。

## Clauses

`Test audit` / `Code audit` 為 Claude 的 shadow verdict（先記錄、不因 Jev 改動）。`Jev test` / `Jev code`、`Status`、`Route` 為 `jev_audit.py` 原樣輸出。

| ID | Clause | Spec-expected (oracle) | Impl | Test | Test audit | Code audit | Jev test | Jev code | Status | Route |
|----|--------|------------------------|------|------|------------|------------|----------|----------|--------|-------|
| AC-1 | 有登記標的名稱的追蹤標的 | 2330 的評等結果附上標的名稱「台積電」 | `tracked_symbol_grades_domain.go:28,66` | `session_grade_application_test.go:511` | asserts-oracle | produces-oracle | asserts-oracle 0.93 | produces-oracle 0.43 | ✅ conforms | escalate (low confidence) |
| AC-2 | 沒有登記標的名稱的追蹤標的 | 2317 照常回傳，標的名稱為空 | `tracked_symbol_grades_domain.go:28,66`；`tracked_symbol_grade_dto.go:7` | `session_grade_application_test.go:511`；`tracked_symbol_grade_controller_test.go:68` | asserts-oracle | produces-oracle | asserts-oracle 0.75 | produces-oracle 0.92 | ✅ conforms | recheck |
| AC-3 | 登記的標的名稱不影響自動分析的搜尋字 | 以「台積電」搜尋新聞；查詢時名稱為「台積」 | `session_grade_service.go:88-105` | `session_grade_application_test.go:551` | asserts-oracle | produces-oracle | asserts-oracle 0.97 | produces-oracle 0.63 | ✅ conforms | recheck |
| AC-4 | 以查詢當下登記的標的名稱為準 | 改登記後再查，名稱為「台積電」 | `session_grade_service.go:127-155` | `session_grade_application_test.go:535` | asserts-oracle | produces-oracle | asserts-oracle 1.00 | produces-oracle 0.69 | ✅ conforms | recheck |
| AC-5 | 台股以公司簡稱為標的名稱 | 分析事件與分析結果的名稱都是「台積電」 | `resolved_symbol_vo.go:11`；`analysis_event_domain.go:33`；`analysis_conclusion_domain.go:94` | `symbol_analysis_application_test.go:477`、`:275` | asserts-oracle | produces-oracle | asserts-oracle 0.67 | produces-oracle 0.92 | ✅ conforms | recheck |
| AC-6 | 加密貨幣以幣種名稱為標的名稱 | 名稱為「Bitcoin」 | `resolved_symbol_vo.go:15`；`analysis_event_domain.go:33` | `symbol_analysis_application_test.go:477` | asserts-oracle | produces-oracle | asserts-oracle 0.93 | produces-oracle 0.44 | ✅ conforms | escalate (low confidence) |
| AC-7 | 美股以報價來源提供的公司名稱為標的名稱 | 名稱為「Apple Inc.」，新聞仍以「AAPL」搜尋 | `symbol_resolution_service.go:72-81`；`resolved_symbol_vo.go:20`；`yahoo_finance_proxy.go:70-85` | `symbol_analysis_application_test.go:477`；`yahoo_finance_price_proxy_test.go:89` | asserts-oracle | produces-oracle | asserts-oracle 1.00 | produces-oracle 0.73 | ✅ conforms | recheck |
| AC-8 | 美股報價來源暫時查不到資料 | 分析照常發起並執行，名稱為空 | `symbol_resolution_service.go:77-80` | `symbol_analysis_application_test.go:477` | asserts-oracle | produces-oracle | asserts-oracle 0.64 | produces-oracle 0.73 | ✅ conforms | recheck |
| AC-9 | 美股報價來源查無此代號 | 照常分析（不拒絕），名稱為空 | `symbol_resolution_service.go:77-80` | `symbol_analysis_application_test.go:477` | asserts-oracle | produces-oracle | asserts-oracle 0.93 | produces-oracle 0.99 | ✅ conforms | accept |
| AC-10 | 台股查無此標的仍然拒絕 | 分析被拒絕（標的不存在），不會產生分析事件 | `symbol_resolution_service.go:63`；`symbol_analysis_service.go:39-44` | `symbol_analysis_application_test.go:216` | asserts-oracle | produces-oracle | asserts-oracle 0.96 | produces-oracle 0.48 | ✅ conforms | escalate (low confidence) |
| AC-11 | 重用既有分析時沿用原本的標的名稱 | 回傳既有分析事件與其原名稱，不重新分析 | `symbol_analysis_service.go:45-56` | `symbol_analysis_application_test.go:123` | asserts-oracle | produces-oracle | asserts-oracle 0.96 | produces-oracle 0.80 | ✅ conforms | recheck |
| AC-12 | 分析失敗仍附上標的名稱 | 名稱為「台積電」，沒有分析結果 | `analysis_event_domain.go:78` | `symbol_analysis_application_test.go:437` | asserts-oracle | produces-oracle | asserts-oracle 0.80 | produces-oracle 0.76 | ✅ conforms | recheck |
| AC-13 | 本功能開始使用前發起的分析 | 照常回傳，名稱為空 | `entities/analysis_event.go:10`（`default:''`） | `symbol_analysis_application_test.go:437`；`symbol_name_persistence_test.go:35` | asserts-oracle | produces-oracle | asserts-oracle 0.95 | produces-oracle 0.73 | ✅ conforms | recheck |
| AC-14 | 本功能開始使用後發起的分析 | 名稱為「台積電」 | `analysis_event_domain.go:33` | `symbol_analysis_application_test.go:477` | asserts-oracle | produces-oracle | asserts-oracle 0.97 | produces-oracle 0.81 | ✅ conforms | recheck |
| BR-3 | 標的名稱最長 100 個字；超過只保留前 100 個字，前後空白去除 | 去空白、截到 100 字 | `symbol_name_vo.go:12-18` | `market_and_symbol_vo_test.go:77` | asserts-oracle | produces-oracle | asserts-oracle 0.63 | produces-oracle 0.68 | ✅ conforms | recheck |
| NFR-1 | 查詢美股公司名稱最多等待 5 秒；超過即視為查不到 | 最多等 5 秒，逾時名稱為空、分析照常 | `symbol_resolution_service.go:16,74-80` | `symbol_analysis_application_test.go:533` | asserts-oracle | produces-oracle | asserts-oracle 0.89 | produces-oracle 0.75 | ✅ conforms | recheck |

其餘商業規則（名稱來源、名稱不影響搜尋字、分析結果與分析事件同名、以查詢當下為準）與 AC-3/4/5/7 重疊，不另列。

### Rechecks

- **NFR-1**（第一輪 Jev：`shallow 0.35`，disagree → escalate）：原測試只驗等待上限，沒驗「逾時 → 名稱為空、分析照常」。已補強為：上限介於 4–5 秒，且逾時後分析事件照常建立、名稱為空；並以兩個突變（拿掉上限、逾時改為回錯）確認測試會失敗。第二輪 Jev：conforms。
- **AC-10**：原本靠 strict mock「沒有設定 Create」間接保證不產生分析事件；已補上明確的 `AssertNotCalled(Create)`。
- **AC-1 / AC-6 / AC-10**（Jev code 信心低）：重讀程式路徑——AC-1 `ToDtos` 由追蹤標的取名稱寫入 DTO；AC-6 `NewCryptoResolvedSymbolVo` 以幣種名稱建 `SymbolNameVo` 再存入分析事件；AC-10 `ResolveSymbol` 回 `ErrSymbolNotFound` 後 `StartSymbolAnalysis` 在建立分析事件前即返回。三者路徑單純，且已有突變測試驗證（拿掉對應那一行測試即失敗）。

## Orphans (code with no clause)

| Code | Description | Verdict |
|------|-------------|---------|
| `symbol_resolution_service.go:72-81`（經 `NewsSearchService`） | 搜尋美股新聞時也會查一次公司名稱（最多 5 秒），但結果不被使用 | undocumented — ARCH §8 已記為已知取捨，非 Out of Scope 違反 |

## Summary

- Conforms: 16/16 clauses ✅ (100%)
- Violations: —
- Mis-asserted: —（NFR-1 已修正）
- Partial: —
- Gaps: —
- Unclear: —
- Orphans: 1（已知取捨）
- Jev: ran on 16 clauses（NFR-1、AC-10 修正後重跑）
- Needs James: AC-1、AC-6、AC-10（Jev 程式碼判讀信心低；Claude 重讀並以突變測試佐證，判定 conforms）
