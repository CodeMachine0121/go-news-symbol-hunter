# Contract Traceability Matrix — 2026-10-07-tracked-symbol-session-grade（追蹤標的時段評等）

Contract: PRD.md（v1.0, Finalized）
Design map: ARCH.md
Implementation: `internal/domain/service/session_grade_service.go`、`internal/domain/models/domains/{trading_day,combined_grade,session_run,analyst_consultation,tracked_symbol_grade}_domain.go`、`internal/domain/models/vo/session_weights_vo.go`、`internal/infrastructure/persistence/{tracked_symbol,session_grade,combined_grade,session_run}_repository.go`、`internal/controller/tracked_symbol_grade_controller.go`、`internal/job/`、`cmd/server/`（branch `feature/tracked-symbol-session-rating`）
Oracle: Acceptance Criteria（56 clauses：AC × 37、BR × 14、NFR × 5）

> 本文件為**靜態符合度稽核**：以 PRD 推導的 oracle 對照測試斷言與程式路徑，不執行自創情境。判定不依賴 pass/fail。資料庫層測試（`persistence/tests`、`cmd/server` 的建表測試）需 `TEST_POSTGRES_DSN`，未設定時 skip。

縮寫：`SGA` = `internal/application/tests/session_grade_application_test.go`；`TDD` = `internal/domain/models/domains/tests/trading_day_domain_test.go`；`CGD` = `internal/domain/models/domains/tests/combined_grade_domain_test.go`；`REPO` = `internal/infrastructure/persistence/tests/session_grade_repository_test.go`；`CTL` = `internal/controller/tests/tracked_symbol_grade_controller_test.go`；`SGS` = `internal/domain/service/session_grade_service.go`。

橋接說明（Phase 3）：
- 「時段評等」↔ `entities.SessionGrade`；「綜合評等」↔ `entities.CombinedGrade`；「時段執行紀錄」↔ `entities.SessionRun`，「已完成 / 失敗 / 執行中」↔ `succeeded` / `failed` / `running`。
- 盤前 / 盤中 / 盤後 ↔ `preMarket` / `intraday` / `afterMarket`；交易日 ↔ `2006-01-02`（台北日期）。
- 「標的與市場類別需一起提供」↔ `service.ErrSymbolAndCategoryRequiredTogether`（400 `symbol_and_category_required_together`）；「市場類別只能是…」↔ `ErrMarketCategoryUnsupported`；「API key 尚未啟用」↔ 403 `api_key_inactive`。
- 「同時段只執行一次」↔ `session_runs (trading_day, session)` 唯一索引 → `ErrSessionRunAlreadyExists`。

## Clauses（初次稽核）

