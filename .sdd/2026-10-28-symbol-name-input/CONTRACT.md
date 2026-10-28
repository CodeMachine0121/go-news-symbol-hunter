# Contract Traceability Matrix — symbol-name-input

Contract: PRD.md
Design map: ARCH.md
Implementation: `internal/`（branch `feature/symbol-name-input`）
Oracle: Acceptance Criteria + Business Rules + NFR（40 clauses）

> Static conformance audit：依規格推導的預期結果，分別檢查測試斷言與程式路徑；不執行自行發明的情境。僅在 AC-13 執行過對應的單一測試作佐證。

## Clauses

`Test audit` / `Code audit` 為 Claude 的 shadow verdict；`Jev test` / `Jev code` 為 Jev 的判斷與信心；`Status` / `Route` 取自 `jev_audit.py`（以最後一次、參照範圍修正後的執行為準）。

縮寫：`SDD` = `internal/domain/models/domains/symbol_directory_domain.go`、`SRS` = `internal/domain/service/symbol_resolution_service.go`、`SAS` = `internal/domain/service/symbol_analysis_service.go`、`DT` = `internal/domain/models/domains/tests/symbol_directory_domain_test.go`、`NIT` = `internal/application/tests/symbol_name_input_test.go`、`SAT` = `internal/application/tests/symbol_analysis_application_test.go`。

