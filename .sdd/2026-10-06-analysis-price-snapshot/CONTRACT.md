# Contract Traceability Matrix — 2026-10-06-analysis-price-snapshot（分析時價格）

Contract: PRD.md（v1.0, Finalized）
Design map: ARCH.md
Implementation: `internal/domain/{models,service,interface}`、`internal/infrastructure/{binance,yahoofinance,twse,cache}`、`cmd/server/dependencies.go`（branch `feature/analysis-price-snapshot`）
Oracle: Acceptance Criteria（20 clauses：AC × 10、BR × 8、NFR × 2）

> 稽核方式：靜態契約一致性稽核。每條 clause 的 oracle 先只依 PRD 文字推導（下表「Spec-expected」欄），再分別獨立判斷「測試是否斷言 oracle」與「正式碼是否產出 oracle」。僅執行對應到 clause 的單一測試作佐證（全部綠燈），判定依據為 oracle 比對而非通過與否；不撰寫新探針、不執行自創情境。

## Oracle 橋接（Phase 3 step 0）

依 UL-MAP 與 ARCH：
- 「分析時價格」↔ `AnalysisResult.Price/PriceCurrency/PricedAt/PriceSource`（保存）↔ API `result.priceAtAnalysis`（查詢）；「未取得」↔ `Price.Valid=false`、`PricedAt=nil`、API `priceAtAnalysis: null`（欄位一律出現）。
- 「價格來源」顯示名：`Binance`、`Yahoo 財經`、`證交所`；分別對應 `BinancePriceProxy`、`YahooFinancePriceProxy`、`TwsePriceProxy`，經 `PriceSnapshotService.CapturePrice` 依市場類別選擇。
- 「已完成」↔ 分析事件 `Status = "succeeded"`。
- 「精確小數／不經浮點」↔ `shopspring/decimal` 由來源字面值直接轉換 + PostgreSQL `numeric` 欄位 + JSON 以字串輸出。

## Clauses

