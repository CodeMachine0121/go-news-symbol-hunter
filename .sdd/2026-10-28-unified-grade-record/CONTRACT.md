# Contract Traceability Matrix — 統一評等紀錄（後端）

Contract: PRD.md  
Design map: ARCH.md  
Implementation: `go-symbol-news-hunter`，分支 `feature/unified-grade-record`  
Oracle: Acceptance Criteria（24）+ Core Business Rules（5）+ NFR（2）= 31 clauses

> 靜態一致性稽核：以 PRD 預期結果為準，分別判斷測試是否斷言該結果、程式是否產生該結果；不執行自創情境。另於本機以正式資料庫的副本實際啟動一次，確認 9 筆分析結果與 20 筆綜合評等正確轉換（佐證，不作為判定依據）。

## Clauses

`Test audit` / `Code audit` 為 Claude 的 shadow verdict；`Jev test` / `Jev code`、`Status`、`Route` 照 `jev_audit.py` 回傳原樣記錄。

| ID | Clause | Spec-expected (oracle) | Test audit | Code audit | Jev test | Jev code | Status | Route | Notes |
|----|--------|------------------------|------------|------------|----------|----------|--------|-------|-------|
| AC-1 | US-01 當天第一次產生綜合評等 | 2330 gains one background grade record whose trading day is Wednesday | asserts-oracle | produces-oracle | asserts-oracle 0.92 | produces-oracle 0.59 | ✅ conforms | escalate (low confidence) |  |
| AC-2 | US-01 同一天再次重算不覆蓋先前的紀錄 | Wednesday has two background records; the pre-market one keeps its grade and creation time | asserts-oracle | produces-oracle | asserts-oracle 0.65 | produces-oracle 0.56 | ✅ conforms | escalate (low confidence) |  |
| AC-3 | US-01 時段評等失敗時不新增背景評等紀錄 | the number of 2330 background records on Wednesday is unchanged when its intraday analysis fails | asserts-oracle | produces-oracle | asserts-oracle 0.79 | produces-oracle 0.92 | ✅ conforms | recheck (mid confidence) | 第一輪：🟠 mis-asserted（Jev test shallow 0.25 / code produces-oracle 0.84）→ 已補測試後重跑；第一輪 shadow 判 asserts-oracle 有誤：測試收集了背景紀錄卻沒斷言；已補斷言（失敗標的不新增背景紀錄）。 |
| AC-4 | US-01 盤前只刪除前一天的時段評等，背景評等紀錄保留 | Tuesday's session grades are deleted and Tuesday's background records remain | asserts-oracle | produces-oracle | mis-asserted 0.53 | unclear 0.08 | ❔ unclear | escalate (disagree) | 第一輪：🔴 violation（Jev test mis-asserted 0.45 / code diverges 0.25）→ 已補測試後重跑；背景紀錄不可能被刪：介面 `IGradeRecordRepository` 根本沒有刪除方法，盤前只呼叫 `sessionGradeRepository.DeleteExceptTradingDay`（`session_grade_service.go:59`）。測試以「grade record repository 只收到 Create」斷言；Jev 要的是「週二紀錄仍在資料庫」的整合斷言。 |
| AC-5 | US-02 背景評等紀錄帶入追蹤標的登記的名稱 | the record's symbol name is 台積電, source background, time horizon short | asserts-oracle | produces-oracle | asserts-oracle 0.40 | produces-oracle 0.87 | ✅ conforms | escalate (low confidence) |  |
| AC-6 | US-02 追蹤標的沒有登記名稱 | the record's symbol name is empty | asserts-oracle | produces-oracle | asserts-oracle 0.99 | produces-oracle 0.48 | ✅ conforms | escalate (low confidence) |  |
| AC-7 | US-02 背景評等紀錄不帶理由、佐證與分析時價格 | the record has no reason, no evidence and its price at analysis is not obtained | asserts-oracle | produces-oracle | asserts-oracle 0.96 | produces-oracle 0.69 | ✅ conforms | recheck (mid confidence) |  |
| AC-8 | US-03 同一天有多筆時取最新一筆 | 2330 appears once and its combined grade is neutral | asserts-oracle | produces-oracle | asserts-oracle 0.96 | produces-oracle 0.51 | ✅ conforms | escalate (low confidence) |  |
| AC-9 | US-03 有多天紀錄時只取最新交易日 | 2330 appears once with trading day Wednesday and the newest Wednesday grade | asserts-oracle | produces-oracle | asserts-oracle 0.73 | produces-oracle 0.93 | ✅ conforms | recheck (mid confidence) |  |
| AC-10 | US-03 已暫停追蹤的標的不出現 | 2330 is not in the result when paused | asserts-oracle | produces-oracle | asserts-oracle 0.39 | produces-oracle 0.55 | ✅ conforms | escalate (low confidence) |  |
| AC-11 | US-04 背景評等紀錄命中 | the background record's backtest outcome is hit and its source is shown as background | asserts-oracle | produces-oracle | asserts-oracle 1.00 | produces-oracle 0.64 | ✅ conforms | recheck (mid confidence) |  |
| AC-12 | US-04 台股開盤前建立的背景評等紀錄 | entry price is Wednesday's last price before close and Thursday counts as trading day 1 | asserts-oracle | produces-oracle | asserts-oracle 0.88 | produces-oracle 0.56 | ✅ conforms | escalate (low confidence) |  |
| AC-13 | US-04 同一天的多筆背景評等紀錄各自回測 | the detail lists all 3 records, each with an entry price from its own creation time | asserts-oracle | produces-oracle | asserts-oracle 1.00 | produces-oracle 0.84 | ✅ conforms | recheck (mid confidence) | 第一輪：🔴 violation（Jev test asserts-oracle 0.89 / code diverges 0.57）→ 已補測試後重跑；已補 application 層測試：同日三筆背景紀錄各自取 90 / 110 / 120 起點價格。 |
| AC-14 | US-04 未完成的手動分析不回測 | a manual record whose analysis event is not succeeded is absent from the detail | asserts-oracle | produces-oracle | asserts-oracle 0.85 | produces-oracle 0.81 | ✅ conforms | recheck (mid confidence) |  |
| AC-15 | US-04 背景評等紀錄尚未到期 | the background record's outcome is pending and not counted in the hit rate | asserts-oracle | produces-oracle | asserts-oracle 0.59 | produces-oracle 0.99 | ✅ conforms | escalate (low confidence) |  |
| AC-16 | US-05 依來源分組顯示命中率 | by source: manual 60%, background 45%; overall 50% | asserts-oracle | produces-oracle | asserts-oracle 0.99 | unclear 0.49 | ❔ unclear | escalate (disagree) | 第一輪：❔ unclear（Jev test asserts-oracle 0.96 / code unclear 0.37）→ 已補測試後重跑；跑了對應測試 `TestGetGradeBacktest_ComparesManualAndBackgroundHitRates`：綠。程式以各分組獨立計數（`grade_backtest_summary_domain.go:39`），手動 6/10、背景 9/20、整體 15/30。 |
| AC-17 | US-05 只看背景來源 | only the 20 background records count; overall 45% | asserts-oracle | produces-oracle | asserts-oracle 0.96 | produces-oracle 0.55 | ✅ conforms | escalate (low confidence) |  |
| AC-18 | US-05 沒有背景評等紀錄時顯示沒有資料 | background hit rate shows no data, not 0% | asserts-oracle | produces-oracle | asserts-oracle 0.77 | produces-oracle 0.55 | ✅ conforms | escalate (low confidence) |  |
| AC-19 | US-05 指定不存在的來源 | the query is rejected saying 來源不正確 | asserts-oracle | produces-oracle | asserts-oracle 0.99 | produces-oracle 0.57 | ✅ conforms | escalate (low confidence) |  |
| AC-20 | US-05 兩種來源合計超過上限 | the query is rejected and the user is asked to narrow the period | asserts-oracle | produces-oracle | asserts-oracle 0.77 | produces-oracle 0.55 | ✅ conforms | escalate (low confidence) | 第一輪：❔ unclear（Jev test asserts-oracle 0.95 / code unclear 0.56）→ 已補測試後重跑；已補兩種來源混合 300 + 201 筆的測試。 |
| AC-21 | US-06 既有分析結果轉成手動評等紀錄 | 9 manual records with the same grade, reason, price at analysis and creation time | asserts-oracle | produces-oracle | asserts-oracle 0.47 | produces-oracle 0.61 | ✅ conforms | escalate (low confidence) |  |
| AC-22 | US-06 既有綜合評等轉成背景評等紀錄 | one 2330 background record created at Wednesday 20:10 with a short time horizon | asserts-oracle | produces-oracle | asserts-oracle 0.95 | produces-oracle 0.52 | ✅ conforms | escalate (low confidence) |  |
| AC-23 | US-06 不重複轉換 | the number of grade records is unchanged after a second start | asserts-oracle | produces-oracle | asserts-oracle 0.96 | produces-oracle 0.73 | ✅ conforms | recheck (mid confidence) |  |
| AC-24 | US-06 轉換後查詢分析事件結果不變 | the analysis event's result after the move equals the result before it | asserts-oracle | produces-oracle | asserts-oracle 0.57 | produces-oracle 0.16 | ✅ conforms | escalate (low confidence) |  |
| BR-1 | 評等紀錄共通內容 | 每筆評等紀錄都有來源、標的、市場類別、標的名稱、評等、信心指數、時間範圍、建立時間 | asserts-oracle | produces-oracle | asserts-oracle 0.75 | produces-oracle 0.61 | ✅ conforms | recheck (mid confidence) |  |
| BR-2 | 手動評等紀錄另有所屬分析事件、理由、佐證、價格；一個分析事件最多一筆 | 手動評等紀錄保留所屬分析事件、理由、關鍵事件、風險因子、佐證、分析時價格；同一分析事件不會有兩筆 | asserts-oracle | produces-oracle | asserts-oracle 0.63 | produces-oracle 0.25 | ✅ conforms | escalate (low confidence) |  |
| BR-3 | 綜合評等計算方式不變，改為每次新增一筆背景評等紀錄 | 綜合評等算法不變，每次重算新增一筆 | asserts-oracle | produces-oracle | no-test 0.59 | produces-oracle 0.52 | 🟡 partial | escalate (disagree) | Jev 引用的測試範圍只涵蓋計算；「每次新增一筆」由 AC-1 / AC-2 的測試涵蓋。 |
| BR-4 | 命中率總表依來源分組並在明細標出來源 | 依來源分組恆列手動與背景；明細標來源，手動附分析事件編號、背景附交易日 | asserts-oracle | produces-oracle | asserts-oracle 0.91 | produces-oracle 0.49 | ✅ conforms | escalate (low confidence) |  |
| BR-5 | 轉換完成後舊資料不再分開存在 | 轉換後只剩評等紀錄 | asserts-oracle | produces-oracle | asserts-oracle 0.89 | produces-oracle 0.84 | ✅ conforms | recheck (mid confidence) |  |
| NFR-1 | 查詢追蹤標的評等不隨背景評等紀錄累積而變慢 | 查詢量只隨追蹤標的數成長 | asserts-oracle | produces-oracle | no-test 0.61 | produces-oracle 0.59 | 🟡 partial | escalate (disagree) | 沒有效能測試；設計上每個追蹤標的一次走索引的查詢，查詢量與紀錄累積無關（`grade_record_repository.go:42`）。 |
| NFR-2 | 查詢分析事件與追蹤標的評等回傳內容不變 | 兩個查詢的對外內容與先前相同 | asserts-oracle | produces-oracle | asserts-oracle 0.88 | produces-oracle 0.83 | ✅ conforms | recheck (mid confidence) |  |

