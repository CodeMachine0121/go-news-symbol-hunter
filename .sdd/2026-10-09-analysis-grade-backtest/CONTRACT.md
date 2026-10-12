# Contract Traceability Matrix — 分析評等回測 (analysis-grade-backtest)

Contract: PRD.md (§3 Gherkin AC, §4 Core Business Rules + Edge Cases, §6 NFR) — working-tree version (uncommitted spec edits included)
Design map: ARCH.md (working-tree version)
Implementation: branch `feature/analysis-grade-backtest` @ 6d96b64 (diff vs `main`)
Oracle: Acceptance Criteria (56 clauses: 39 AC · 14 BR · 3 NFR)

> Static conformance audit, run 2 (supersedes run 1); BR-10 re-judged at ab08782 against the updated PRD wording. Clause IDs were re-extracted from the current PRD, so the IDs differ from run 1. Oracles were derived from the PRD before any code or test was opened. As a cross-check, I ran the mapped domain, application, controller and persistence tests once (`-p 1`, `TEST_POSTGRES_DSN` → `go_symbol_news_hunter_test`), and all passed. Verdicts come from comparing each test and code path with the expected outcome, not from whether the tests pass.
>
> Path abbreviations: `ABD` = `internal/domain/models/domains/analysis_backtest_domain.go`; `SUM` = `internal/domain/models/domains/grade_backtest_summary_domain.go`; `SVC` = `internal/domain/service/grade_backtest_service.go`; `ERR` = `internal/domain/service/grade_backtest_errors.go`; `PER` = `internal/domain/models/vo/analysis_completed_period_vo.go`; `SFV` = `internal/domain/models/vo/symbol_filter_vo.go`; `REPO` = `internal/infrastructure/persistence/analysis_result_repository.go`; `CTL` = `internal/controller/grade_backtest_controller.go`; `ERT` = `internal/controller/error_response_table.go`; `PXY` = `internal/infrastructure/gotrading/go_trading_price_history_proxy.go`; `DEP` = `cmd/server/dependencies.go`.
> Tests: `ABDt` = `internal/domain/models/domains/tests/analysis_backtest_domain_test.go`; `SUMt` = `.../domains/tests/grade_backtest_summary_domain_test.go`; `APPt` = `internal/application/tests/grade_backtest_application_test.go`; `CTLt` = `internal/controller/tests/grade_backtest_controller_test.go`; `PERt` = `internal/domain/models/vo/tests/analysis_completed_period_vo_test.go`; `SFVt` = `.../vo/tests/symbol_filter_vo_test.go`; `REPOt` = `internal/infrastructure/persistence/tests/analysis_result_backtest_repository_test.go`; `PXYt` = `internal/infrastructure/gotrading/tests/go_trading_price_history_proxy_test.go`; `DEPt` = `cmd/server/dependencies_test.go`.

## Clauses

The `Spec-expected` column holds the business-observable oracle from Phase 2; the
concrete artifact it bridges to (via UL-MAP/ARCH) is what the audit columns check.