| ID | Clause | Spec-expected (oracle) | Impl | Test | Test audit | Code audit | Status |
|----|--------|------------------------|------|------|------------|------------|--------|
| AC-1 | Scenario: 加密貨幣記錄 Binance 對 USDT 的最新成交價 — Given BTC 在 Binance 對 USDT 的最新成交價為 86607.62 / When BTC（加密貨幣）的分析完成 / Then 分析時價格為 86607.62，計價幣別 USDT，價格來源「Binance」 / And 價格時間為分析完成的時間 | 完成的分析結果帶有價格 86607.62、幣別 USDT、來源 Binance，價格時間＝分析完成（取價）當下 | `price_snapshot_service.go:24-27`；`binance_price_proxy.go:34,42`；`analysis_conclusion_domain.go:104-109`；`symbol_analysis_service.go:130-131` | `binance_price_proxy_test.go:36`（BTCUSDT、USDT、Binance、PricedAt=clock.Now）；`symbol_analysis_application_test.go:772`（86607.62/USDT/Binance 隨結果保存）；`symbol_analysis_application_test.go:811`（crypto→Binance）；`analysis_domain_test.go:195`（PricedAt 帶入） | asserts-oracle | produces-oracle | ✅ conforms |
| AC-2 | Scenario: 美股記錄 Yahoo 財經的最新市價 — Given AAPL 在 Yahoo 財經的最新市價為 332.94 USD，報價時間為 2026-10-06 22:33:42（UTC） / When AAPL（美股）的分析完成 / Then 分析時價格為 332.94，計價幣別 USD，價格時間 2026-10-06 22:33:42（UTC），價格來源「Yahoo 財經」 | 結果帶有 332.94、USD、價格時間 2026-10-06 22:33:42 UTC（來源報價時間）、來源 Yahoo 財經 | `yahoo_finance_price_proxy.go:42-54`；`price_snapshot_service.go:20` | `yahoo_finance_price_proxy_test.go:31`；`symbol_analysis_application_test.go:811`（usStock→Yahoo 財經） | asserts-oracle | produces-oracle | ✅ conforms |
| AC-3 | Scenario: 台股記錄證交所最近交易日收盤價 — Given 證交所公布 2330 在 2026-10-05 的收盤價為 2575.00 / When 2330（台股）的分析完成 / Then 分析時價格為 2575.00，計價幣別 TWD，價格時間 2026-10-05（台北時間），價格來源「證交所」 | 結果帶有數值等於 2575.00、TWD、價格時間 2026-10-05 00:00 台北時間、來源證交所 | `twse_price_proxy.go:49-67,90-100`；`price_snapshot_service.go:22-23` | `twse_price_proxy_test.go:32`（2575.00 等值、TWD、證交所、2026-10-04T16:00Z）；`symbol_analysis_application_test.go:811` | asserts-oracle | produces-oracle | ✅ conforms |
| AC-4 | Scenario: 小數位數很多的價格原樣保存 — Given 某幣種在 Binance 的最新成交價為 0.00001234 / When 該幣種的分析完成 / Then 分析時價格為 0.00001234，沒有被四捨五入 | 保存並可讀回的價格恰為 0.00001234，未被進位/截斷 | `binance_price_proxy.go:19-21,39`；`analysis_result.go:21`（`numeric`）；`analysis_conclusion_domain.go:105` | `binance_price_proxy_test.go:36-42`；`analysis_domain_test.go:195`；`analysis_repository_test.go:139` | asserts-oracle | produces-oracle | ✅ conforms |
| AC-5 | Scenario: Binance 沒有該交易對 — Given Binance 沒有 XYZUSDT 交易對 / When XYZ（加密貨幣）的分析完成 / Then 分析事件狀態為「已完成」 / And 分析時價格為「未取得」 | 事件為已完成；結果的分析時價格為未取得 | `binance_price_proxy.go:34-37`（非 2xx → error）；`price_snapshot_service.go:28-30`；`symbol_analysis_service.go:130-136,164-165` | `binance_price_proxy_test.go:45`（unknown pair → error）；`symbol_analysis_application_test.go:793`（取價失敗 → succeeded、Price 無效、PricedAt nil） | asserts-oracle | produces-oracle | ✅ conforms |
| AC-6 | Scenario: 價格來源沒有回應 — Given Yahoo 財經沒有回應 / When AAPL（美股）的分析完成 / Then 分析事件狀態為「已完成」 / And 分析時價格為「未取得」 | 事件為已完成；分析時價格為未取得 | `http_body_reader.go:26-28`；`yahoo_finance_price_proxy.go:42-45`；`price_snapshot_service.go:28-30` | `http_body_reader_test.go:40,52`（無法連線/逾時 → error）；`dependencies_test.go:120`（usStock 503 → error）；`symbol_analysis_application_test.go:793` | asserts-oracle | produces-oracle | ✅ conforms |
| AC-7 | Scenario: 證交所清單沒有該代號 — Given 證交所最近交易日的收盤資料中沒有 9999 / When 9999（台股）的分析完成 / Then 分析事件狀態為「已完成」 / And 分析時價格為「未取得」 | 事件為已完成；分析時價格為未取得 | `twse_price_proxy.go:54-57`；`price_snapshot_service.go:28-30` | `twse_price_proxy_test.go:51`（9999 → error）；`symbol_analysis_application_test.go:793` | asserts-oracle | produces-oracle | ✅ conforms |
| AC-8 | Scenario: 價格不是正數 — Given 價格來源回傳的價格為 0 / When 分析完成 / Then 分析時價格為「未取得」 | 價格 0（或負數）時分析時價格為未取得 | `price_quote_vo.go:19-22`；三個 proxy 皆經 `NewPriceQuoteVo`（`binance_price_proxy.go:42`、`yahoo_finance_price_proxy.go:54`、`twse_price_proxy.go:66`） | `price_quote_vo_test.go:12`；`binance_price_proxy_test.go:53`；`yahoo_finance_price_proxy_test.go:50`；`twse_price_proxy_test.go:51`（9997 = 0.00） | asserts-oracle | produces-oracle | ✅ conforms |
| AC-9 | Scenario: 查詢取得價格的分析 — Given 一筆已完成且分析時價格為 86607.62 USDT 的分析事件 / When 使用者查詢該分析事件 / Then 分析結果顯示價格 86607.62、計價幣別 USDT、價格時間與價格來源 | 查詢回應的分析結果中顯示 86607.62、USDT、價格時間、來源 | `analysis_event_domain.go:94-97,109`；`price_at_analysis_dto.go:9-14`；`analysis_result_dto.go:16` | `analysis_event_controller_test.go:154`（完整 JSON 比對 `priceAtAnalysis`）；`analysis_domain_test.go:213` | asserts-oracle | produces-oracle | ✅ conforms |
| AC-10 | Scenario: 查詢未取得價格的分析 — Given 一筆已完成但分析時價格未取得的分析事件 / When 使用者查詢該分析事件 / Then 分析結果明確顯示分析時價格未取得 | 查詢回應明確呈現「未取得」（欄位存在且為空值，而非省略） | `analysis_event_domain.go:94-97`；`analysis_result_dto.go:16`（無 `omitempty`） | `analysis_event_controller_test.go:238`（欄位存在且為 `null`）；`analysis_domain_test.go:213` | asserts-oracle | produces-oracle | ✅ conforms |
| BR-1 | Flow: AI 提交結論並通過正規化 → 依市場類別取得價格（最多等待 10 秒）→ 與分析結果一起保存 → 分析事件改為已完成。 | 已完成的分析，其結果在保存時即帶有（或明確未取得）分析時價格；保存後事件才標為已完成（10 秒上限另見 NFR-1） | `symbol_analysis_service.go:124-137,164-165` | `symbol_analysis_application_test.go:772,793` | asserts-oracle | produces-oracle | ✅ conforms |
| BR-2 | 價格來源：加密貨幣 → Binance（標的 + USDT 交易對）；美股 → Yahoo 財經；台股 → 證交所每日收盤行情。 | 加密貨幣以「標的+USDT」向 Binance 取價；美股向 Yahoo 財經；台股向證交所每日收盤資料 | `price_snapshot_service.go:20-27`；`dependencies.go:105-111`；`binance_price_proxy.go:34` | `symbol_analysis_application_test.go:811`；`dependencies_test.go:120`；`binance_price_proxy_test.go:40`（BTCUSDT） | asserts-oracle | produces-oracle | ✅ conforms |
| BR-3 | 價格時間：Binance 為取得價格的時間；Yahoo 財經為來源提供的報價時間；證交所為收盤資料日期（台北時間當日 00:00）。 | Binance＝取價當下；Yahoo＝來源報價時間；證交所＝收盤日台北 00:00 | `binance_price_proxy.go:42`；`yahoo_finance_price_proxy.go:54`；`twse_price_proxy.go:90-100` | `binance_price_proxy_test.go:41`；`yahoo_finance_price_proxy_test.go:36`；`twse_price_proxy_test.go:46` | asserts-oracle | produces-oracle | ✅ conforms |
| BR-4 | 價格必須大於 0，否則視為未取得。 | 價格 ≤ 0 時分析時價格為未取得 | `price_quote_vo.go:19-22`；`price_snapshot_service.go:28-30` | `price_quote_vo_test.go:12`（0、-1） | asserts-oracle | produces-oracle | ✅ conforms |
| BR-5 | 價格以來源提供的十進位字串保存，不經浮點數轉換。 | 保存的價格與來源字面值逐位相同，即使該值無法以浮點精確表示也不失真 | `binance_price_proxy.go:20`、`yahoo_finance_price_proxy.go:25`（`decimal.UnmarshalJSON` → `NewFromString`）；`twse_price_proxy.go:58`；`analysis_result.go:21` | `binance_price_proxy_test.go:36`（0.00001234）；`yahoo_finance_price_proxy_test.go:31`（332.94）；`analysis_repository_test.go:139`（0.000012345678901234） | shallow | produces-oracle | 🟠 mis-asserted |
| BR-6 | 證交所每日收盤資料保留 1 小時後重新取得。 | 1 小時內重複取價不重新下載，滿 1 小時後重新下載 | `twse_price_proxy.go:22,45`；`refreshing_cache.go:28-45` | `twse_price_proxy_test.go:64`（59:59 不下載、60:00 下載） | asserts-oracle | produces-oracle | ✅ conforms |
| BR-7 | Edge: 價格來源回傳格式無法解析 → 未取得。 | 回應格式無法解析時分析時價格為未取得 | `binance_price_proxy.go:39-41`；`yahoo_finance_price_proxy.go:47-52`；`twse_price_proxy.go:58-65,76-81`；`price_snapshot_service.go:28-30` | `binance_price_proxy_test.go:52`；`yahoo_finance_price_proxy_test.go:47-49`；`twse_price_proxy_test.go:51`（空價、壞日期）、`:80`（malformed/empty）；`symbol_analysis_application_test.go:793` | asserts-oracle | produces-oracle | ✅ conforms |
| BR-8 | Edge: 取價格時發生任何錯誤 → 未取得，分析照常完成。 | 任何取價錯誤 → 分析時價格未取得，且分析事件仍為已完成 | `price_snapshot_service.go:27-31`（回傳 `nil`，無錯誤外洩）；`symbol_analysis_service.go:130-136` | `symbol_analysis_application_test.go:793` | asserts-oracle | produces-oracle | ✅ conforms |
| NFR-1 | Performance: 取價格最多增加 10 秒的分析時間。 | 取價對分析時間的增加不超過 10 秒（來源卡住時於 10 秒內放棄並記為未取得） | `dependencies.go:31,61-62`（`http.Client.Timeout` 10s）；`dependencies.go:113`+`buildPriceProviderCatalog`（價格 proxy 共用該 client）；每次取價僅一個 HTTP 請求 | `dependencies_test.go:105`（client 逾時 10s）；`http_body_reader_test.go:52` | shallow | produces-oracle | 🟠 mis-asserted |
| NFR-2 | Accuracy: 價格以精確小數保存。 | 保存與讀回的價格為精確小數，與來源值完全一致 | `analysis_result.go:21`（`decimal.NullDecimal` + `type:numeric`） | `analysis_repository_test.go:139` | shallow | produces-oracle | 🟠 mis-asserted |