| ID | Clause | Spec-expected (oracle) | Impl | Test | Test audit | Code audit | Jev test | Jev code | Status | Route |
|----|--------|------------------------|------|------|------------|------------|----------|----------|--------|-------|
| AC-01 | 以台股代號發起分析 | 分析 2330，名稱台積電 | SDD:57、SAS:38 | DT:61、SAT:480 | asserts-oracle | produces-oracle | asserts-oracle 1.00 | produces-oracle 0.47 | ✅ conforms | escalate (low confidence) |
| AC-02 | 以公司簡稱發起分析 | 填台積電 → 分析 2330，名稱台積電 | SDD:57 | NIT:51、DT:62 | asserts-oracle | produces-oracle | asserts-oracle 1.00 | produces-oracle 0.71 | ✅ conforms | recheck |
| AC-03 | 上市 ETF 可以被辨識 | 填 0050 → 分析 0050，名稱元大台灣50 | twse_open_data_proxy.go:51 | NIT:54、twse_symbol_directory_proxy_test.go:34 | asserts-oracle | produces-oracle | asserts-oracle 0.97 | produces-oracle 0.73 | ✅ conforms | recheck |
| AC-04 | 以上櫃 ETF 名稱發起分析 | 填元大美債20年 → 分析 00679B | tpex_open_data_proxy.go:50、SDD:57 | NIT:FindsOtcEtfsByName、DT:64 | asserts-oracle | produces-oracle | asserts-oracle 1.00 | produces-oracle 0.63 | ✅ conforms | recheck |
| AC-05 | 唯一部分符合即採用 | 填台積 → 分析 2330 | SDD:62-68 | DT:66 | asserts-oracle | produces-oracle | asserts-oracle 1.00 | produces-oracle 0.83 | ✅ conforms | recheck |
| AC-06 | 完全相同優先於部分符合 | 填元大台灣50 → 分析 0050，不算多個 | SDD:56-60 | DT:67 | asserts-oracle | produces-oracle | asserts-oracle 0.53 | produces-oracle 0.13 | ✅ conforms | escalate (low confidence) |
| AC-07 | 名稱符合多個標的 | 不建立分析事件；回覆「符合多個標的，請改填代號或更完整的名稱」＋最多 10 個候選（代號＋名稱） | SDD:73-77、error_response_table.go:19 | NIT:95、DT:116、analysis_event_controller_test.go:145 | asserts-oracle | produces-oracle | asserts-oracle 0.97 | produces-oracle 0.83 | ✅ conforms | recheck |
| AC-08 | 找不到標的 | 回覆「找不到此標的」，不建立分析事件 | SDD:70-71 | NIT:98、DT:138 | asserts-oracle | produces-oracle | asserts-oracle 0.93 | produces-oracle 0.68 | ✅ conforms | recheck |
| AC-09 | 權證不辨識 | 填 710001 → 找不到此標的 | tpex_open_data_proxy.go:57-59 | tpex_open_data_proxy_test.go:46 | asserts-oracle | produces-oracle | asserts-oracle 0.40 | produces-oracle 0.87 | ✅ conforms | escalate (low confidence) |
| AC-10 | 台股名單暫時取不到 | 回覆新聞來源暫時無法使用，不回覆找不到 | SRS:76-77 | NIT:TrustsOnlyASymbolMatch | asserts-oracle | produces-oracle | asserts-oracle 0.65 | produces-oracle 0.79 | ✅ conforms | recheck |
| AC-11 | 以加密貨幣代號發起分析 | 分析 BTC，名稱 Bitcoin | SDD:57 | SAT:90、DT:68 | asserts-oracle | produces-oracle | asserts-oracle 1.00 | produces-oracle 0.73 | ✅ conforms | recheck |
| AC-12 | 以幣種名稱發起分析，不分大小寫 | 填 bitcoin → 分析 BTC，名稱 Bitcoin | SDD:92-95 | NIT:57、DT:69 | asserts-oracle | produces-oracle | asserts-oracle 0.99 | produces-oracle 0.75 | ✅ conforms | recheck |
| AC-13 | 代號或名稱完全相同的有多個時取市值排名最前 | 填 Bitcoin → 分析 BTC | SDD:34-36、56-60 | DT:70 | asserts-oracle | produces-oracle | asserts-oracle 0.79 | produces-oracle 0.63 | ✅ conforms | recheck |
| AC-14 | 名稱符合多個幣種 | 回覆符合多個標的，候選依市值排名 | SDD:34-36、73-77 | DT:96、analysis_event_controller_test.go:145 | asserts-oracle | produces-oracle | asserts-oracle 0.89 | produces-oracle 0.36 | ✅ conforms | escalate (low confidence) |
| AC-15 | 找不到幣種 | 回覆找不到此標的，不建立分析事件 | SDD:70-71 | NIT:101 | asserts-oracle | produces-oracle | asserts-oracle 0.96 | produces-oracle 0.63 | ✅ conforms | recheck |
| AC-16 | 以美股代號發起分析 | 分析 TSLA，名稱 Tesla, Inc. | SDD:57、SRS:63-71 | DT:71、SAT:480 | asserts-oracle | produces-oracle | asserts-oracle 0.92 | produces-oracle 0.41 | ✅ conforms | escalate (low confidence) |
| AC-17 | 填的字不是已知標的時改以標的名稱比對 | 填 TESLA → 分析 TSLA | SDD:57、92-101 | NIT:60、DT:72 | asserts-oracle | produces-oracle | asserts-oracle 0.95 | produces-oracle 0.63 | ✅ conforms | recheck |
| AC-18 | 不計公司字尾後完全相同 | 填 Apple → 分析 AAPL | SDD:96-100 | DT:73 | asserts-oracle | produces-oracle | asserts-oracle 0.81 | produces-oracle 0.39 | ✅ conforms | escalate (low confidence) |
| AC-19 | 名稱符合多個美股 | 回覆符合多個標的，候選 SPY、SPYX、EFIV | SDD:73-77 | DT:99 | asserts-oracle | produces-oracle | asserts-oracle 0.76 | produces-oracle 0.73 | ✅ conforms | recheck |
| AC-20 | 只考慮美國掛牌的股票與 ETF | 填 Tesla → 分析 TSLA（排除 TSLA.TO、期貨） | yahoo_finance_proxy.go:106-108 | yahoo_finance_symbol_directory_test.go:29 | asserts-oracle | produces-oracle | asserts-oracle 0.41 | produces-oracle 0.39 | ✅ conforms | escalate (low confidence) |
| AC-21 | 美股資料來源可用但查無此標的 | 填 ZZZZ → 找不到此標的，不建立分析事件 | SRS:71、SDD:70 | NIT:104 | asserts-oracle | produces-oracle | asserts-oracle 0.96 | produces-oracle 0.67 | ✅ conforms | recheck |
| AC-22 | 美股資料來源暫時無法使用 | 以 TESLA 當代號分析，名稱空 | SRS:68-69 | NIT:63 | asserts-oracle | produces-oracle | asserts-oracle 0.97 | produces-oracle 0.97 | ✅ conforms | accept |
| AC-23 | 名稱發起的分析重用代號發起的分析 | 回傳既有分析事件，不重新分析 | SAS:38-52 | NIT:ANameReusesTheAnalysisStartedBySymbol | asserts-oracle | produces-oracle | asserts-oracle 0.95 | produces-oracle 0.41 | ✅ conforms | escalate (low confidence) |
| AC-24 | 超過重用時間窗時重新分析 | 建立新的分析事件，標的 2330 | SAS:46-70 | NIT:144 | asserts-oracle | produces-oracle | asserts-oracle 0.88 | produces-oracle 0.87 | ✅ conforms | accept |
| AC-25 | 追蹤標的照常分析 | 以台積電搜尋新聞並產生時段評等 | session_grade_service.go:90、SRS:32-51 | session_grade_application_test.go:170 | asserts-oracle | produces-oracle | asserts-oracle 0.72 | produces-oracle 0.56 | ✅ conforms | escalate (low confidence) |
| AC-26 | 追蹤上市 ETF | 以元大台灣50搜尋新聞並產生時段評等 | SRS:32-51、twse_open_data_proxy.go:51 | session_grade_application_test.go:212 | asserts-oracle | produces-oracle | asserts-oracle 0.75 | produces-oracle 0.64 | ✅ conforms | recheck |
| BR-1 | 比對順序：代號完全相同 → 名稱完全相同 → 名稱包含 | 前一步有結果不看下一步 | SDD:54-78 | DT:51 | asserts-oracle | produces-oracle | asserts-oracle 0.68 | produces-oracle 0.67 | ✅ conforms | recheck |
| BR-2 | 比對正規化 | 忽略空白、不分大小寫；美股不計標點與公司字尾 | SDD:14-19、92-101 | DT:69、73-74 | asserts-oracle | produces-oracle | asserts-oracle 0.48 | produces-oracle 0.88 | ✅ conforms | escalate (low confidence) |
| BR-3 | 唯一性 | 完全相同多筆取排序第一；部分符合多筆為符合多個標的 | SDD:26-78 | DT:51-129 | asserts-oracle | produces-oracle | asserts-oracle 0.72 | produces-oracle 0.49 | ✅ conforms | escalate (low confidence) |
| BR-4 | 候選 | 最多 10 個；台股依代號、加密貨幣依市值（無排名最後）、美股依來源順序 | SDD:26-78、coin_gecko_cryptocurrency_proxy.go:66-70 | DT:88-129、coin_gecko_cryptocurrency_proxy_test.go:34 | asserts-oracle | produces-oracle | asserts-oracle 0.83 | produces-oracle 0.33 | ✅ conforms | escalate (low confidence) |
| BR-5 | 台股辨識範圍 | 上市與上櫃每日收盤行情所列；權證不辨識 | twse/tpex proxy、dependencies.go | twse/tpex proxy tests | asserts-oracle | produces-oracle | asserts-oracle 0.72 | produces-oracle 0.77 | ✅ conforms | recheck |
| BR-6 | 美股辨識範圍 | 美國交易所、股票或 ETF | yahoo_finance_proxy.go:20-30、95-118 | yahoo_finance_symbol_directory_test.go:29 | asserts-oracle | produces-oracle | asserts-oracle 0.28 | produces-oracle 0.89 | ✅ conforms | escalate (low confidence) |
| BR-7 | 一律記成代號 | 分析事件標的為辨識出的代號 | SAS:38 | NIT:42 | asserts-oracle | produces-oracle | asserts-oracle 0.88 | produces-oracle 0.91 | ✅ conforms | accept |
| BR-8 | 重用時間窗以辨識出的代號＋市場類別判斷 | 以代號查重用 | SAS:46 | NIT:130 | asserts-oracle | produces-oracle | asserts-oracle 0.93 | produces-oracle 0.89 | ✅ conforms | accept |
| BR-9 | 台股或加密貨幣名單取不到 | 新聞來源暫時無法使用，不判定找不到 | SRS:74-79 | NIT:107、TrustsOnlyASymbolMatch | asserts-oracle | produces-oracle | asserts-oracle 0.88 | produces-oracle 0.85 | ✅ conforms | accept |
| BR-10 | 美股資料來源暫時無法使用 | 以填的字（大寫）當代號，名稱空 | SRS:63-70 | NIT:63 | asserts-oracle | produces-oracle | asserts-oracle 0.99 | produces-oracle 0.80 | ✅ conforms | recheck |
| BR-11 | 填的字是空白 | 回覆標的為必填 | SRS:59-62 | SAT:216 | asserts-oracle | produces-oracle | asserts-oracle 0.96 | produces-oracle 0.99 | ✅ conforms | accept |
| NFR-1 | 效能 | 台股名錄共用每小時快取、不額外連線；美股至多多一次查詢 | twse_open_data_proxy.go:44-62、SRS:63-72 | twse_symbol_directory_proxy_test.go:45 | asserts-oracle | produces-oracle | asserts-oracle 0.65 | produces-oracle 0.28 | ✅ conforms | escalate (low confidence) |
| NFR-2 | 安全 | 發起分析仍需已啟用 API key | dependencies.go:217-219 | analysis_event_controller_test.go:157 | asserts-oracle | produces-oracle | asserts-oracle 0.83 | produces-oracle 0.51 | ✅ conforms | escalate (low confidence) |
| NFR-3 | 相容 | 以代號發起的結果不變（美股查無改拒絕除外） | SDD:54-60、SRS:54-80 | SAT:90、SAT:480、otc_stock_support_test.go | asserts-oracle | produces-oracle | asserts-oracle 0.29 | diverges 0.33 | 🔴 violation | escalate (violation) |

