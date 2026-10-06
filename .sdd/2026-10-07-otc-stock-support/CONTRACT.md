# Contract Traceability Matrix — 2026-10-07-otc-stock-support（上櫃股票支援）

Contract: PRD.md（v1.0, Finalized）
Design map: ARCH.md
Implementation: `internal/infrastructure/tpex/`、`internal/domain/service/symbol_resolution_service.go`、`internal/domain/service/price_snapshot_service.go`、`cmd/server/dependencies.go`（branch `feature/otc-stock-support`）
Oracle: Acceptance Criteria（18 clauses：AC × 10、BR × 6、NFR × 2）

> 本文件為**靜態符合度稽核**：以 PRD 推導的 oracle 對照測試斷言與程式路徑，不執行自創情境。僅以「已對應到條款的單一測試」做佐證（皆為綠燈），判定不依賴 pass/fail。

## Clauses

`Spec-expected` 欄是 Phase 2 僅依 PRD 推導出的業務可觀察結果；稽核欄檢查其經 UL-MAP / ARCH 橋接後的具體形式。

橋接說明（Phase 3）：
- 「搜尋被拒，告知『找不到此標的』」↔ `service.ErrSymbolNotFound`（訊息即 `找不到此標的`，`internal/domain/service/news_search_errors.go:12`），controller 以 `errors.Is` 回 sentinel 訊息（`internal/controller/error_response_table.go:20-21`）。
- 「新聞來源暫時無法使用」↔ `service.ErrNewsProvidersUnavailable`（`news_search_errors.go:13`），同上以 `errors.Is` 對映，被 `%w` 包裝仍回原 sentinel 訊息。
- 「分析中」↔ status `running`；「已完成」↔ `succeeded`；「分析時價格為未取得」↔ `CapturePrice` 回 `nil` → `AnalysisResult.Price.Valid == false`。
- 「價格來源『櫃買中心』／『證交所』」↔ `tpex.TpexPriceSource` / `twse.TwsePriceSource`。
- 「該股無價格」「取不到資料」↔ proxy 回 error（service 層轉為 `nil` 價格或 `ErrNewsProvidersUnavailable`）。