| ID | Clause | Spec-expected (oracle) | Impl | Test | Test audit | Code audit | Status |
|----|--------|------------------------|------|------|------------|------------|--------|
| AC-1 | 追蹤中的台股標的產生時段評等 | 2330 有一筆當天（週三）盤前時段評等 | `SGS:64-69`、`SGS:86-114` | `SGA:138` | asserts-oracle | produces-oracle | ✅ |
| AC-2 | 暫停的標的不分析 | 2317 沒有任何時段評等 | `tracked_symbol_repository.go:23` | `REPO:18` | asserts-oracle | produces-oracle | ✅ |
| AC-3 | 非台股的追蹤標的不分析 | AAPL 沒有任何時段評等 | `SGS:64`（只取 twStock）、repository category 篩選 | `SGA:138`（只接受 `FindTracking(…, "twStock")`）、`REPO:18` | asserts-oracle | produces-oracle | ✅ |
| AC-4 | 沒有可分析的標的仍完成執行 | 執行紀錄已完成，成功 0、失敗 0 | `SGS:72-73` | `SGA:248` | asserts-oracle | produces-oracle | ✅ |
| AC-5 | 同一標的同一市場類別只能登記一次 | 第二次登記不成立，清單仍一筆 | `entities/tracked_symbol.go` 唯一索引 | `REPO:34` | asserts-oracle | produces-oracle | ✅ |
| AC-6 | 週三盤前執行 | 執行週三盤前，新聞從週二 13:30 起 | `trading_day_domain.go` | `TDD:17` | asserts-oracle | produces-oracle | ✅ |
| AC-7 | 週一盤前新聞從上週五收盤起 | 新聞從上週五 13:30 起 | 同上 | `TDD:17` | asserts-oracle | produces-oracle | ✅ |
| AC-8 | 週三盤中執行 | 執行盤中，新聞從週三 09:00 起 | 同上 | `TDD:17` | asserts-oracle | produces-oracle | ✅ |
| AC-9 | 週三盤後執行 | 執行盤後，新聞從週三 13:30 起 | 同上 | `TDD:17` | asserts-oracle | produces-oracle | ✅ |
| AC-10 | 執行時段結束的那一刻不執行（09:00 整） | 不執行任何時段 | 同上 | `TDD:17`、`SGA:217` | asserts-oracle | produces-oracle | ✅ |
| AC-11 | 盤後最後一分鐘仍執行（23:59） | 執行週三盤後 | 同上 | `TDD:17` | asserts-oracle | produces-oracle | ✅ |
| AC-12 | 週末不執行 | 週六 08:30 不執行 | 同上 | `TDD:17`、`SGA:217` | asserts-oracle | produces-oracle | ✅ |
| AC-13 | 同時段只執行一次 | 第二次檢查不再執行 | `SGS:45-48`、唯一索引 | `SGA:234`、`REPO:118`、`service/tests` | asserts-oracle | produces-oracle | ✅ |
| AC-14 | 錯過的時段不補跑（停機 12:00–14:00，14:00 恢復） | 週三盤中沒有執行紀錄與時段評等 | `trading_day_domain.go` DueSession 只看當下 | `TDD:17` 只測 13:30 整，**沒有 14:00 的情境** | no-test | produces-oracle | 🟡 partial |
| AC-15 | 範圍外的新聞不採用 | 盤中時段評等佐證只有新聞 A | `news_collection_domain.go` Curate、`SGS:94` | `SGA:172` | asserts-oracle | produces-oracle | ✅ |
| AC-16 | 剛好在範圍起點發布的新聞採用 | 09:00 整的新聞列入佐證 | 同上 | `SGA:172`（新聞 C） | asserts-oracle | produces-oracle | ✅ |
| AC-17 | 範圍內沒有新聞時為中性 | 盤中時段評等中性、信心 0 | `analysis_conclusion_domain.go` | `SGA:199` | asserts-oracle | produces-oracle | ✅ |
| AC-18 | 自動分析不重用使用者發起的分析 | 2330 重新分析並產生時段評等；使用者的分析不變 | `SessionGradeService` 不依賴分析事件 / 分析結果（結構上不可能重用或修改） | `SGA:138`（每次都呼叫 AI 並產生時段評等；fixture 無分析事件 repository） | asserts-oracle | produces-oracle | ✅ |
| AC-19 | 只有盤前時綜合等於盤前 | 看多、信心 60 | `combined_grade_domain.go` | `CGD:15` | asserts-oracle | produces-oracle | ✅ |
| AC-20 | 盤前與盤中加權平均 | +0.2、中性、信心 52 | 同上 | `CGD:15`、`SGA:172` | asserts-oracle | produces-oracle | ✅ |
| AC-21 | 三個時段加權平均 | +0.3、中性 | 同上 | `CGD:15` | asserts-oracle | produces-oracle | ✅ |
| AC-22 | 缺少的時段不列入計算 | +1、看多 | 同上 | `CGD:15` | asserts-oracle | produces-oracle | ✅ |
| AC-23 | 綜合分數換回評等的邊界（+1.5/+0.5/+0.4/−0.4/−0.5/−1.5） | 強烈看多 / 看多 / 中性 / 中性 / 看空 / 強烈看空 | 同上 | `CGD:40` | asserts-oracle | produces-oracle | ✅ |
| AC-24 | 權重設定不是正數時使用預設值 | 盤後權重 0 → 0.5 | `session_weights_vo.go`、`cmd/server/config.go` | `session_weights_vo_test.go:10`、`cmd/server/config_test.go:13` | asserts-oracle | produces-oracle | ✅ |
| AC-25 | 新時段完成時覆蓋當天的綜合評等 | 當天仍只有一筆綜合評等，為三段重算結果 | `SGS:108-113`、`combined_grade_repository.go` upsert | `SGA:172`（以既有盤前重算）、`REPO:80`（同日覆蓋） | asserts-oracle | produces-oracle | ✅ |
| AC-26 | 盤前開始時刪除前一天的結果 | 週二資料全刪，之後寫入週三 | `SGS:55-61` | `SGA:138`、`REPO:63`、`REPO:104` | asserts-oracle | produces-oracle | ✅ |
| AC-27 | 週末仍查得到週五的結果 | 週六查詢得到週五的綜合與時段評等 | 查詢不篩交易日（`SGS:116-147`），週末無任何執行 | 查詢測試只在週三（`SGA:393`），**沒有週六查週五的情境** | no-test | produces-oracle | 🟡 partial |
| AC-28 | 週一盤前刪除週五的結果 | 週五資料全刪 | `SGS:55-61` | 刪除只在週三測（`SGA:138`），**沒有週一的情境** | no-test | produces-oracle | 🟡 partial |
| AC-29 | 只有盤前會刪除前一天的結果 | 盤中執行時週二資料保留，新增週三盤中 | `SGS:55` | `SGA:172`（strict mock：呼叫刪除即失敗） | asserts-oracle | produces-oracle | ✅ |
| AC-30 | 一檔失敗其他照常 | 2330 無評等、2317 有；已完成，成功 1 失敗 1 | `SGS:66-69` | `SGA:261` | asserts-oracle | produces-oracle | ✅ |
| AC-31 | 服務重新啟動中斷執行 | 改為失敗、原因「服務重新啟動，執行中斷」；當天盤前不再重跑 | `SGS:149-154`、唯一索引 | `SGA:379`、`REPO:134`、`REPO:118` | asserts-oracle | produces-oracle | ✅ |
| AC-32 | 查詢全部追蹤標的評等 | 回 2330：交易日、綜合評等、綜合信心、盤前與盤中時段評等（評等、信心、理由、關鍵事件、風險因子） | `SGS:116-147`、`tracked_symbol_grade_domain.go` | `SGA:393`、`CTL:55` | asserts-oracle | produces-oracle | ✅ |
| AC-33 | 指定標的沒有評等時回傳空結果 | 查詢成功、結果為空 | 同上 | `SGA:413`、`CTL:69` | asserts-oracle | produces-oracle | ✅ |
| AC-34 | 指定標的只回傳該標的 | 只有 2330 | `combined_grade_repository.go` FindAll 篩選 | `REPO:80`、`CTL:55` | asserts-oracle | produces-oracle | ✅ |
| AC-35 | 只提供標的沒有市場類別 | 被拒，「標的與市場類別需一起提供」 | `SGS:119-122` | `SGA:423`、`CTL:80` | asserts-oracle | produces-oracle | ✅ |
| AC-36 | 不支援的市場類別 | 被拒，「市場類別只能是 crypto、twStock、usStock」 | `SGS:123-126` | `SGA:423`、`CTL:80` | asserts-oracle | produces-oracle | ✅ |
| AC-37 | 停用中的 API key 不能查詢 | 被拒，「API key 尚未啟用」 | `RequireActiveApiKey` | `CTL:80`、`cmd/server/dependencies_test.go`（路由受保護） | asserts-oracle | produces-oracle | ✅ |
| BR-1 | 執行時段起點含、終點不含 | 08:00/12:30/20:00 執行；09:00/13:30 不執行 | `trading_day_domain.go` | `TDD:17` | asserts-oracle | produces-oracle | ✅ |
| BR-2 | 新聞範圍起點含，另受 7 天 / 30 則限制 | 範圍外剔除；超過 7 天仍剔除 | `news_collection_domain.go` | `news_collection_domain_test.go`（新增案例） | asserts-oracle | produces-oracle | ✅ |
| BR-3 | 前一個交易日 = 往前最近的週一到週五 | 週一 → 上週五 | `trading_day_domain.go` | `TDD:17` | asserts-oracle | produces-oracle | ✅ |
| BR-4 | 交易日 = 台北日期 | UTC 23:00 → 隔天日期 | 同上 | `TDD:52` | asserts-oracle | produces-oracle | ✅ |
| BR-5 | 正規化規則與使用者發起分析相同 | 非法評等中性、信心 clamp、關鍵事件只採用讀過的新聞 | `AnalysisConclusionDomain`（共用） | `analysis_domain_test.go`（既有）、`SGA:138` | asserts-oracle | produces-oracle | ✅ |
| BR-6 | 綜合分數 / 綜合信心公式（四捨五入） | 加權平均；信心 52.6 → 53 | `combined_grade_domain.go` | `CGD:15` | asserts-oracle | produces-oracle | ✅ |
| BR-7 | 綜合分數換回評等 | 門檻 1.5/0.5/−0.5/−1.5 | 同上 | `CGD:40` | asserts-oracle | produces-oracle | ✅ |
| BR-8 | 權重可設定，非正數用預設 | 0.3/0.2/0.5 | `session_weights_vo.go`、`config.go` | `session_weights_vo_test.go`、`config_test.go` | asserts-oracle | produces-oracle | ✅ |
| BR-9 | 一個標的沿用 10 分鐘、最多 5 輪 AI 工具使用上限 | 每檔分析最多 10 分鐘 | `SGS:87`（`AnalysisTimeout`）、`AnalystConsultationService`（5 輪，既有測試） | **沒有測試確認時段分析帶 10 分鐘上限** | no-test | produces-oracle | 🟡 partial |
| BR-10 | 不建立分析事件、不寫分析結果、不受重用與同時上限限制 | 同 AC-18 | 結構保證 | `SGA:138` | asserts-oracle | produces-oracle | ✅ |
| BR-11 | 執行紀錄狀態轉換、永久保留 | 執行中 → 已完成 / 失敗；無刪除 | `session_run_domain.go`；無刪除程式 | `SGA:138`、`SGA:331` | asserts-oracle | produces-oracle | ✅ |
| BR-12 | 單一標的失敗（AI、標的查無、保存失敗）→ 計為失敗，繼續下一檔 | 失敗 +1，其他照常 | `SGS:86-114` | `SGA:261`、`SGA:285` | asserts-oracle | produces-oracle | ✅ |
| BR-13 | 建立執行紀錄失敗 → 本次不執行 | 不分析 | `SGS:49-51` | `SGA:241`、`service/tests` | asserts-oracle | produces-oracle | ✅ |
| BR-14 | 刪除前一天的結果失敗 → 執行紀錄失敗、本時段不分析 | 失敗且不讀追蹤標的 | `SGS:55-62` | `SGA:331` | asserts-oracle | produces-oracle | ✅ |
| NFR-1 | 依序分析（一次一檔） | 不會同時分析兩檔 | `SGS:66-68` 單一 goroutine 迴圈 | **沒有測試確認不重疊** | no-test | produces-oracle | 🟡 partial |
| NFR-2 | 查詢需已啟用 API key | 同 AC-37 | 路由群組 | `CTL:80`、`dependencies_test.go` | asserts-oracle | produces-oracle | ✅ |
| NFR-3 | 時間以台北時間判斷 | UTC 時鐘換算台北 | `trading_day_domain.go` | `TDD:17`（UTC 案例） | asserts-oracle | produces-oracle | ✅ |
| NFR-4 | AI 用量記在時段評等 | 兩輪合計 300 / 30 | `analyst_consultation_domain.go` | `SGA:138` | asserts-oracle | produces-oracle | ✅ |
| NFR-5 | 每次時段執行留下成功 / 失敗檔數 | 計數正確 | `session_run_domain.go` | `SGA:261`、`SGA:285` | asserts-oracle | produces-oracle | ✅ |