### Notes

- **第一輪稽核的發現與修正**：
  - AC-26 第一輪 Jev 判 `partial`（no-test）。已補 `TestRunDueTradingSession_GradesATrackedListedEtf`，重跑為 conforms。
  - AC-24 第一輪測試信心 0.28：原本只有以 BTC 驗證時間窗的既有測試。已補 `TestStartSymbolAnalysis_ASymbolStartsAfreshOnceTheNamedAnalysisIsOutOfWindow`，重跑為 conforms／accept。
- **參照範圍修正**：AC-13 / AC-14 第一輪判 unclear / violation，原因是程式碼參照漏了 Domain Model 建構子的排序（SDD:34-36）。AC-25 / AC-26 漏了 `SessionGradeService`；BR-6 的測試資料行數位移；NFR-2 漏了路由群組。補齊後全部重跑，表中為重跑結果。AC-13 另執行對應測試佐證通過。
- **NFR-3（violation，Jev diverges 0.33）**：重新追蹤以代號發起的各條路徑：
  - 台股與加密貨幣的代號都在 `Identify` 第一層以「代號完全相同」命中，回傳的代號、名稱、搜尋字與改版前相同。
  - 美股改為「資料來源可用且查無即拒絕」，是 PRD 明訂的例外。
  - 唯一的實質差異是「當天暫停交易的台股」會暫時查不到，已列在 PRD 第 7 節 Known Risks。
  - 判斷：不是缺陷，維持 Jev 原判並列入 Needs James。