| ID | Clause | Spec-expected (oracle) | Impl | Test | Test audit | Code audit | Status |
|----|--------|------------------------|------|------|------------|------------|--------|
| AC-1 | Scenario: 上市股票行為不變 — Given 證交所上市公司 2330 的公司簡稱為「台積電」 When 使用者搜尋標的 2330、市場類別台股的新聞 Then 系統以「台積電」搜尋新聞 | 以「台積電」作為新聞搜尋字（且不需詢問櫃買中心） | `internal/domain/service/symbol_resolution_service.go:34-38`；順序 `cmd/server/dependencies.go:126` | `internal/application/tests/otc_stock_support_test.go:54` `TestTaiwanStockResolution_ListedStocksStillComeFromTwse`（TPEx mock 無 expectation，被呼叫即失敗） | asserts-oracle | produces-oracle | ✅ conforms |
| AC-2 | Scenario: 上櫃股票以櫃買中心簡稱搜尋 — Given 證交所上市公司中沒有 6182 And 櫃買中心資料中 6182 的公司簡稱為「合晶」 When 搜尋 6182 台股新聞 Then 系統以「合晶」搜尋新聞 | 以「合晶」作為新聞搜尋字 | `symbol_resolution_service.go:34-38`；`internal/infrastructure/tpex/tpex_open_data_proxy.go:50-57` | `otc_stock_support_test.go:62` `TestTaiwanStockResolution_FallsBackToTpexForOtcStocks`；`internal/infrastructure/tpex/tests/tpex_open_data_proxy_test.go:43` | asserts-oracle | produces-oracle | ✅ conforms |
| AC-3 | Scenario: 上市上櫃都找不到 — Given 證交所與櫃買中心都沒有 9999 When 搜尋 9999 台股新聞 Then 搜尋被拒，告知「找不到此標的」 | 搜尋被拒，提示「找不到此標的」 | `symbol_resolution_service.go:47`；`internal/controller/news_controller.go:15` | `otc_stock_support_test.go:84`（ErrorIs `ErrSymbolNotFound`、未搜尋）；訊息對映 `internal/controller/tests/news_controller_test.go:100-102` | asserts-oracle | produces-oracle | ✅ conforms |
| AC-4 | Scenario: 證交所取不到但櫃買中心找到 — Given 證交所上市公司清單暫時取不到 And 櫃買中心 6182 簡稱「合晶」 When 搜尋 6182 台股新聞 Then 系統以「合晶」搜尋新聞 | 證交所失敗不影響，以「合晶」作為搜尋字 | `symbol_resolution_service.go:35-41`（found 優先於 error） | `otc_stock_support_test.go:69` `TestTaiwanStockResolution_UsesTpexWhenTwseIsUnavailable` | asserts-oracle | produces-oracle | ✅ conforms |
| AC-5 | Scenario: 證交所取不到且櫃買中心沒有 — Given 證交所清單暫時取不到 And 櫃買中心沒有 2330 When 搜尋 2330 台股新聞 Then 搜尋被拒，告知「新聞來源暫時無法使用」 | 搜尋被拒，提示「新聞來源暫時無法使用」 | `symbol_resolution_service.go:43-46`；`news_controller.go:16` | `otc_stock_support_test.go:85`（ErrorIs `ErrNewsProvidersUnavailable`、未搜尋）；訊息對映 `news_controller_test.go:103-107` | asserts-oracle | produces-oracle | ✅ conforms |
| AC-6 | Scenario: 證交所沒有且櫃買中心取不到 — Given 證交所上市公司中沒有 6182 And 櫃買中心資料暫時取不到 When 搜尋 6182 台股新聞 Then 搜尋被拒，告知「新聞來源暫時無法使用」 | 搜尋被拒，提示「新聞來源暫時無法使用」（非「找不到此標的」） | `symbol_resolution_service.go:43-46` | `otc_stock_support_test.go:86` | asserts-oracle | produces-oracle | ✅ conforms |
| AC-7 | Scenario: 上櫃股票可以發起分析 — Given 櫃買中心資料中 6182 的公司簡稱為「合晶」 When 使用者發起標的 6182、市場類別台股的分析 Then 使用者取得分析事件編號，狀態「分析中」 | 發起成功，取得分析事件編號且狀態為「分析中」 | `internal/domain/service/symbol_analysis_service.go:45`（共用 `ResolveSymbol`）→ `:82` | 無：`symbol_analysis_application_test.go:87` 只以 crypto（BTC）驗證發起；所有分析 fixture 的台股目錄只有單一 proxy（`symbol_analysis_application_test.go:67`、`analysis_event_controller_test.go:48`），沒有「證交所沒有、櫃買中心有」的發起分析測試 | no-test | produces-oracle | 🟡 partial |
| AC-8 | Scenario: 記錄櫃買中心收盤價 — Given 櫃買中心公布 6182 在 2026-10-06 的收盤價為 128.00 When 6182（台股）的分析完成 Then 分析時價格為 128.00，計價幣別 TWD，價格時間 2026-10-06（台北時間），價格來源「櫃買中心」 | 價格 128.00、TWD、2026-10-06 台北時間、來源「櫃買中心」 | `tpex_open_data_proxy.go:59-78`；`price_snapshot_service.go:35-39`；`symbol_analysis_service.go:130-131` | `tpex_open_data_proxy_test.go:61`（128.00／TWD／櫃買中心／2026-10-05T16:00Z＝台北 10-06 00:00）＋`otc_stock_support_test.go:98`（證交所無 → 採櫃買中心報價）＋`symbol_analysis_application_test.go:773`（報價寫入分析結果） | asserts-oracle | produces-oracle | ✅ conforms |
| AC-9 | Scenario: 上櫃股票當日無成交 — Given 櫃買中心資料中 6182 當日沒有收盤價 When 6182（台股）的分析完成 Then 分析事件狀態為「已完成」 And 分析時價格為「未取得」 | 分析仍「已完成」，價格「未取得」 | `tpex_open_data_proxy.go:69-72`；`price_snapshot_service.go:40`；`symbol_analysis_service.go:130-136` | `tpex_open_data_proxy_test.go:76`（`---` → 無價格）＋`otc_stock_support_test.go:115,121`（全來源失敗 → nil）＋`symbol_analysis_application_test.go:794`（無價仍 succeeded、Price 無值） | asserts-oracle | produces-oracle | ✅ conforms |
| AC-10 | Scenario: 上市股票價格來源不變 — Given 證交所公布 2330 的收盤價為 2575.00 When 2330（台股）的分析完成 Then 分析時價格為 2575.00，價格來源「證交所」 | 價格 2575.00、來源「證交所」（不改用櫃買中心） | `price_snapshot_service.go:35-38`；`internal/infrastructure/twse/twse_open_data_proxy.go:71-89` | `otc_stock_support_test.go:114,119-120`（2330 採證交所；TPEx mock 無 2330 expectation）＋`internal/infrastructure/twse/tests/twse_price_proxy_test.go:32`（2575.00、證交所） | asserts-oracle | produces-oracle | ✅ conforms |
| BR-1 | 台股標的辨識順序：證交所上市公司 → 櫃買中心上櫃股票；先找到者為準。 | 先問證交所再問櫃買中心；證交所找到即採用、不再問櫃買中心 | `symbol_resolution_service.go:34-38`；`dependencies.go:124-129` | `otc_stock_support_test.go:54`（先找到者為準）＋`cmd/server/dependencies_test.go:152` `TestBuildSymbolDirectoryCatalog_AsksTwseBeforeTpex`（順序） | asserts-oracle | produces-oracle | ✅ conforms |
| BR-2 | 任一邊找到即辨識成功；兩邊都明確沒有 → 找不到此標的；有一邊取不到資料且另一邊沒找到 → 新聞來源暫時無法使用。 | 三種結果：任一找到＝成功；皆明確沒有＝「找不到此標的」；一邊取不到且另一邊沒有＝「新聞來源暫時無法使用」 | `symbol_resolution_service.go:33-47` | `otc_stock_support_test.go:62,69,76-96` | asserts-oracle | produces-oracle | ✅ conforms |
| BR-3 | 台股分析時價格依同樣順序取價：證交所有該股收盤價則用證交所，否則用櫃買中心。 | 證交所有價→證交所價；否則→櫃買中心價；皆無→未取得 | `price_snapshot_service.go:35-40`；`dependencies.go:131-137` | `otc_stock_support_test.go:98`；`dependencies_test.go:121`（`TwStock` 為 [twse, tpex]） | asserts-oracle | produces-oracle | ✅ conforms |
| BR-4 | 櫃買中心資料（上櫃股票每日收盤行情）保留 1 小時後重新取得；失敗時沿用上一次成功的資料；從未成功過則視為取不到。 | (a) 1 小時內重用、滿 1 小時重新下載；(b) 重新下載失敗時沿用上一次成功資料；(c) 從未成功→取不到 | `tpex_open_data_proxy.go:22,46`（`time.Hour`）；`internal/infrastructure/cache/refreshing_cache.go:28-58` | (c) `tpex_open_data_proxy_test.go:86`；(a) 僅 `tpex_open_data_proxy_test.go:58` 斷言同一時刻多次查詢只下載 1 次，**未推進時鐘驗證 1 小時界線**；(b) **櫃買中心層無測試**，只有泛用 `cache/tests/refreshing_cache_test.go:138`（不綁櫃買中心設定） | shallow | produces-oracle | 🟠 mis-asserted |
| BR-5 | 櫃買中心收盤價欄位不是數字（例如當日無成交）→ 該股無價格。 | 該股無價格（未取得） | `tpex_open_data_proxy.go:68-72` | `tpex_open_data_proxy_test.go:76`（`5483` Close `" ---"` → error）＋`otc_stock_support_test.go:121` | asserts-oracle | produces-oracle | ✅ conforms |
| BR-6 | Edge Case：櫃買中心回傳空資料或格式錯誤 → 視為取不到資料。 | 辨識與取價都視為「取不到」 | `tpex_open_data_proxy.go:81-100` | `tpex_open_data_proxy_test.go:86`（502／`{`／`[]`／無可用列 → 查名與取價皆 error、found=false） | asserts-oracle | produces-oracle | ✅ conforms |
| NFR-1 | Performance：櫃買中心資料約 5 MB，與其他外部來源相同最多等待 10 秒；1 小時快取避免每次查詢都重新下載。 | 單次櫃買中心下載最多等 10 秒；約 5 MB 可被接受；1 小時內不重新下載 | 10 秒：`dependencies.go:32,64-66,118`（共用 `httpBodyReader`）＋`price_snapshot_service.go:33`；5 MB：`internal/infrastructure/httpfetch/http_body_reader.go:10`（上限 10 MB）；1 小時：`tpex_open_data_proxy.go:22` | 10 秒：`dependencies_test.go:106`＋`symbol_analysis_application_test.go:833`；1 小時：**無櫃買中心層時鐘推進測試**（同 BR-4(a)） | shallow | produces-oracle | 🟠 mis-asserted |
| NFR-2 | Compatibility：上市股票的辨識與價格結果與現行完全相同。 | 上市股票的搜尋字、價格、價格時間、來源與改版前一致 | `symbol_resolution_service.go:34-38`；`price_snapshot_service.go:35-38`；`twse_open_data_proxy.go:84`＋`internal/utilities/republic_of_china_date_parser.go:21-31`（純搬移，邏輯與原 `tradingDate()` 相同） | `otc_stock_support_test.go:54,114`；既有 `twse_price_proxy_test.go:32,51`、`twse_listed_company_proxy_test.go:34`；`internal/utilities/tests/republic_of_china_date_parser_test.go:12,21` | asserts-oracle | produces-oracle | ✅ conforms |