## Orphans

| Code | Description | Verdict |
|------|-------------|---------|
| `tracked_symbol_grade_domain.go`、`dto/tracked_symbol_grade_dto.go` | 查詢回應另含「綜合分數」、「更新時間」、每筆時段評等的「佐證」與「建立時間」；PRD 只列綜合評等、綜合信心與時段評等五項 | undocumented |
| `session_grade_errors.go:6-7` | 刪除失敗與追蹤標的讀取失敗的執行紀錄失敗原因文字（「前一天結果刪除失敗」「追蹤標的讀取失敗」）PRD 未定 | undocumented |
| `cmd/server/config.go` | 背景 job 總開關與檢查間隔設定（`BACKGROUND_JOBS_ENABLED`、`TRADING_SESSION_JOB_INTERVAL_SECONDS`），PRD 未提；來自專案背景作業規則 | undocumented |
| `session_grade_service.go:127-130` | 指定市場類別但標的為空白時被拒「標的為必填」，PRD 未列 | undocumented |

無任何 orphan 落在 Out of Scope（無補跑、無重試、無管理 API、無價格）。

## Summary

- Conforms: 51/56 clauses ✅ (91%)
- Violations: —
- Mis-asserted: —
- Partial: AC-14、AC-27、AC-28、BR-9、NFR-1
- Gaps: —
- Unclear: —
- Orphans: 4（皆 undocumented，非越界）