### 判定說明（非 ✅ 項）

- **BR-5 🟠（shallow）**：正式碼確實不經浮點——`decimal.Decimal.UnmarshalJSON` 對未加引號的 JSON 數字也是取原始位元組再 `NewFromString`（shopspring/decimal v1.4.0 `decimal.go:1764-1780`），TWSE 用 `decimal.NewFromString`。但所有測試值（`0.00001234`、`332.94`、`86607.62`、`2575.00`、`0.000012345678901234`）經 `strconv.ParseFloat(...,64)` 再格式化都能逐位還原（已以 scratch 程式驗證），因此若有人把任一 proxy 改成先解析成 `float64` 再 `decimal.NewFromFloat`，這些測試仍會綠燈。缺一個超過 float64 精度（≥ 17 位有效數字，如 `123456789.123456789` → float 會變 `123456789.12345679`）的來源字面值測試，至少涵蓋 Binance（字串）與 Yahoo（JSON 數字）兩條路徑。
- **NFR-2 🟠（shallow）**：`analysis_repository_test.go:139` 用 `0.000012345678901234`（14 位有效數字），在 PostgreSQL `float8`／`double precision` 欄位也能以最短表示法無損往返，故欄位型別若退化為浮點，此測試不會失敗。需改用超過 float64 精度的值（如 `123456789.123456789`）才能真正釘住「精確小數保存」。
- **NFR-1 🟠（shallow）**：10 秒上限完全來自共用的 `externalHttpClient`（`dependencies.go:31,62`），`PriceSnapshotService.CapturePrice`／`symbol_analysis_service.go:130` 傳入的是 10 分鐘的 `analysisContext`，自身沒有取價專屬的期限。`dependencies_test.go:105` 只釘住 client 的 Timeout，`dependencies_test.go:120` 用的是 `server.Client()`（無 10 秒逾時），沒有任何測試確認「價格 proxy 實際被組裝在 10 秒 client 上」或「取價卡住時 10 秒內放棄、分析仍完成」。若有人把價格 proxy 改接到別的 `HttpBodyReader`/`http.Client`，測試全綠但上限消失。程式碼目前的行為符合 oracle（每次取價僅一次請求；TWSE singleflight 下載雖 `WithoutCancel`，仍受同一 client 10 秒逾時約束）。
- **AC-1 備註（仍判 ✅）**：應用層測試 `symbol_analysis_application_test.go:772` 未斷言保存的 `PricedAt`；「價格時間＝分析完成（取價）當下」由 `binance_price_proxy_test.go:41`（`PricedAt = clock.Now()`）與 `analysis_domain_test.go:195`（`PricedAt` 帶入結果）串接釘住。另 PRD 本身 AC-1「價格時間為分析完成的時間」與 BR-3「Binance 為取得價格的時間」兩種說法並存；實作取的是取價當下（`binance_price_proxy.go:42`），早於事件 `FinishedAt`（`symbol_analysis_service.go:165`）一次存檔的時間差，業務上視為相同時點。
- **ARCH 偏差（不影響判定）**：ARCH 寫 Yahoo 以 `json.Number` 保留字面值，實作用 `decimal.Decimal`（`yahoo_finance_price_proxy.go:25`），效果等價（同樣不經浮點）。