| ID | Clause | Spec-expected (oracle) | Impl | Test | Test audit | Code audit | Status |
|----|--------|------------------------|------|------|------------|------------|--------|
| AC-01 | 看多且之後上漲為命中 | 看多短期 100 → 第 10 交易日 105 → 命中，+5% | ABD:82-107, ABD:148 | ABDt:61 (case "bullish and rose") | asserts-oracle | produces-oracle | ✅ conforms |
| AC-02 | 看空且之後下跌為命中 | 看空中期 100 → 第 60 交易日 92 → 命中，−8% | ABD:117-122 | ABDt:61 ("bearish and fell") | asserts-oracle | produces-oracle | ✅ conforms |
| AC-03 | 強烈看多但之後下跌為沒命中 | 100 → 97 → 沒命中，−3% | ABD:94-99 | ABDt:61 ("strong bullish but fell") | asserts-oracle | produces-oracle | ✅ conforms |
| AC-04 | 看多但價格沒變為沒命中 | 100 → 100 → 沒命中，0% | ABD:94-99 | ABDt:61 ("bullish but unchanged") | asserts-oracle | produces-oracle | ✅ conforms |
| AC-05 | 中性評等不計入命中率 | 中性 100 → 103 → 中性 +3%；不進命中率，只進中性平均 | ABD:93, SUM:47-52, SUM:85-88 | ABDt:61 ("neutral is not judged"); SUMt:69 | asserts-oracle | produces-oracle | ✅ conforms |
| AC-06 | 交易時段內完成的台股分析 | 週三 10:15 完成 → 起點＝該分鐘 1000 | ABD:65-74 | ABDt:127 (neighbours 990/1020) | asserts-oracle | produces-oracle | ✅ conforms |
| AC-07 | 收盤後完成的台股分析 | 週三 20:30 → 起點＝收盤前最後一筆 1010 | ABD:56, ABD:65-74 | ABDt:127 | asserts-oracle | produces-oracle | ✅ conforms |
| AC-08 | 週末完成的加密貨幣分析 | 週六 03:00 → 起點 86000 | ABD:65-74 | ABDt:127 | asserts-oracle | produces-oracle | ✅ conforms |
| AC-09 | 開盤前完成的台股分析，完成當天即為第 1 個交易日 | 週四 07:30 → 起點 1010，週四為第 1 交易日 | ABD:57, ABD:65-74, ABD:80-86; SVC:72-78 | ABDt:127, ABDt:182; APPt:224 | asserts-oracle | produces-oracle | ✅ conforms |
| AC-10 | 交易時段內完成的台股分析，完成當天不算交易日 | 週三 10:15 → 週四為第 1 交易日 | ABD:80-86 | ABDt:182 | asserts-oracle | produces-oracle | ✅ conforms |
| AC-11 | 分析時價格未取得仍可評估 | 分析時價格 null → 仍判命中/沒命中 | ABD:63-110 (never reads `Price`) | ABDt:258 | asserts-oracle | produces-oracle | ✅ conforms |
| AC-12 | 檢查點那天還在交易 | 第 10 交易日＝今天且未收盤 → 尚未到期，不以盤中價判斷 | ABD:86-90, SVC:55, SVC:83 | ABDt:345 (23:59 UTC → pending); APPt:313 (crypto, 12:00 UTC on checkpoint day) | asserts-oracle | produces-oracle | ✅ conforms |
| AC-13 | 沒有每日價格的日子不算交易日 | 中間一天無日價 → 檢查點往後延一天 | ABD:82-92 (counts candles) | ABDt:369 (gap on 10-10 → checkpoint 10-18) | asserts-oracle | produces-oracle | ✅ conforms |
| AC-14 | 時間範圍尚未走完 | 只過 7 交易日 → 尚未到期；計入尚未到期筆數 | ABD:109, SUM:46,51 | ABDt:214; SUMt:83 | asserts-oracle | produces-oracle | ✅ conforms |
| AC-15 | 美股無法評估 | AAPL → 無法評估；計入無法評估筆數 | SVC:61-64, ABD:113-115, DEP:146-148 | APPt:199; DEPt:174 | asserts-oracle | produces-oracle | ✅ conforms |
| AC-16 | 起點價格不是正數 | 起點價格 0 → 無法評估 | ABD:75-76 | ABDt:229 ("the entry price is not positive") | asserts-oracle | produces-oracle | ✅ conforms |
| AC-17 | 一筆無法評估不影響其他分析 | 6488 無法評估、2330 命中（同一次查詢） | SVC:59-86 | APPt:331 | asserts-oracle | produces-oracle | ✅ conforms |
| AC-18 | 價格服務沒有登錄的代號無法評估 | 6488 未登錄 → 無法評估 | ABD:75-76; PXY:89-93 | APPt:211; PXYt:126 | asserts-oracle | produces-oracle | ✅ conforms |
| AC-19 | 失敗的分析不出現 | 失敗事件不在明細、不在任何筆數 | REPO:39-45 (subquery on `analysis_events.status = succeeded`) | REPOt:96 (succeeded/failed/running → only succeeded; ran green on Postgres) | asserts-oracle | produces-oracle | ✅ conforms |
| AC-20 | 整體命中率與隨機比較 | 40 筆 26 命中 → 65%，並列 50% | SUM:63-67, SUM:77-78 | SUMt:24 | asserts-oracle | produces-oracle | ✅ conforms |
| AC-21 | 依市場類別、評等、信心分組 | 各分組列命中率與筆數 | SUM:33-38, SUM:55-61, SUM:69-82 | SUMt:33; CTLt:67-77 | asserts-oracle | produces-oracle | ✅ conforms |
| AC-22 | 信心 75 歸入高 | 75 → 高 | ABD:126-131 | ABDt:270 | asserts-oracle | produces-oracle | ✅ conforms |
| AC-23 | 信心 74 歸入中 | 74 → 中 | ABD:126-131 | ABDt:270 | asserts-oracle | produces-oracle | ✅ conforms |
| AC-24 | 沒有可評估分析的分組 | 「低」命中率＝沒有資料，非 0% | SUM:63-67 (nil → JSON `null`) | SUMt:60; CTLt:70-72 | asserts-oracle | produces-oracle | ✅ conforms |
| AC-25 | 指定分析完成期間 | 只含 10 月完成者 | PER:22-41, REPO:46-51 | PERt:12; APPt:106; REPOt:40 | asserts-oracle | produces-oracle | ✅ conforms |
| AC-26 | 不指定期間 | 含全部已完成分析 | PER:23, REPO:46-51 | REPOt:40 ("everything"); APPt:106 | asserts-oracle | produces-oracle | ✅ conforms |
| AC-27 | 指定標的 | 2330 + 台股 → 只含 2330 | SFV:14-29, REPO:52-55 | REPOt:40 ("one symbol in one market") | asserts-oracle | produces-oracle | ✅ conforms |
| AC-28 | 標的寫法與分析時一致 | 「 btc 」+ 加密貨幣 → 含 BTC | SFV:25-29 | SFVt:10; APPt:106 | asserts-oracle | produces-oracle | ✅ conforms |
| AC-29 | 只給市場類別沒給標的 | 拒絕，「標的與市場類別需一起提供」 | SFV:18-20, CTL:15 | SFVt:10 ("category without symbol"); APPt:145 | asserts-oracle | produces-oracle | ✅ conforms |
| AC-30 | 標的空白 | 拒絕，「標的為必填」 | SFV:25-28, CTL:17 | CTLt:93; APPt:145 | asserts-oracle | produces-oracle | ✅ conforms |
| AC-31 | 只給標的沒給市場類別 | 拒絕，「標的與市場類別需一起提供」 | SFV:18-20, CTL:15 | CTLt:91 | asserts-oracle | produces-oracle | ✅ conforms |
| AC-32 | 不支援的市場類別 | futures → 拒絕，說明市場類別不支援 | SFV:21-24, CTL:16 | CTLt:92 | asserts-oracle | produces-oracle | ✅ conforms |
| AC-33 | 逐筆明細 | 列出標的、市場類別、評等、信心、時間範圍、起點 100、檢查點 105、+5%、命中 | ABD:124-153 | ABDt:99; CTLt:75-76 | asserts-oracle | produces-oracle | ✅ conforms |
| AC-34 | 涵蓋分析超過 500 筆 | 501 → 拒絕，「涵蓋分析超過 500 筆，請縮小期間」 | SVC:48-54, ERR:11, CTL:14 | APPt:169; CTLt:94-96 | asserts-oracle | produces-oracle | ✅ conforms |
| AC-35 | 涵蓋分析剛好 500 筆 | 500 → 正常回傳 500 筆明細 | SVC:48-54 | APPt:169 | asserts-oracle | produces-oracle | ✅ conforms |
| AC-36 | 指定的期間不合理 | 起日晚於迄日或格式錯 → 拒絕，「期間不正確」 | PER:24-40, CTL:13 | PERt:38; APPt:145; CTLt:90 | asserts-oracle | produces-oracle | ✅ conforms |
| AC-37 | 分析資料讀取失敗 | 查詢失敗，「服務暫時無法使用」 | SVC:49-51, ERT:25 | APPt:278; CTLt:101-103 | asserts-oracle | produces-oracle | ✅ conforms |
| AC-38 | 價格服務無法連線 | 查詢失敗，「價格服務暫時無法使用」，不回傳無法評估結果 | SVC:89-91, PXY:71-74, CTL:18 | APPt:251; CTLt:97-100; PXYt:166-200 | asserts-oracle | produces-oracle | ✅ conforms |
| AC-39 | 未啟用的 API key 不能查詢 | 拒絕，說明 API key 尚未啟用 | DEP:217 (protected group) | CTLt:105; DEPt:55 | asserts-oracle | produces-oracle | ✅ conforms |
| BR-01 | 起點價格：完成那一分鐘的價格；不在交易中用完成前最後一筆 | 如左 | ABD:56-57, ABD:65-74; SVC:72-78 | ABDt:127; APPt:224, APPt:238 | asserts-oracle | produces-oracle | ✅ conforms |
| BR-02 | 檢查點：短期第 10、中期第 60 交易日收盤價；第 1 交易日定義 | 如左 | ABD:80-107, ABD:117-122 | ABDt:61, ABDt:182 | asserts-oracle | produces-oracle | ✅ conforms |
| BR-03 | 交易日：價格服務有當天每日價格的日子；加密貨幣每天 | 無日價的日子不算 | ABD:82-92 | ABDt:369 | asserts-oracle | produces-oracle | ✅ conforms |
| BR-04 | 漲跌幅 = (檢查點收盤價 − 起點價格) ÷ 起點價格 | 如左 | ABD:148 | ABDt:61, ABDt:318 | asserts-oracle | produces-oracle | ✅ conforms |
| BR-05 | 看多類 >0 命中；看空類 <0 命中；中性不判斷 | 如左（=0 為沒命中） | ABD:93-105 | ABDt:61 (all grades, both edges) | asserts-oracle | produces-oracle | ✅ conforms |
| BR-06 | 命中率 = 命中 ÷（命中 + 沒命中）；分母 0 為沒有資料 | 如左 | SUM:63-67 | SUMt:24, SUMt:60 | asserts-oracle | produces-oracle | ✅ conforms |
| BR-07 | 信心分組 0–49 低、50–74 中、75–100 高 | 如左 | ABD:126-131 | ABDt:270 (0/49/50/74/75/100) | asserts-oracle | produces-oracle | ✅ conforms |
| BR-08 | 價格一律以精確小數計算 | 價格與漲跌幅無浮點誤差 | ABD:148, `go_trading_wire.go:24`, SUM:86 | PXYt:53 (`"1000.5"`); ABDt:318 | asserts-oracle | produces-oracle | ✅ conforms |
| BR-09 | 漲跌幅、中性平均漲跌幅與命中率四捨五入到小數 4 位 | 三者皆 4 位 | ABD:148, SUM:65, SUM:86 | ABDt:318 (0.3333); SUMt:69 (0.0213); SUMt:33 (0.3333) | asserts-oracle | produces-oracle | ✅ conforms |
| BR-10 | 某一天還在交易（尚未收盤）時，那一天不算已走完的交易日。台股以台北時間 13:30 收盤為準；加密貨幣全天交易，以世界標準時間當日結束為準。 | 台股：當天 13:30 台北前不算、13:30 起算；加密貨幣：UTC 當日 24:00 前不算、24:00 起算 | ABD:34, ABD:84-87, ABD:95 (`candleDay + closeIntoDay` after `evaluatedAt` → stop; equal counts) | ABDt:345 (twStock 13:29 → pending, 13:30 → hit; crypto 23:59 UTC → pending, 24:00 → hit) | asserts-oracle | produces-oracle | ✅ conforms |
| BR-11 | 只回測狀態為已完成的分析事件；失敗或分析中的事件即使有結果也不列入 | 如左 | REPO:39-45 | REPOt:96 | asserts-oracle | produces-oracle | ✅ conforms |
| BR-12 | 單一標的查不到 → 該筆無法評估，其餘照算 | 如左 | SVC:59-86, ABD:75-76 | APPt:331 | asserts-oracle | produces-oracle | ✅ conforms |
| BR-13 | 價格服務整個無法連線 → 整次查詢失敗 | 如左 | SVC:87-91 | APPt:251; CTLt:97-100 | asserts-oracle | produces-oracle | ✅ conforms |
| BR-14 | 期間內沒有任何分析 → 空總表，命中率皆沒有資料 | 空明細；整體與每一分組命中率皆沒有資料 | SUM:33-38, SUM:63-82 | SUMt:97 (now loops every group, asserts nil) | asserts-oracle | produces-oracle | ✅ conforms |
| NFR-01 | 一次最多 500 筆；當場計算，不保存 | ≤500；無任何寫入 | SVC:48-54; no write path | APPt:169 (strict mocks reject any unexpected write) | asserts-oracle | produces-oracle | ✅ conforms |
| NFR-02 | 需要已啟用且未撤銷的 API key | 無有效 key 被拒 | DEP:217 → `RequireActiveApiKey` | DEPt:55; CTLt:105 | asserts-oracle | produces-oracle | ✅ conforms |
| NFR-03 | 價格服務登入憑證不得出現在任何回應中 | 回應不含帳密/憑證 | ERT:21,25 (fixed messages); PXY:70 (token only in header) | CTLt:97-100 (exact message) | asserts-oracle | produces-oracle | ✅ conforms |

## Orphans (code with no clause)

Out-of-Scope checklist (session/combined grade backtest, US prices, auto-registering symbols / backfill, persisting or scheduling results, fixed 1/5/20-day horizons): no matches.

All five orphans from run 1 are now covered by AC-16, AC-29, AC-30, AC-37 and BR-09. I found no new behaviour that lacks a clause.

| Code | Description | Verdict |
|------|-------------|---------|
| — | — | none |

## Summary

- Conforms: 56/56 clauses ✅ (100%)
- Violations: none  (BR-10 fixed in ab08782: Taiwan days close at 13:30 Taipei, crypto at the end of the UTC day)
- Mis-asserted: none
- Partial: none
- Gaps: none
- Unclear: none
- Orphans: 0