---

## 稽核後修正紀錄（2026-10-07）

| 項目 | 處理 |
| :--- | :--- |
| AC-14 錯過的時段不補跑無測試 | 補測試：週三 14:00 不在任何執行時段，不執行 |
| AC-27 週六查得到週五無測試 | 補測試：週六時鐘下執行檢查不做事，查詢仍回週五（2026-10-09）的綜合評等與盤後時段評等 |
| AC-28 週一盤前刪除週五無測試 | 補測試：週一 08:00 盤前以交易日 2026-10-05 刪除其他日子的時段與綜合評等 |
| BR-9 每檔 10 分鐘上限無測試 | 補測試：每檔分析時 AI 收到的期限在 9–10 分鐘內（改成無期限即失敗） |
| NFR-1 依序分析無測試 | 補測試：兩檔標的的 AI 分析從不重疊（改成並行即失敗） |
| O-1 查詢回應的額外欄位 | PRD Business Rules 補規則 |
| O-2 執行紀錄失敗原因文字 | PRD Edge Cases 補規則 |
| O-3 背景作業總開關與檢查頻率 | PRD Business Rules 補規則 |
| O-4 空白標的被拒 | PRD Edge Cases 補規則 |

修正後：56/56 clauses ✅（100%），0 orphans。

---