## Orphans (code with no clause)

| Code | Description | Verdict |
|------|-------------|---------|
| `grade_backtest_summary_domain.go:39` | 各分組改為獨立計數，未列出的分組值（例如沒有來源的資料）不會被計入 | undocumented：防止分組值互撞的內部修正，不改變任何對外結果 |
| `grade_backtest_errors.go` | 「涵蓋分析超過 500 筆」改為「涵蓋評等紀錄超過 500 筆」 | 對應 PRD US-05 上限情境的用語，錯誤碼不變 |

## Summary

- Conforms: 27/31 clauses ✅ (87%)
- Violations: 無（第一輪 AC-4、AC-13 被判 violation，皆為 Jev 低信心；AC-13 補測試後 conforms，AC-4 見 Needs James）
- Mis-asserted: 無（第一輪 AC-3 已修正）
- Partial: BR-3, NFR-1
- Gaps: 無
- Unclear: AC-4, AC-16
- Orphans: 2（皆非 Out of Scope 項目）
- Jev: ran on 31 clauses（AC-3、AC-4、AC-13、AC-16、AC-20 補測試後重跑）
- Needs James: AC-1（low confidence）, AC-2（low confidence）, AC-4（disagree）, AC-5（low confidence）, AC-6（low confidence）, AC-8（low confidence）, AC-10（low confidence）, AC-12（low confidence）, AC-15（low confidence）, AC-16（disagree）, AC-17（low confidence）, AC-18（low confidence）, AC-19（low confidence）, AC-20（low confidence）, AC-21（low confidence）, AC-22（low confidence）, AC-24（low confidence）, BR-2（low confidence）, BR-3（disagree）, BR-4（low confidence）, NFR-1（disagree）
  - 大多為 Jev「low confidence」而雙方結論一致（conforms）；真正分歧只有 AC-4、AC-16、BR-3、NFR-1，理由見各列 Notes。
- 另：`TestStartSymbolAnalysis_RejectsWhenAnalysisCapacityIsReached` 在本機偶發失敗（背景 goroutine 時序競態），main 上同一測試未變更、本分支未改動相關程式。