- 突變測試：變動的關鍵規則共 30 個突變（含收斂後 service 的 4 個），全部被測試抓到；變動程式碼覆蓋率 100%（`go test -coverpkg`）。

- **Code review 後的修正**（稽核完成之後）：美股 OTC（`PNK OQB OQX`）列為美國掛牌；名稱空白的美股可以代號辨識；`Identify` 改回傳候選、錯誤改由 service 產生。PRD 業務規則已同步，相關 AC（AC-07、AC-14、AC-19、AC-20）的測試已調整並重跑，所有修正也都做過突變測試。

## Orphans (code with no clause)

| Code | Description | Verdict |
|------|-------------|---------|
| yahoo_finance_proxy.go:138 `toSearchQuery` | 填的字像股票類別代號（如 `BRK.B`）時，改以 Yahoo 的 `BRK-B` 搜尋；起因是實測 `BRK.B` 被誤判成名稱含 BRK.B 的 ETF | undocumented（支援 AC-16 的美股代號辨識；建議補進 PRD 業務規則） |
| coin_gecko_cryptocurrency_proxy.go:49 | 幣種搜尋快取以小寫查詢字為鍵，不同大小寫共用快取 | undocumented（效能細節） |
| SRS:32-51 `ResolveSymbol` | 搜尋標的新聞仍只認代號，名稱會回找不到 | 符合 Out of Scope「搜尋標的新聞以名稱查詢」，**非**違反 |

## Summary

- Conforms: 39/40 clauses ✅（97.5%）
- Violations: NFR-3（Jev 低信心，Claude 判斷不是缺陷，見 Notes）
- Mis-asserted: —
- Partial: —（AC-26 已補測試）
- Gaps: —
- Unclear: —
- Orphans: 3（1 個建議補進規格、1 個效能細節、1 個符合範圍外規定）
- Jev: ran on 40 clauses（被升級的條款於修正參照後重跑）
- Needs James:
  - NFR-3（violation，Jev 0.33；Claude 判斷不是缺陷）
  - 低信心 escalate：AC-01、06、09、14、16、18、20、23、25、BR-2、3、4、6、NFR-1、2。Jev 的測試判斷與 Claude 一致，信心偏低多半是因為比對規則集中在 Domain Model、跨檔追蹤。全部已由突變測試確認測試會抓到對應行為。