## Orphans (code with no clause)

| Code | Description | Verdict |
|------|-------------|---------|
| `internal/domain/service/price_snapshot_service.go:20-26` | 市場類別不是 `twStock`/`crypto` 時一律落到 `UsStock`（Yahoo 財經），包含非法或空字串類別；PRD 只定義三種類別的來源。實務上類別已由 `NewMarketCategoryVo` 驗證，此分支應不可達，但屬未記載的預設行為 | undocumented |
| `internal/infrastructure/cache/refreshing_cache.go:27-45`（經 `twse_price_proxy.go:45`） | 證交所收盤資料快取的額外行為：下載失敗不快取（下一次立即重試）、併發呼叫共用一次下載、下載不受單一呼叫者取消影響。PRD 只寫「保留 1 小時後重新取得」；ARCH 有記載，PRD 未記載 | undocumented |

Out of Scope 檢查：未發現回測計算、既有分析補抓價格、匯率換算的實作（AutoMigrate 只新增可為 NULL 的欄位，舊分析呈現未取得；Yahoo 幣別直接取來源 `currency`，無換算）。無範圍外違規。

## Summary

- Conforms: 17/20 clauses ✅ (85%)
- Violations: —
- Mis-asserted: BR-5, NFR-1, NFR-2
- Partial: —
- Gaps: —
- Unclear: —
- Orphans: 2

---

## 稽核後修正紀錄（2026-10-06）

| 項目 | 處理 |
| :--- | :--- |
| BR-5 測試值可被浮點數往返 | 補測試：Binance（字串）與 Yahoo 財經（JSON 數字）各以 `123456789.123456789` 驗證不經浮點數 |
| NFR-2 測試值可被 double 保存 | 資料庫往返改用 `123456789.123456789012345678` |
| NFR-1 10 秒只來自 HTTP client | `PriceSnapshotService` 自帶 10 秒期限；補測試：來源卡住時於期限內放棄並回未取得 |
| O-1 不認得的市場類別改問 Yahoo 財經 | 修正：回未取得 |
| O-2 快取行為 | 保留，ARCH 已記載（`RefreshingCache`） |
| ARCH `json.Number` 寫法 | 已更正為 `decimal.Decimal` 直接解碼 |