## Code review 後修正紀錄（2026-10-07，PR #7）

| Review 意見 | 處理 | 新增條款 / 測試 |
| :--- | :--- | :--- |
| 盤前錯過時，查詢整天回傳前一天結果 | 保留「只有盤前會刪除」（PRD 既定），改為查詢只列每檔最新一天 | 新 AC「同一標的有兩天的結果時只查得到最新一天」→ `SGA` `TestGetTrackedSymbolGrades_ShowsOnlyTheNewestDayOfSymbolsStillTracked` ✅ |
| 依序執行跨出時段，後面的標的混入下一時段新聞 | 時段新聞範圍加上終點（時段收盤）；依序分析維持（PRD Performance） | BR-2 改寫 → `TDD:17`（終點）、`news_collection_domain_test.go`（終點不含）、`SGA` 盤前測試（09:10 新聞不採用）✅ |
| 失敗標的的原因與 AI 用量遺失 | 執行紀錄記下失敗標的、原因與 AI 用量合計 | 新 BR → `SGA` `OneFailingSymbol…`、`CountsEveryStep…`、`REPO` ✅ |
| 每分鐘一次 duplicate-key INSERT 錯誤 | 改為 insert-or-skip（不再報錯） | AC-13 → `REPO` `RecordsEachTradingSessionOnce`、`ReportsStorageFailuresApartFromExistingRuns` ✅ |
| 查詢 N+1 | 一次讀回所有列出交易日的時段評等 | `SGA` `ShowsOnlyTheNewestDay…`（`FindByTradingDays` 只呼叫一次）✅ |
| 暫停追蹤的標的仍出現在查詢 | 查詢只列仍追蹤中的標的 | 新 AC「暫停追蹤的標的不出現在查詢結果」→ 同上測試 ✅ |
| `ToSessionGradeEntity` 讀別人的欄位（Feature Envy） | 欄位對映移到 `AnalysisConclusionDomain.ToSessionGradeEntity`，consultation 只補自己的用量與模型 | 行為不變，既有測試覆蓋 ✅ |
| 權重打錯字被靜默改成預設 | 有設定但無法解讀 → 拒絕啟動；0 / 負數仍用預設（PRD 既定） | 新 BR → `cmd/server/config_test.go` ✅ |

修正後：59/59 clauses ✅（100%）。
