# Product Requirements Document (PRD) — 分析時價格

**Status:** Finalized
**Version:** v1.0
**Owner:** James Hsueh
**Stakeholders:** Engineering

---

## 1. Background & Goal (Why & Goal)

- **Problem Statement:** 專案成功指標是「評等可回測驗證」，但目前分析結果沒有記錄分析當下的價格，日後無法判斷評等方向是否正確。
- **Expected Outcome:**
  - 每個完成的分析結果都嘗試記錄分析時價格（價格、計價幣別、價格時間、價格來源）。
  - 取得的價格保持精確，不因浮點誤差失真。
  - 取不到價格不影響分析完成。
- **Out of Scope:**
  - 回測計算（評等命中率）。
  - 補抓既有分析的價格。
  - 匯率換算。

---

## 2. User Personas

- **Primary Role(s):** 持有已啟用 API key 的使用者；日後的回測功能。
- **Usage Context:** 查詢分析事件時一併取得分析時價格。

---

## 3. User Stories & Acceptance Criteria

### US-01 — 依市場類別記錄分析時價格 [priority: P0]
**As a** 使用者, **I want** 每個分析結果都記錄分析當下的價格, **so that** 我日後能驗證評等是否正確。

```gherkin
Scenario: 加密貨幣記錄 Binance 對 USDT 的最新成交價
  Given BTC 在 Binance 對 USDT 的最新成交價為 86607.62
  When BTC（加密貨幣）的分析完成
  Then 分析時價格為 86607.62，計價幣別 USDT，價格來源「Binance」
  And 價格時間為分析完成的時間

Scenario: 美股記錄 Yahoo 財經的最新市價
  Given AAPL 在 Yahoo 財經的最新市價為 332.94 USD，報價時間為 2026-10-06 22:33:42（UTC）
  When AAPL（美股）的分析完成
  Then 分析時價格為 332.94，計價幣別 USD，價格時間 2026-10-06 22:33:42（UTC），價格來源「Yahoo 財經」

Scenario: 台股記錄證交所最近交易日收盤價
  Given 證交所公布 2330 在 2026-10-05 的收盤價為 2575.00
  When 2330（台股）的分析完成
  Then 分析時價格為 2575.00，計價幣別 TWD，價格時間 2026-10-05（台北時間），價格來源「證交所」

Scenario: 小數位數很多的價格原樣保存
  Given 某幣種在 Binance 的最新成交價為 0.00001234
  When 該幣種的分析完成
  Then 分析時價格為 0.00001234，沒有被四捨五入
```

### US-02 — 取不到價格不影響分析 [priority: P0]
**As a** 使用者, **I want** 價格來源的問題不會讓整個分析失敗, **so that** 我仍能拿到資訊面結論。

```gherkin
Scenario: Binance 沒有該交易對
  Given Binance 沒有 XYZUSDT 交易對
  When XYZ（加密貨幣）的分析完成
  Then 分析事件狀態為「已完成」
  And 分析時價格為「未取得」

Scenario: 價格來源沒有回應
  Given Yahoo 財經沒有回應
  When AAPL（美股）的分析完成
  Then 分析事件狀態為「已完成」
  And 分析時價格為「未取得」

Scenario: 證交所清單沒有該代號
  Given 證交所最近交易日的收盤資料中沒有 9999
  When 9999（台股）的分析完成
  Then 分析事件狀態為「已完成」
  And 分析時價格為「未取得」

Scenario: 價格不是正數
  Given 價格來源回傳的價格為 0
  When 分析完成
  Then 分析時價格為「未取得」
```

### US-03 — 查詢顯示分析時價格 [priority: P0]
**As a** 使用者, **I want** 查詢分析事件時看到分析時價格, **so that** 我可以把評等與價格一起保存。

```gherkin
Scenario: 查詢取得價格的分析
  Given 一筆已完成且分析時價格為 86607.62 USDT 的分析事件
  When 使用者查詢該分析事件
  Then 分析結果顯示價格 86607.62、計價幣別 USDT、價格時間與價格來源

Scenario: 查詢未取得價格的分析
  Given 一筆已完成但分析時價格未取得的分析事件
  When 使用者查詢該分析事件
  Then 分析結果明確顯示分析時價格未取得
```

---

## 4. Business Flow & Logic

- **Flow:** AI 提交結論並通過正規化 → 依市場類別取得價格（最多等待 10 秒）→ 與分析結果一起保存 → 分析事件改為已完成。
- **Core Business Rules:**
  - 價格來源：加密貨幣 → Binance（標的 + USDT 交易對）；美股 → Yahoo 財經；台股 → 證交所每日收盤行情。
  - 價格時間：Binance 為取得價格的時間；Yahoo 財經為來源提供的報價時間；證交所為收盤資料日期（台北時間當日 00:00）。
  - 價格必須大於 0，否則視為未取得。
  - 價格以來源提供的十進位字串保存，不經浮點數轉換。
  - 證交所每日收盤資料保留 1 小時後重新取得；重新取得失敗時沿用上一次成功取得的資料。
  - 美股代號中的「.」以「-」向 Yahoo 財經查詢（例如 BRK.B → BRK-B）；報價沒有計價幣別視為未取得。
  - 取價有獨立的 10 秒上限，不受分析剩餘時間影響。
- **Edge Cases:**
  - 價格來源回傳格式無法解析 → 未取得。
  - 取價格時發生任何錯誤 → 未取得，分析照常完成。

---

## 5. UI/UX Design & Interaction

N/A — 無圖形介面。

---

## 6. Non-Functional Requirements

- **Performance:** 取價格最多增加 10 秒的分析時間。
- **Accuracy:** 價格以精確小數保存。

---

## 7. Dependencies & Risks

- **External Dependencies:** Binance 公開行情、Yahoo 財經報價（非正式公開服務）、證交所開放資料。
- **Known Risks:** 台股價格為最近交易日收盤，盤中分析時與即時價有落差；Yahoo 財經格式可能無預警變更。

---

## 8. Appendix

- `.sdd/2026-10-06-analysis-price-snapshot/BRIEF.md`
- `.sdd/2026-10-06-symbol-news-analysis/PRD.md`