## Orphans (code with no clause)

Out of Scope 檢查：興櫃（端點為 `tpex_mainboard_daily_close_quotes`，僅上櫃主板，未觸及興櫃）、上櫃即時價格（無）、新增新聞來源（`buildNewsProviderCatalog` 未改）— 皆無越界。

| Code | Description | Verdict |
|------|-------------|---------|
| `internal/infrastructure/tpex/tpex_open_data_proxy.go:93-95` | 下載時丟棄 `CompanyName` 為空白的列；此類列不只無法辨識，**連其收盤價也一併取不到**（同代號若名稱缺漏但有正常 Close，價格仍為未取得）。PRD 未規範「名稱缺漏的列」 | undocumented |
| `internal/infrastructure/tpex/tpex_open_data_proxy.go:73-76` | 單列 `Date` 無法解析時該股無價格（測試 `tpex_open_data_proxy_test.go:25,79` 的 `9997`）。BR-5 只規範收盤價非數字，未規範日期異常；行為與證交所一致 | undocumented |

## Summary

- Conforms: 15/18 clauses ✅ (83%)
- Violations: —
- Mis-asserted: BR-4, NFR-1（櫃買中心 1 小時快取與失敗沿用舊資料未在櫃買中心層被斷言）
- Partial: AC-7（上櫃股票發起分析無測試）
- Gaps: —
- Unclear: —
- Orphans: 2（皆 undocumented，非越界）

---

## 稽核後修正紀錄（2026-10-07）

| 項目 | 處理 |
| :--- | :--- |
| BR-4、NFR-1 快取 1 小時與失敗沿用未在櫃買中心層驗證 | 補測試：59 分 59 秒內不重抓、滿 1 小時重抓；更新失敗時仍回傳上一次的「合晶」與 128.00 |
| AC-7 上櫃股票發起分析無測試 | 補測試：證交所沒有 6182、櫃買中心有「合晶」→ 建立分析中的分析事件，搜尋字為「合晶」 |
| O-1 名稱空白的資料列被排除 | PRD Edge Cases 補規則 |
| O-2 日期無法解析 → 無價格 | PRD Edge Cases 補規則 |
